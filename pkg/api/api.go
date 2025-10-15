package api

import (
	"github.com/bwmarrin/discordgo"
)

type Discord struct {
	client *discordgo.Session
}

func NewClient(discordClient *discordgo.Session) *Discord {
	return &Discord{
		client: discordClient,
	}
}
