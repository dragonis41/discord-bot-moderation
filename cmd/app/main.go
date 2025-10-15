package main

import (
	"fmt"
	"os"

	"github.com/bwmarrin/discordgo"
	"github.com/dragonis41/discord-bot-moderation/pkg/api"
	"github.com/dragonis41/discord-bot-moderation/pkg/utils"
	_ "github.com/joho/godotenv/autoload"
)

func init() {
	utils.CheckEnvironmentVariables()
}

func main() {
	// Create the Discord client
	discordClient, err := discordgo.New("Bot " + os.Getenv("DISCORD_BOT_TOKEN"))
	if err != nil {
		utils.LogError(fmt.Sprintf("Error while creating the Discord client: %s", err))
		return
	}

	client := api.NewClient(discordClient)
	client.RunDiscordBot()

	fmt.Printf("\nThe bot now shut down\n")
}
