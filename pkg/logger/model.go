package logger

import "github.com/dragonis41/discord-bot-moderation/internal/database"

type LogModel struct {
	Database *database.Database
	GuildID  string
	Function string
	Message  string
}
