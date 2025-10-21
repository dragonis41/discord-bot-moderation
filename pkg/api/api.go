package api

import (
	"github.com/bwmarrin/discordgo"
	"github.com/dragonis41/discord-bot-moderation/internal/database"
	"github.com/dragonis41/discord-bot-moderation/pkg/logger"
)

type Discord struct {
	log    *logger.Logger
	db     *database.Database
	client *discordgo.Session
}

func NewClient(log *logger.Logger, db *database.Database, discordClient *discordgo.Session) *Discord {
	return &Discord{
		log:    log,
		db:     db,
		client: discordClient,
	}
}
