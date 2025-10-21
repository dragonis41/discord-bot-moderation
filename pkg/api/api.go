package api

import (
	"github.com/bwmarrin/discordgo"
	"github.com/dragonis41/discord-bot-moderation/internal/database"
)

var (
	Red   = 0xff0000
	Green = 0x00dd00
	Blue  = 0x0099ff

	defaultFooter = &discordgo.MessageEmbedFooter{Text: "💡 Hint: Utilisez /help pour lister les commandes disponibles."}
)

type Discord struct {
	db     *database.Database
	client *discordgo.Session
}

func NewClient(db *database.Database, discordClient *discordgo.Session) *Discord {
	return &Discord{
		db:     db,
		client: discordClient,
	}
}
