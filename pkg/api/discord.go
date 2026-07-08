package api

import (
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/bwmarrin/discordgo"
	"github.com/dragonis41/discord-bot-moderation/internal/database"
	"github.com/dragonis41/discord-bot-moderation/pkg/i18n"
	"github.com/dragonis41/discord-bot-moderation/pkg/logger"
	"github.com/dragonis41/discord-bot-moderation/pkg/utils"
)

// descLocsMap builds the per-locale translations of a command/option description
// from the i18n catalog. Discord shows the localized description to users whose
// client language matches; everyone else sees the default (English) Description.
func descLocsMap(key string) map[discordgo.Locale]string {
	return map[discordgo.Locale]string{
		discordgo.French:    i18n.T(i18n.FR, key),
		discordgo.SpanishES: i18n.T(i18n.ES, key),
	}
}

// descLocs is descLocsMap as the pointer type that ApplicationCommand expects
// (ApplicationCommandOption uses the non-pointer map directly).
func descLocs(key string) *map[discordgo.Locale]string {
	m := descLocsMap(key)
	return &m
}

// store is everything the api package needs from the database. It is composed
// from the interfaces the database package already exposes, so *database.Database
// satisfies it automatically while tests can supply a lightweight fake (embed
// store and override only the methods a given test exercises).
type store interface {
	database.HelperInterface
	database.LogChannelsInterface
	database.ModerationChannelsInterface
	database.ExcludedChannelsInterface
	database.ModerationRolesInterface
	database.GuildSettingsInterface
	database.LogsConfigInterface
	database.ModerationLogsInterface
	database.SystemLogsInterface
	database.AutomoderationInterface
}

// discordSender is the subset of *discordgo.Session used by the message-effect
// layer (DM/warn/delete/kick/ban and embed delivery). The real session satisfies
// it, so production passes its concrete *discordgo.Session unchanged; tests pass
// a fake that records the calls. The signatures match discordgo exactly,
// including the variadic RequestOption, so *discordgo.Session implements it for
// free.
type discordSender interface {
	Guild(guildID string, options ...discordgo.RequestOption) (*discordgo.Guild, error)
	UserChannelCreate(recipientID string, options ...discordgo.RequestOption) (*discordgo.Channel, error)
	ChannelMessageSend(channelID string, content string, options ...discordgo.RequestOption) (*discordgo.Message, error)
	ChannelMessageSendEmbed(channelID string, embed *discordgo.MessageEmbed, options ...discordgo.RequestOption) (*discordgo.Message, error)
	ChannelMessageDelete(channelID, messageID string, options ...discordgo.RequestOption) error
	GuildMemberDeleteWithReason(guildID, userID, reason string, options ...discordgo.RequestOption) error
	GuildBanCreateWithReason(guildID, userID, reason string, days int, options ...discordgo.RequestOption) error
}

// discordClient extends discordSender with the interaction-oriented methods used
// by the slash-command and selection-menu handlers (acknowledge, follow-up,
// edit, and the channel/role/user lookups). *discordgo.Session satisfies it, so
// production passes its concrete session unchanged while tests pass a fake.
// Handlers that need only the message-effect subset keep discordSender; those
// that drive an interaction take discordClient. Note the AddHandler callbacks
// (slashCommandHandler, the selection callbacks, etc.) must keep the concrete
// *discordgo.Session signature so discordgo's reflection can register them; they
// pass that concrete session down, which satisfies these interfaces for free.
type discordClient interface {
	discordSender
	InteractionRespond(interaction *discordgo.Interaction, resp *discordgo.InteractionResponse, options ...discordgo.RequestOption) error
	InteractionResponseEdit(interaction *discordgo.Interaction, newresp *discordgo.WebhookEdit, options ...discordgo.RequestOption) (*discordgo.Message, error)
	FollowupMessageCreate(interaction *discordgo.Interaction, wait bool, data *discordgo.WebhookParams, options ...discordgo.RequestOption) (*discordgo.Message, error)
	FollowupMessageEdit(interaction *discordgo.Interaction, messageID string, data *discordgo.WebhookEdit, options ...discordgo.RequestOption) (*discordgo.Message, error)
	GuildChannels(guildID string, options ...discordgo.RequestOption) ([]*discordgo.Channel, error)
	GuildRoles(guildID string, options ...discordgo.RequestOption) ([]*discordgo.Role, error)
	User(userID string, options ...discordgo.RequestOption) (*discordgo.User, error)
	ChannelMessageEditComplex(m *discordgo.MessageEdit, options ...discordgo.RequestOption) (*discordgo.Message, error)
}

type Discord struct {
	log    *logger.Logger
	db     store
	client *discordgo.Session
	cache  *Cache
}

