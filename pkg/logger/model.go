package logger

import "github.com/dragonis41/discord-bot-moderation/internal/database"

type LogModel struct {
	Database *database.Database // If nil, the log won't be saved to the database
	GuildID  string
	Function string
	Message  string
}
