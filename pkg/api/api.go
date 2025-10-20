package api

import (
	"github.com/bwmarrin/discordgo"
)

var (
	red   = 0xff0000
	green = 0x00dd00
	blue  = 0x0099ff

	defaultFooter = &discordgo.MessageEmbedFooter{Text: "💡 Hint: Utilisez /help pour lister les commandes disponibles."}
)

type Discord struct {
	client *discordgo.Session
}

func NewClient(discordClient *discordgo.Session) *Discord {
	return &Discord{
		client: discordClient,
	}
}
