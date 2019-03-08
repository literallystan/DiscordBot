package cmd

import (
	"github.com/bwmarrin/discordgo"
	"github.com/literallystan/DiscordBot/session"
)

//Skip ...
func Skip(session *session.Session, discord *discordgo.Session) {
	session.SkipSong(discord)
	session.AudioManager.Skip = true

}
