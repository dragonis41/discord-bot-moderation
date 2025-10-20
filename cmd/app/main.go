package main

import (
	"fmt"
	"os"

	"github.com/bwmarrin/discordgo"
	"github.com/dragonis41/discord-bot-moderation/internal/database"
	"github.com/dragonis41/discord-bot-moderation/pkg/api"
	"github.com/dragonis41/discord-bot-moderation/pkg/utils"
	_ "github.com/joho/godotenv/autoload"
)

func init() {
	utils.CheckEnvironmentVariables()
}

func main() {
	db, err := database.NewDatabase()
	if err != nil {
		utils.LogError(fmt.Sprintf("Error while initializing the database: %s", err))
		return
	}
	if err := db.Migrate(); err != nil {
		utils.LogError(fmt.Sprintf("Error while migrating the database: %s", err))
		return
	}

	// Create the Discord client
	discordClient, err := discordgo.New("Bot " + os.Getenv("DISCORD_BOT_TOKEN"))
	if err != nil {
		utils.LogError(fmt.Sprintf("Error while creating the Discord client: %s", err))
		return
	}

	client := api.NewClient(db, discordClient)
	client.RunDiscordBot()

	if err := db.CloseDatabase(); err != nil {
		utils.LogWarning(fmt.Sprintf("Error while closing the database: %s\n", err))
	}
	fmt.Printf("\nThe bot now shut down\n")
}
