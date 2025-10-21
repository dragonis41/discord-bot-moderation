package utils

import (
	"os"

	"github.com/dragonis41/discord-bot-moderation/pkg/logger"
)

func CheckEnvironmentVariables() {
	l := logger.NewLogger()

	if os.Getenv("DISCORD_BOT_TOKEN") == "" {
		l.LogFatal("DISCORD_BOT_TOKEN environment variable is not set")
	}
}
