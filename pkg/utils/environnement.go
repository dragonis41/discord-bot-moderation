package utils

import (
	"fmt"
	"os"
)

func CheckEnvironmentVariables() {
	if os.Getenv("DISCORD_BOT_TOKEN") == "" {
		LogFatal(fmt.Sprintf("DISCORD_BOT_TOKEN environment variable is not set"))
	}
}
