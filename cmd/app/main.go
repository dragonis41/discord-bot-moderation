package main

import (
	"fmt"
	"os"
	"time"

	"github.com/bwmarrin/discordgo"
	"github.com/dragonis41/discord-bot-moderation/internal/database"
	"github.com/dragonis41/discord-bot-moderation/pkg/api"
	"github.com/dragonis41/discord-bot-moderation/pkg/localization"
	"github.com/dragonis41/discord-bot-moderation/pkg/logger"
	"github.com/dragonis41/discord-bot-moderation/pkg/utils"
	_ "github.com/joho/godotenv/autoload"
)

func init() {
	utils.CheckEnvironmentVariables()
}

func main() {
	// Initialize the logger
	l := logger.NewLogger()

	// Initialize localization
	l.LogInfo(logger.LogModel{Message: "Initializing localization..."})
	locale := localization.NewLocale()
	l.LogSuccess(logger.LogModel{Message: "Localization initialized successfully"})

	// Initialize the database
	db, err := database.NewDatabase()
	if err != nil {
		l.LogError(logger.LogModel{Message: fmt.Sprintf("Error while initializing the database: %s", err)})
		return
	}
	// Migrate the database
	l.LogInfo(logger.LogModel{Message: "Starting database migration..."})
	if err := db.Migrate(); err != nil {
		l.LogError(logger.LogModel{Message: fmt.Sprintf("Error while migrating the database: %s", err)})
		return
	}
	l.LogSuccess(logger.LogModel{Message: "Database initialized successfully"})

	// Create the Discord client
	discord, err := discordgo.New("Bot " + os.Getenv("DISCORD_BOT_TOKEN"))
	if err != nil {
		l.LogError(logger.LogModel{Database: db, Function: "main()", Message: fmt.Sprintf("Error while creating the Discord client: %s", err)})
		return
	}

	client := api.NewClient(l, db, discord, locale.InitLocalization(), api.NewCache(100, 3, 5*time.Minute))
	client.RunDiscordBot()

	if err := db.CloseDatabase(); err != nil {
		l.LogError(logger.LogModel{Message: fmt.Sprintf("Error while closing the database connection: %s", err)})
	}
	l.LogSuccess(logger.LogModel{Message: "Database connection closed successfully\n"})
	fmt.Printf("The bot is now shut down\n\n")
}
