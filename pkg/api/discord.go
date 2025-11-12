package api

import (
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/bwmarrin/discordgo"
	"github.com/dragonis41/discord-bot-moderation/internal/database"
	"github.com/dragonis41/discord-bot-moderation/pkg/logger"
	"github.com/dragonis41/discord-bot-moderation/pkg/utils"
)

type Discord struct {
	log    *logger.Logger
	db     *database.Database
	client *discordgo.Session
	cache  *Cache
}

func NewClient(log *logger.Logger, db *database.Database, discordClient *discordgo.Session, cache *Cache) *Discord {
	return &Discord{
		log:    log,
		db:     db,
		client: discordClient,
		cache:  cache,
	}
}

type DiscordHandlerInterface interface {
	RunDiscordBot()
}

func (d *Discord) RunDiscordBot() {
	// Set intents to receive message content
	d.client.Identify.Intents = discordgo.IntentsGuildMessages | discordgo.IntentGuildMembers | discordgo.IntentsDirectMessages | discordgo.IntentsMessageContent

	// add an event handler for slash commands
	d.client.AddHandler(d.slashCommandHandler)            // Handler for slash commands
	d.client.AddHandler(d.handleLogChannelSelection)      // Handler for message component interaction for log channel selection
	d.client.AddHandler(d.handleModChannelSelection)      // Handler for message component interaction for moderation channel selection
	d.client.AddHandler(d.handleExcludedChannelSelection) // Handler for message component interaction for excluded channel selection
	d.client.AddHandler(d.handleModRoleSelection)         // Handler for message component interaction for role selection
	d.client.AddHandler(d.handleAutomoderationSettings)   // Handler for automoderation settings selection
	d.client.AddHandler(d.messageCreateHandler)           // Handler for message creation events
	d.client.AddHandler(d.messageUpdateHandler)           // Handler for message update events
	d.client.AddHandler(d.handleReportActions)            // Handler for report action buttons (kick/ban)

	// open session
	err := d.client.Open()
	if err != nil {
		d.log.LogError(logger.LogModel{Database: d.db, Function: "RunDiscordBot()",
			Message: fmt.Sprintf("Error opening Discord session: %s", err),
		})
		return
	}
	defer func(discordClient *discordgo.Session) {
		err := discordClient.Close()
		if err != nil {
			d.log.LogWarning(logger.LogModel{Database: d.db, Function: "RunDiscordBot()",
				Message: fmt.Sprintf("Error while closing the Discord session: %s", err),
			})
		}
	}(d.client) // close session, after function termination

	// Initialize uptime tracking
	utils.NewUptime()

	// Display connected guilds
	d.displayConnectedGuilds()

	// Set max log retention for system and moderation logs
	d.setMaxLogRetention()

	// Check if the moderation roles, log channels and moderation channels are set for each guild
	d.checkForGuildSetup()

	// Initialize system stats
	if err := utils.InitSystemStats(); err != nil {
		d.log.LogWarning(logger.LogModel{Database: d.db, Function: "RunDiscordBot()",
			Message: fmt.Sprintf("Failed to initialize system stats: %v", err),
		})
	}

	// Register slash commands
	d.registerSlashCommands()

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
	var minValue float64 = 1
	var maxValue float64 = 100
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
			Name:        "get-message-history",
			Description: "Affiche l'historique des messages en cache de la guild",
			Options: []*discordgo.ApplicationCommandOption{
				{
					Type:        discordgo.ApplicationCommandOptionInteger,
					Name:        "limit",
					Description: "Le nombre de messages à récupérer (par défaut 10, maximum 100)",
					MinValue:    &minValue,
					MaxValue:    maxValue,
					Required:    false,
				},
			},
		},
		{
			Name:        "get-bot-logs",
			Description: "Affiche les derniers logs du bot",
		},
		{
			Name:        "get-moderation-logs",
			Description: "Affiche les derniers logs de modération",
		},
		{
			Name:        "set-log-channels",
			Description: "Sélectionne les canaux où les rapports de modération seront envoyés",
		},
		{
			Name:        "set-moderation-channels",
			Description: "Sélectionne les canaux où les signalements seront envoyés",
		},
		{
			Name:        "set-excluded-channels",
			Description: "Sélectionne les canaux exclus de l'automodération",
		},
		{
			Name:        "set-moderation-roles",
			Description: "Sélectionne les roles qui auront les permissions de modération",
		},
		{
			Name:        "add-banned-word",
			Description: "Ajoute un mot ou une expression à la liste des mots interdits",
			Options: []*discordgo.ApplicationCommandOption{
				{
					Type:        discordgo.ApplicationCommandOptionString,
					Name:        "word",
					Description: "Le mot ou la regex à interdire",
					Required:    true,
				},
				{
					Type:        discordgo.ApplicationCommandOptionBoolean,
					Name:        "is_regex",
					Description: "Si le mot est une regex",
					Required:    false,
				},
			},
		},
		{
			Name:        "get-banned-words",
			Description: "Affiche la liste des mots interdits",
		},
		{
			Name:        "remove-banned-word",
			Description: "Supprime un mot de la liste des mots interdits",
			Options: []*discordgo.ApplicationCommandOption{
				{
					Type:        discordgo.ApplicationCommandOptionInteger,
					Name:        "id",
					Description: "L'ID du mot à supprimer (utilisez /list-banned-words pour voir les IDs)",
					Required:    true,
				},
			},
		},
		{
			Name:        "add-banned-website",
			Description: "Ajoute un site web à la liste des sites interdits",
			Options: []*discordgo.ApplicationCommandOption{
				{
					Type:        discordgo.ApplicationCommandOptionString,
					Name:        "url",
					Description: "L'URL du site à interdire (ex: example.com)",
					Required:    true,
				},
			},
		},
		{
			Name:        "get-banned-websites",
			Description: "Affiche la liste des sites web interdits",
		},
		{
			Name:        "remove-banned-website",
			Description: "Supprime un site web de la liste des sites interdits",
			Options: []*discordgo.ApplicationCommandOption{
				{
					Type:        discordgo.ApplicationCommandOptionInteger,
					Name:        "id",
					Description: "L'ID du site à supprimer (utilisez /list-banned-websites pour voir les IDs)",
					Required:    true,
				},
			},
		},
		{
			Name:        "configure-automod",
			Description: "Active ou désactive les fonctionnalités d'automodération",
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
			d.log.LogError(logger.LogModel{Database: d.db, Function: "registerSlashCommands()",
				Message: fmt.Sprintf("Cannot register commands for guild %s: %v", guild.ID, err),
			})
		} else {
			d.log.LogSuccess(logger.LogModel{Database: d.db, Function: "registerSlashCommands()",
				Message: fmt.Sprintf("Registered all slash commands for guild %s", guild.ID),
			})
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
	case "get-message-history":
		d.getMessageHistory(s, i)
	case "get-bot-logs":
		d.getBotLogs(s, i)
	case "get-moderation-logs":
		d.getModerationLogs(s, i)
	case "set-log-channels":
		d.selectLogChannels(s, i)
	case "set-moderation-channels":
		d.selectModeratorChannels(s, i)
	case "set-excluded-channels":
		d.selectExcludedChannels(s, i)
	case "set-moderation-roles":
		d.selectModeratorRoles(s, i)
	case "add-banned-word":
		d.addBannedWord(s, i)
	case "get-banned-words":
		d.listBannedWords(s, i)
	case "remove-banned-word":
		d.removeBannedWord(s, i)
	case "add-banned-website":
		d.addBannedWebsite(s, i)
	case "get-banned-websites":
		d.listBannedWebsites(s, i)
	case "remove-banned-website":
		d.removeBannedWebsite(s, i)
	case "configure-automod":
		d.configureAutomod(s, i)
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
			d.log.LogWarning(logger.LogModel{Database: d.db, Function: "removeSlashCommands()",
				Message: fmt.Sprintf("Cannot remove commands from guild %s: %v", guild.ID, err),
			})
		} else {
			d.log.LogSuccess(logger.LogModel{Database: d.db, Function: "removeSlashCommands()",
				Message: fmt.Sprintf("Removed all slash commands from guild %s", guild.ID),
			})
		}
	}
}
