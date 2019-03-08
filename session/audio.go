package session

import (
	"bufio"
	"bytes"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"io"
	"os/exec"
	"strconv"
	"strings"

	"github.com/bwmarrin/discordgo"
	"github.com/layeh/gopus"
)

const (
	channels  int = 2
	frameRate int = 48000
	frameSize int = 960
	maxBytes  int = (frameSize * 2) * 2
)

//Audio constructs the currently playing audio
type Audio struct {
	currentlyPlaying string
	isPlaying        bool
	bufferQueue      map[string]*bufio.Reader
	queue            []*Song
	Paused           bool
	Skip             bool
}

//Song ...
type Song struct {
	title     string
	duration  string
	webURL    string
	fileURL   string
	requestor string
}

//CreateAudio ...
func (session Session) CreateAudio() *Audio {
	return &Audio{bufferQueue: make(map[string]*bufio.Reader)}
}

func getYoutubeData(input string, user string) Song {
	//call youtube-dl and assign the values into a Song struct
	cmd := exec.Command("youtube-dl", "--print-json", "--flat-playlist", "-f", "bestaudio", "--skip-download", "--default-search", "ytsearch", input)
	var out bytes.Buffer
	cmd.Stdout = &out
	err := cmd.Run()
	if err != nil {
		fmt.Println("error gathering youtube data:", err)
	}
	str := out.String()
	song := make(map[string]string)
	_ = json.Unmarshal([]byte(str), &song)

	return Song{title: song["title"], duration: song["duration"], webURL: song["webpage_url"], fileURL: song["url"], requestor: user}
}

//AddToQueue ...
func (session Session) AddToQueue(discord *discordgo.Session, input string, user string) (err error) {

	song := getYoutubeData(input, user)
	//playback doesn't happen properly with search links so we have to do another getYoutubeData if it wasn't originally a link using
	//the link we got from the first call
	if !strings.Contains(input, "https://") {
		vidURL := "https://www.youtube.com/watch?v=" + song.fileURL
		song = getYoutubeData(vidURL, user)
	}
	//queueing the same song twice while it is playing causes it to restart so we skip
	if session.AudioManager.currentlyPlaying == song.title {
		return
	}
	ffmpeg := exec.Command("ffmpeg", "-reconnect", "1", "-reconnect_streamed", "1", "-reconnect_delay_max", "2", "-i", song.fileURL, "-f", "s16le", "-ar", strconv.Itoa(frameRate), "-ac", strconv.Itoa(channels), "pipe:1")

	audioData, err := ffmpeg.StdoutPipe()
	if err != nil {
		fmt.Println("whoops there was an error creating ffmpeg pipe:", err)
		return err
	}

	if len(session.AudioManager.queue) < 1 {
		session.AudioManager.queue = make([]*Song, 0)
	}

	session.AudioManager.bufferQueue[song.title] = bufio.NewReaderSize(audioData, 16384)
	err = ffmpeg.Start()

	if err != nil {
		fmt.Println("there was an error in the ffmpeg command error", err)
	}
	session.lock.Lock()
	session.AudioManager.queue = append(session.AudioManager.queue, &song)
	session.lock.Unlock()

	_, _ = discord.ChannelMessageSend(session.textChannel, "```Queued: "+song.title+"\nRequested by: "+user+"```")

	return nil
}

//SkipSong moves the song queue forward and delets the entry from the AudioBuffer's bufferQueue
func (session Session) SkipSong(discord *discordgo.Session) {

	session.AudioManager.Skip = true
}

func (session Session) updateUsers(discord *discordgo.Session) {
	if len(session.AudioManager.queue) > 0 {

		_, _ = discord.ChannelMessageSend(session.textChannel, "```Now Playing:\n"+session.AudioManager.currentlyPlaying+
			"\nAt URL: "+session.AudioManager.queue[0].webURL+"\nRequested by: "+session.AudioManager.queue[0].requestor+"```")
		discord.UpdateStatus(0, session.AudioManager.currentlyPlaying)
		return
	}
	discord.UpdateStatus(0, "nothing, the queue is empty")
	session.AudioManager.currentlyPlaying = "abosulutely nothing: N/A"
}

//Stop empty the playing queue
func (session Session) Stop(discord *discordgo.Session) {
	discord.UpdateStatus(0, "")
	session.AudioManager.currentlyPlaying = "abosulutely nothing: N/A"
	session.LeaveChannel(session.voiceChannel)
}

//PlayQueue ...
func (session Session) PlayQueue(discord *discordgo.Session) (err error) {
	if session.AudioManager.isPlaying {
		return
	}
	session.AudioManager.isPlaying = true

	for len(session.AudioManager.queue) > 0 {
		session.AudioManager.currentlyPlaying = session.AudioManager.queue[0].title
		session.updateUsers(discord)
		session.playSound(discord, session.AudioManager.bufferQueue[session.AudioManager.currentlyPlaying])
		session.lock.Lock()
		delete(session.AudioManager.bufferQueue, session.AudioManager.queue[0].title)
		session.AudioManager.queue = session.AudioManager.queue[1:]
		if session.AudioManager.Skip {
			session.AudioManager.Skip = false
		}
		session.lock.Unlock()
	}

	session.AudioManager.isPlaying = false
	discord.UpdateStatus(0, "nothing, the queue is empty")
	session.AudioManager.currentlyPlaying = "abosulutely nothing: N/A"
	return nil
}

//PlaySound ...
func (session Session) playSound(discord *discordgo.Session, input *bufio.Reader) (err error) {

	vc := session.voiceChannel

	vc.Speaking(true)
	defer vc.Speaking(false)
	send := make(chan []int16)
	go sendVoice(vc, send)
	//fmt.Println(session.AudioManager.queue)

	for session.AudioManager.isPlaying {
		if session.AudioManager.Paused {
			continue
		}

		if session.AudioManager.Skip {
			return
		}

		audioBuffer := make([]int16, frameSize*channels)
		if _, ok := session.AudioManager.bufferQueue[session.AudioManager.currentlyPlaying]; !ok {
			return
		}
		songReader := session.AudioManager.bufferQueue[session.AudioManager.currentlyPlaying]

		status := binary.Read(songReader, binary.LittleEndian, &audioBuffer)

		if status == io.EOF || err == io.ErrUnexpectedEOF {
			return nil
		}

		if err != nil {
			return err
		}

		send <- audioBuffer
	}

	return nil
}

func sendVoice(voice *discordgo.VoiceConnection, audioData <-chan []int16) {
	encoder, _ := gopus.NewEncoder(frameRate, channels, gopus.Audio)

	for {
		receive, ok := <-audioData
		if !ok {
			fmt.Println("PCM channel closed")
		}
		opus, _ := encoder.Encode(receive, frameSize, maxBytes)
		voice.OpusSend <- opus
	}
}
