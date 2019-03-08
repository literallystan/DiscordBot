package main

import (
	"flag"
	"fmt"
	"os"
	"os/signal"
	"strings"
	"syscall"

	"github.com/literallystan/DiscordBot/cmd"
	"github.com/literallystan/DiscordBot/session"

	"github.com/bwmarrin/discordgo"
)

var token string
var channelID string
var sessions = make(map[string]*session.Session)
var buffer = make([][]byte, 0)

func init() {
	flag.StringVar(&token, "t", "", "Bot Token")
	flag.StringVar(&channelID, "c", "", "channel ID")
	flag.Parse()
}

func main() {

	if token == "" {
		fmt.Println("No token provided. Please run: <bot> -t <bot token>")
		return
	}

	// Create a new Discord session using the provided bot token.
	discord, err := discordgo.New("Bot " + token)
	if err != nil {
		fmt.Println("Error creating Discord session: ", err)
		return
	}

	// Register messageCreate as a callback for the messageCreate events.
	discord.AddHandler(messageCreate)

	// Open the websocket and begin listening.
	err = discord.Open()
	if err != nil {
		fmt.Println("Error opening Discord session: ", err)
		fmt.Println(err)
		panic(err)
	}

	// Wait here until CTRL-C or other term signal is received.
	fmt.Println("Bot is now running.  Press CTRL-C to exit.")
	sc := make(chan os.Signal, 1)
	signal.Notify(sc, syscall.SIGINT, syscall.SIGTERM, os.Interrupt, os.Kill)
	<-sc

	// Cleanly close down the Discord session.
	discord.Close()
}

// This function will be called (due to AddHandler above) every time a new
// message is created on any channel that the autenticated bot has access to.
func messageCreate(discord *discordgo.Session, m *discordgo.MessageCreate) {

	// Ignore all messages created by the bot itself
	// This isn't required in this specific example but it's a good practice.
	if m.Author.ID == discord.State.User.ID {
		return
	}

	if strings.HasPrefix(m.Content, "!play") {

		c, err := discord.State.Channel(m.ChannelID)
		if err != nil {
			// Could not find channel.
			return
		}
		// Find the guild for that channel.
		g, err := discord.State.Guild(c.GuildID)
		if err != nil {
			// Could not find guild.
			return
		}
		for _, vs := range g.VoiceStates {
			if vs.UserID == m.Author.ID {
				sessions[vs.ChannelID].AudioManager.Paused = false

				_ = sessions[vs.ChannelID].PlayQueue(discord)

				return
			}
		}
	}

	if strings.HasPrefix(m.Content, "!skip") {
		c, err := discord.State.Channel(m.ChannelID)
		if err != nil {
			// Could not find channel.
			return
		}
		// Find the guild for that channel.
		g, err := discord.State.Guild(c.GuildID)
		if err != nil {
			// Could not find guild.
			return
		}
		for _, vs := range g.VoiceStates {
			if vs.UserID == m.Author.ID {
				cmd.Skip(sessions[vs.ChannelID], discord)

				return
			}
		}
	}

	if strings.HasPrefix(m.Content, "!pause") {
		c, err := discord.State.Channel(m.ChannelID)
		if err != nil {
			// Could not find channel.
			return
		}
		// Find the guild for that channel.
		g, err := discord.State.Guild(c.GuildID)
		if err != nil {
			// Could not find guild.
			return
		}
		for _, vs := range g.VoiceStates {
			if vs.UserID == m.Author.ID {
				sessions[vs.ChannelID].AudioManager.Paused = true

				return
			}
		}
	}

	if strings.HasPrefix(m.Content, "!queue") {
		c, err := discord.State.Channel(m.ChannelID)
		if err != nil {
			// Could not find channel.
			return
		}
		// Find the guild for that channel.
		g, err := discord.State.Guild(c.GuildID)
		if err != nil {
			// Could not find guild.
			return
		}
		for _, vs := range g.VoiceStates {
			if vs.UserID == m.Author.ID {
				_ = sessions[vs.ChannelID].AddToQueue(discord, m.Content[6:], m.Author.Username)

				return
			}
		}
	}

	if strings.HasPrefix(m.Content, "!join") {
		c, err := discord.State.Channel(m.ChannelID)
		if err != nil {
			// Could not find channel.
			return
		}

		// Find the guild for that channel.
		g, err := discord.State.Guild(c.GuildID)
		if err != nil {
			// Could not find guild.
			return
		}

		for _, vs := range g.VoiceStates {
			if vs.UserID == m.Author.ID {
				sess, err := session.JoinChannel(discord, g.ID, vs.ChannelID, channelID)
				if err != nil {
					fmt.Println("failed to join channel")
				}

				sess.AudioManager = sess.CreateAudio()
				sessions[vs.ChannelID] = sess

			}
		}
	}

	if strings.HasPrefix(m.Content, "!stop") {
		c, err := discord.State.Channel(m.ChannelID)
		if err != nil {
			// Could not find channel.
			return
		}

		// Find the guild for that channel.
		g, err := discord.State.Guild(c.GuildID)
		if err != nil {
			// Could not find guild.
			return
		}

		for _, vs := range g.VoiceStates {
			if vs.UserID == m.Author.ID {
				sessions[vs.ChannelID].Stop(discord)

			}
		}
	}
}
