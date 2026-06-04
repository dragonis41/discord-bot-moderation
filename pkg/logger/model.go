package logger

import "github.com/dragonis41/discord-bot-moderation/pkg/model"

// SystemLogStore is the slice of the database the logger actually needs:
// persisting a single system log entry. *database.Database satisfies it, and so
// can a test double. Depending on this interface instead of the concrete
// database type keeps the logger decoupled from the database package.
type SystemLogStore interface {
	AddSystemLogEntry(guildID string, logType model.SystemLogType, function string, content string) error
}

type LogModel struct {
	Database SystemLogStore // If nil, the log won't be saved to the database
	GuildID  string
	Function string
	Message  string
}
