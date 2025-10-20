package api

import (
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/bwmarrin/discordgo"
	"github.com/dragonis41/discord-bot-moderation/pkg/utils"
)

type DiscordHandlerInterface interface {
	RunDiscordBot()
}

func (d *Discord) displayConnectedGuilds() {
	fmt.Printf("\n======= Connected Servers =======\n")
	for _, guild := range d.client.State.Guilds {
		// Fetch full guild data if name is empty
		if guild.Name == "" {
			fullGuild, err := d.client.Guild(guild.ID)
			if err != nil {
				fmt.Printf("Server: [<error fetching>] (ID: %s)\n", guild.ID)
				continue
			}
			guild = fullGuild
		}
		fmt.Printf("Server: [%s] (ID: %s)\n", guild.Name, guild.ID)
	}
	fmt.Printf("=================================\n\n")
}

func (d *Discord) RunDiscordBot() {
	// Set intents to receive message content
	d.client.Identify.Intents = discordgo.IntentsGuildMessages | discordgo.IntentsDirectMessages | discordgo.IntentsMessageContent

	// add an event handler for slash commands
	d.client.AddHandler(d.slashCommandHandler)
	d.client.AddHandler(d.handleChannelSelection)

	// open session
	err := d.client.Open()
	if err != nil {
		utils.LogError(fmt.Sprintf("Error opening Discord session: %s", err))
		return
	}
	defer func(discordClient *discordgo.Session) {
		err := discordClient.Close()
		if err != nil {
			utils.LogWarning(fmt.Sprintf("Error while closing the Discord session: %s", err))
		}
	}(d.client) // close session, after function termination

	// Display connected guilds
	d.displayConnectedGuilds()

	// Register slash commands
	d.registerSlashCommands()

	// Initialize system stats
	if err := utils.InitSystemStats(); err != nil {
		utils.LogWarning(fmt.Sprintf("Failed to initialize system stats: %v", err))
	}

	// Initialize uptime tracking
	utils.NewUptime()

	// keep bot running until there is an os interruption (ctrl+c or SIGTERM signal)
	fmt.Printf("\n")
	log.Printf("Bot running....\n")
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)
	<-c

	// Clean up slash commands on exit
	d.removeSlashCommands()
}

func (d *Discord) registerSlashCommands() {
	commands := []*discordgo.ApplicationCommand{
		{
			Name:        "report",
			Description: "Signal un utilisateur à la modération",
			Options: []*discordgo.ApplicationCommandOption{
				{
					Type:        discordgo.ApplicationCommandOptionUser,
					Name:        "user",
					Description: "L'utilisateur à signaler",
					Required:    true,
				},
				{
					Type:        discordgo.ApplicationCommandOptionString,
					Name:        "reason",
					Description: "La raison du signalement",
					Required:    false,
				},
			},
		},
		{
			Name:        "status",
			Description: "Affiche le statut du bot",
		},
		{
			Name:        "set-moderation-channels",
			Description: "Sélectionne les canaux où les rapports de modération seront envoyés",
		},
		{
			Name:        "help",
			Description: "Liste toutes les commandes disponibles",
		},
	}

	// Register commands for each guild the bot is connected to
	for _, guild := range d.client.State.Guilds {
		_, err := d.client.ApplicationCommandBulkOverwrite(d.client.State.User.ID, guild.ID, commands)
		if err != nil {
			utils.LogError(fmt.Sprintf("Cannot register commands for guild %s: %v", guild.ID, err))
		} else {
			utils.LogSuccess(fmt.Sprintf("Registered all slash commands for guild %s", guild.ID))
		}
	}
}

func (d *Discord) slashCommandHandler(s *discordgo.Session, i *discordgo.InteractionCreate) {
	// Check if this is a command interaction
	if i.Type != discordgo.InteractionApplicationCommand {
		return
	}

	switch i.ApplicationCommandData().Name {
	case "report":
		d.reportUser(s, i)
	case "status":
		d.showStatus(s, i)
	case "set-moderation-channels":
		d.selectModeratorChannels(s, i)
	case "help":
		d.showHelp(s, i)
	}
}

func (d *Discord) removeSlashCommands() {
	fmt.Printf("\n")
	log.Printf("Shutting down, removing slash commands...\n")

	for _, guild := range d.client.State.Guilds {
		// Pass an empty slice to remove all commands
		_, err := d.client.ApplicationCommandBulkOverwrite(d.client.State.User.ID, guild.ID, []*discordgo.ApplicationCommand{})
		if err != nil {
			utils.LogWarning(fmt.Sprintf("Cannot remove commands from guild %s: %v", guild.ID, err))
		} else {
			utils.LogSuccess(fmt.Sprintf("Removed all slash commands from guild %s", guild.ID))
		}
	}
}