func NewClient(log *logger.Logger, db store, discordClient *discordgo.Session, cache *Cache) *Discord {
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
	d.client.AddHandler(d.slashCommandHandler)                // Handler for slash commands
	d.client.AddHandler(d.handleLogChannelSelection)          // Handler for message component interaction for log channel selection
	d.client.AddHandler(d.handleModChannelSelection)          // Handler for message component interaction for moderation channel selection
	d.client.AddHandler(d.handleExcludedChannelSelection)     // Handler for message component interaction for excluded channel selection
	d.client.AddHandler(d.handleModRoleSelection)             // Handler for message component interaction for role selection
	d.client.AddHandler(d.handleAutomoderationSettings)       // Handler for automoderation settings selection
	d.client.AddHandler(d.handleRemoveBannedWordSelection)    // Handler for the banned-word removal dropdown
	d.client.AddHandler(d.handleRemoveBannedWebsiteSelection) // Handler for the banned-website removal dropdown
	d.client.AddHandler(d.handleLanguageSelection)            // Handler for the server language selection
	d.client.AddHandler(d.messageCreateHandler)               // Handler for message creation events
	d.client.AddHandler(d.messageUpdateHandler)               // Handler for message update events
	d.client.AddHandler(d.handleReportActions)                // Handler for report action buttons (kick/ban)

	// open session
	err := d.client.Open()
	if err != nil {
		d.logError("", "RunDiscordBot()", "Error opening Discord session: %s", err)
		return
	}
	defer func(discordClient *discordgo.Session) {
		if err := discordClient.Close(); err != nil {
			d.logWarning("", "RunDiscordBot()", "Error while closing the Discord session: %s", err)
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
		d.logWarning("", "RunDiscordBot()", "Failed to initialize system stats: %v", err)
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
	var maxValue float64 = 1000
	commands := []*discordgo.ApplicationCommand{
		{
			Name:                     "report",
			Description:              i18n.T(i18n.EN, "cmd.report.desc"),
			DescriptionLocalizations: descLocs("cmd.report.desc"),
			Options: []*discordgo.ApplicationCommandOption{
				{
					Type:                     discordgo.ApplicationCommandOptionUser,
					Name:                     "user",
					Description:              i18n.T(i18n.EN, "cmd.report.opt.user"),
					DescriptionLocalizations: descLocsMap("cmd.report.opt.user"),
					Required:                 true,
				},
				{
					Type:                     discordgo.ApplicationCommandOptionString,
					Name:                     "reason",
					Description:              i18n.T(i18n.EN, "cmd.report.opt.reason"),
					DescriptionLocalizations: descLocsMap("cmd.report.opt.reason"),
					Required:                 false,
				},
			},
		},
		{
			Name:                     "status",
			Description:              i18n.T(i18n.EN, "cmd.status.desc"),
			DescriptionLocalizations: descLocs("cmd.status.desc"),
		},
		{
			Name:                     "get-message-history",
			Description:              i18n.T(i18n.EN, "cmd.get-message-history.desc"),
			DescriptionLocalizations: descLocs("cmd.get-message-history.desc"),
			Options: []*discordgo.ApplicationCommandOption{
				{
					Type:                     discordgo.ApplicationCommandOptionInteger,
					Name:                     "limit",
					Description:              i18n.T(i18n.EN, "cmd.get-message-history.opt.limit"),
					DescriptionLocalizations: descLocsMap("cmd.get-message-history.opt.limit"),
					MinValue:                 &minValue,
					MaxValue:                 maxValue,
					Required:                 false,
				},
			},
		},
		{
			Name:                     "get-bot-logs",
			Description:              i18n.T(i18n.EN, "cmd.get-bot-logs.desc"),
			DescriptionLocalizations: descLocs("cmd.get-bot-logs.desc"),
		},
		{
			Name:                     "get-moderation-logs",
			Description:              i18n.T(i18n.EN, "cmd.get-moderation-logs.desc"),
			DescriptionLocalizations: descLocs("cmd.get-moderation-logs.desc"),
		},
		{
			Name:                     "set-log-channels",
			Description:              i18n.T(i18n.EN, "cmd.set-log-channels.desc"),
			DescriptionLocalizations: descLocs("cmd.set-log-channels.desc"),
		},
		{
			Name:                     "set-moderation-channels",
			Description:              i18n.T(i18n.EN, "cmd.set-moderation-channels.desc"),
			DescriptionLocalizations: descLocs("cmd.set-moderation-channels.desc"),
		},
		{
			Name:                     "set-excluded-channels",
			Description:              i18n.T(i18n.EN, "cmd.set-excluded-channels.desc"),
			DescriptionLocalizations: descLocs("cmd.set-excluded-channels.desc"),
		},
		{
			Name:                     "set-moderation-roles",
			Description:              i18n.T(i18n.EN, "cmd.set-moderation-roles.desc"),
			DescriptionLocalizations: descLocs("cmd.set-moderation-roles.desc"),
		},
		{
			Name:                     "add-banned-word",
			Description:              i18n.T(i18n.EN, "cmd.add-banned-word.desc"),
			DescriptionLocalizations: descLocs("cmd.add-banned-word.desc"),
			Options: []*discordgo.ApplicationCommandOption{
				{
					Type:                     discordgo.ApplicationCommandOptionString,
					Name:                     "word",
					Description:              i18n.T(i18n.EN, "cmd.add-banned-word.opt.word"),
					DescriptionLocalizations: descLocsMap("cmd.add-banned-word.opt.word"),
					Required:                 true,
				},
				{
					Type:                     discordgo.ApplicationCommandOptionBoolean,
					Name:                     "is_regex",
					Description:              i18n.T(i18n.EN, "cmd.add-banned-word.opt.is_regex"),
					DescriptionLocalizations: descLocsMap("cmd.add-banned-word.opt.is_regex"),
					Required:                 false,
				},
			},
		},
		{
			Name:                     "get-banned-words",
			Description:              i18n.T(i18n.EN, "cmd.get-banned-words.desc"),
			DescriptionLocalizations: descLocs("cmd.get-banned-words.desc"),
		},
		{
			Name:                     "remove-banned-word",
			Description:              i18n.T(i18n.EN, "cmd.remove-banned-word.desc"),
			DescriptionLocalizations: descLocs("cmd.remove-banned-word.desc"),
		},
		{
			Name:                     "add-banned-website",
			Description:              i18n.T(i18n.EN, "cmd.add-banned-website.desc"),
			DescriptionLocalizations: descLocs("cmd.add-banned-website.desc"),
			Options: []*discordgo.ApplicationCommandOption{
				{
					Type:                     discordgo.ApplicationCommandOptionString,
					Name:                     "url",
					Description:              i18n.T(i18n.EN, "cmd.add-banned-website.opt.url"),
					DescriptionLocalizations: descLocsMap("cmd.add-banned-website.opt.url"),
					Required:                 true,
				},
			},
		},
		{
			Name:                     "get-banned-websites",
			Description:              i18n.T(i18n.EN, "cmd.get-banned-websites.desc"),
			DescriptionLocalizations: descLocs("cmd.get-banned-websites.desc"),
		},
		{
			Name:                     "remove-banned-website",
			Description:              i18n.T(i18n.EN, "cmd.remove-banned-website.desc"),
			DescriptionLocalizations: descLocs("cmd.remove-banned-website.desc"),
		},
		{
			Name:                     "configure-automod",
			Description:              i18n.T(i18n.EN, "cmd.configure-automod.desc"),
			DescriptionLocalizations: descLocs("cmd.configure-automod.desc"),
		},
		{
			Name:                     "help",
			Description:              i18n.T(i18n.EN, "cmd.help.desc"),
			DescriptionLocalizations: descLocs("cmd.help.desc"),
		},
		{
			Name:                     "lang",
			Description:              i18n.T(i18n.EN, "cmd.lang.desc"),
			DescriptionLocalizations: descLocs("cmd.lang.desc"),
		},
	}

	// Register commands for each guild the bot is connected to
	for _, guild := range d.client.State.Guilds {
		_, err := d.client.ApplicationCommandBulkOverwrite(d.client.State.User.ID, guild.ID, commands)
		if err != nil {
			d.logError(guild.ID, "registerSlashCommands()", "Cannot register commands for guild %s: %v", guild.ID, err)
		} else {
			d.logSuccess(guild.ID, "registerSlashCommands()", "Registered all slash commands for guild %s", guild.ID)
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
	case "lang":
		d.selectLanguage(s, i)
	}
}

func (d *Discord) removeSlashCommands() {
	fmt.Printf("\n")
	log.Printf("Shutting down, removing slash commands...\n")

	for _, guild := range d.client.State.Guilds {
		// Pass an empty slice to remove all commands
		_, err := d.client.ApplicationCommandBulkOverwrite(d.client.State.User.ID, guild.ID, []*discordgo.ApplicationCommand{})
		if err != nil {
			d.logWarning(guild.ID, "removeSlashCommands()", "Cannot remove commands from guild %s: %v", guild.ID, err)
		} else {
			d.logSuccess(guild.ID, "removeSlashCommands()", "Removed all slash commands from guild %s", guild.ID)
		}
	}
}
