package session

import (
	"sync"

	"github.com/bwmarrin/discordgo"
)

//Session defines the discordGo connection
type Session struct {
	self         *discordgo.Session
	voiceChannel *discordgo.VoiceConnection
	textChannel  string
	AudioManager *Audio
	lock         *sync.Mutex
}

//CreateSession creates a new Session struct
func CreateSession(voiceChannel *discordgo.VoiceConnection, textChannel string) *Session {
	session := new(Session)
	session.voiceChannel = voiceChannel
	session.textChannel = textChannel
	session.lock = &sync.Mutex{}
	//connection.channel = channel

	return session
}

//JoinChannel joins the channel
func JoinChannel(discord *discordgo.Session, guildID, voiceChannel, textChannel string) (*Session, error) {
	vc, err := discord.ChannelVoiceJoin(guildID, voiceChannel, false, true)
	if err != nil {
		return nil, err
	}

	return CreateSession(vc, textChannel), nil
}

//LeaveChannel leaves the channel
func (session Session) LeaveChannel(voice *discordgo.VoiceConnection) {
	//discord.Stop()
	voice.Disconnect()
}
