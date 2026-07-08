package api

import (
	"fmt"
	"time"

	"github.com/bwmarrin/discordgo"
	"github.com/dragonis41/discord-bot-moderation/pkg/i18n"
	"github.com/dragonis41/discord-bot-moderation/pkg/model"
)

type DiscordUtilsInterface interface {
	displayConnectedGuilds()
	setMaxLogRetention()
	checkForGuildSetup()
	sendPrivateMessage(s discordSender, m *discordgo.Message, message string)
	sendLogChannelsEmbed(s discordSender, guildID string, embed *discordgo.MessageEmbed)
	sendModChannelsEmbed(s discordSender, guildID string, embed *discordgo.MessageEmbed)
	getUserRecentMessagesString(guildID string, user *discordgo.User, limit int) string
	logAutomoderationAction(s discordSender, m *discordgo.Message, action model.ModerationLogAction, trigger string, reason string)
	takeAutomoderationAction(s discordSender, m *discordgo.Message, action model.ModerationLogAction, reason string)
}

func (d *Discord) displayConnectedGuilds() {
	fmt.Printf("\n============== Connected Servers ==============\n")
	for _, guild := range d.client.State.Guilds {
		fmt.Printf("- [%s] (ID: %s)\n", d.guildDisplayName(guild), guild.ID)
	}
	fmt.Printf("===============================================\n\n")
}

func (d *Discord) setMaxLogRetention() {
	// Set default max log entries for each connected guild
	for _, guild := range d.client.State.Guilds {
		// Set default max system log entries
		systemLogIsSet, err := d.db.IsMaxSystemLogEntriesSet(guild.ID)
		if err != nil {
			d.logWarning(guild.ID, "setMaxLogRetention()", "Failed to check if max system log entries is set for guild %s: %v", guild.ID, err)
		}
		if !systemLogIsSet {
			if err := d.db.SetMaxSystemLogEntries(guild.ID, 10000); err != nil {
				d.logWarning(guild.ID, "setMaxLogRetention()", "Failed to set default max log entries for guild %s: %v", guild.ID, err)
			}
		}

		// Set default max moderation log entries
		moderationLogIsSet, err := d.db.IsMaxModerationLogEntriesSet(guild.ID)
		if err != nil {
			d.logWarning(guild.ID, "setMaxLogRetention()", "Failed to check if max moderation log entries is set for guild %s: %v", guild.ID, err)
		}
		if !moderationLogIsSet {
			if err := d.db.SetMaxModerationLogEntries(guild.ID, 1000); err != nil {
				d.logWarning(guild.ID, "setMaxLogRetention()", "Failed to set default max moderation log entries for guild %s: %v", guild.ID, err)
			}
		}
	}
}

// checkForGuildSetup checks if the guilds are properly set up in the database
func (d *Discord) checkForGuildSetup() {
	for _, guild := range d.client.State.Guilds {
		name := d.guildDisplayName(guild)

		modRoles, err := d.db.GetModerationRolesByGuildId(guild.ID)
		if err != nil {
			d.logError(guild.ID, "checkForGuildSetup()", "Failed to fetch moderation roles for guild [%s]: %s", name, err)
			continue
		}
		if len(modRoles) == 0 {
			d.logWarning(guild.ID, "checkForGuildSetup()", "⚠️ No moderation roles configured for guild [%s]", name)
		}

		logChannels, err := d.db.GetLogChannelsByGuildId(guild.ID)
		if err != nil {
			d.logError(guild.ID, "checkForGuildSetup()", "Failed to fetch log channels for guild [%s]: %s", name, err)
			continue
		}
		if len(logChannels) == 0 {
			d.logWarning(guild.ID, "checkForGuildSetup()", "⚠️ No log channels configured for guild [%s]", name)
		}

		modChannels, err := d.db.GetModerationChannelsByGuildId(guild.ID)
		if err != nil {
			d.logError(guild.ID, "checkForGuildSetup()", "Failed to fetch moderation channels for guild [%s]: %s", name, err)
			continue
		}
		if len(modChannels) == 0 {
			d.logWarning(guild.ID, "checkForGuildSetup()", "⚠️ No moderation channels configured for guild [%s]", name)
		}
	}
}

func (d *Discord) getUserRecentMessagesString(guildID string, user *discordgo.User, limit int) string {
	lang := d.lang(guildID)
	recentMessages := d.cache.GetUserRecentMessages(guildID, user.ID, limit)
	messagesText := ""

	// Budget for the whole list. The list is embedded in a description that also
	// holds the user/reason header, so we stay well below Discord's 4096 limit to
	// leave room for the rest; callers also truncate the final description as a
	// safety net.
	const maxRecentMessagesLength = 3500
	// Per-message cap, so a single long message can't crowd out the others.
	const maxMessageContentLength = 300

	for idx, msg := range recentMessages {
		content := msg.Content

		// Handle messages with attachments, stickers or embeds but no text
		if content == "" {
			if msg.AttachmentCount > 0 && msg.HasEmbeds {
				content = i18n.T(lang, "msg.files_embeds", msg.AttachmentCount)
			} else if msg.AttachmentCount > 0 {
				if msg.AttachmentCount == 1 {
					content = i18n.T(lang, "msg.file")
				} else {
					content = i18n.T(lang, "msg.files", msg.AttachmentCount)
				}
			} else if msg.HasEmbeds {
				content = i18n.T(lang, "msg.embed")
			} else if len(msg.Stickers) > 0 {
				if len(msg.Stickers) == 1 {
					content = i18n.T(lang, "msg.sticker")
				} else {
					content = i18n.T(lang, "msg.stickers", len(msg.Stickers))
				}
			} else {
				content = i18n.T(lang, "msg.empty")
			}
		} else {
			// Truncate overly long messages
			content = truncate(content, maxMessageContentLength)
			// Add attachment indicator if message has both text and attachments
			if msg.AttachmentCount > 0 {
				if msg.AttachmentCount == 1 {
					content += i18n.T(lang, "msg.attachment_one")
				} else {
					content += i18n.T(lang, "msg.attachments", msg.AttachmentCount)
				}
			}
		}

		line := fmt.Sprintf("%d. `%s` in <#%s>\n", idx+1, sanitizeInlineCode(content), msg.ChannelID)

		if len(messagesText)+len(line) > maxRecentMessagesLength {
			messagesText += i18n.T(lang, "msg.more", len(recentMessages)-idx)
			break
		}

		messagesText += line
	}

	if messagesText == "" {
		messagesText = i18n.T(lang, "msg.none_recent")
	}

	return messagesText
}

func (d *Discord) sendPrivateMessage(s discordSender, m *discordgo.Message, message string) {
	d.sendDM(s, m.GuildID, m.Author.ID, "sendPrivateMessage()", message)
}

func (d *Discord) sendPrivateMessageOnInteraction(s discordSender, i *discordgo.InteractionCreate, user *discordgo.User, message string) {
	d.sendDM(s, i.GuildID, user.ID, "sendPrivateMessageOnInteraction()", message)
}

// sendDM opens (or reuses) a DM channel with the user and sends a message,
// logging any failure under function.
func (d *Discord) sendDM(s discordSender, guildID, userID, function, message string) {
	channel, err := s.UserChannelCreate(userID)
	if err != nil {
		d.logError(guildID, function, "Failed to create DM channel: %s", err)
		return
	}
	if _, err := s.ChannelMessageSend(channel.ID, message); err != nil {
		d.logError(guildID, function, "Failed to send DM: %s", err)
	}
}

func (d *Discord) sendLogChannelsEmbed(s discordSender, guildID string, embed *discordgo.MessageEmbed) {
	logChannels, err := d.db.GetLogChannelsByGuildId(guildID)
	if err != nil {
		d.logError(guildID, "sendLogChannelsEmbed()", "Failed to fetch log channels: %s", err)
		return
	}

	for _, channelID := range logChannels {
		if _, err := s.ChannelMessageSendEmbed(channelID, embed); err != nil {
			d.logError(guildID, "sendLogChannelsEmbed()", "Failed to send embed to channel %s: %s", channelID, err)
		}
	}
}

func (d *Discord) sendModChannelsEmbed(s discordSender, guildID string, embed *discordgo.MessageEmbed) {
	modChannels, err := d.db.GetModerationChannelsByGuildId(guildID)
	if err != nil {
		d.logError(guildID, "sendModChannelsEmbed()", "Failed to fetch moderation channels: %s", err)
		return
	}

	for _, channelID := range modChannels {
		if _, err := s.ChannelMessageSendEmbed(channelID, embed); err != nil {
			d.logError(guildID, "sendModChannelsEmbed()", "Failed to send embed to channel %s: %s", channelID, err)
		}
	}
}

// logAutomoderationAction sends an alert to moderators and save it in DB
func (d *Discord) logAutomoderationAction(s discordSender, m *discordgo.Message, action model.ModerationLogAction, trigger string, reason string) {
	// Log in the database
	if err := d.db.AddModerationLogEntry(m.GuildID, action, m.Author.ID, m.Author.Username, trigger, reason); err != nil {
		d.logError(m.GuildID, "logAutomoderationAction()", "Error logging automod action to database: %s", err)
	}

	lang := d.lang(m.GuildID)

	// Get last 10 recent messages from the user
	recentMessages := d.getUserRecentMessagesString(m.GuildID, m.Author, 10)

	description := i18n.T(lang, "automod.alert.description",
		m.Author.ID, m.Author.Username, m.Author.ID, action, trigger, reason, recentMessages,
	)

	embed := &discordgo.MessageEmbed{
		Title: i18n.T(lang, "automod.alert.title"),
		// Keep within Discord's description limit so the alert never fails to send.
		Description: truncate(description, maxEmbedDescriptionLength),
		Color:       model.Violet.Int(),
		Timestamp:   time.Now().UTC().Format(time.RFC3339),
	}

	// Send alert to all log channels
	d.sendLogChannelsEmbed(s, m.GuildID, embed)
}

func (d *Discord) takeAutomoderationAction(s discordSender, m *discordgo.Message, action model.ModerationLogAction, reason string) {
	guildName := guildNameByID(s, m.GuildID)
	lang := d.lang(m.GuildID)

	switch action {
	case model.ActionDeleteMessage:
		// Delete the message
		if err := s.ChannelMessageDelete(m.ChannelID, m.ID); err != nil {
			d.logError(m.GuildID, "takeAutomoderationAction()", "Failed to delete message: %s", err)
			return
		}
	case model.ActionWarn:
		// Send a private warning message to the user
		d.sendPrivateMessage(s, m, reason)
		return
	case model.ActionKick:
		message := i18n.T(lang, "action.kick_dm", guildName, reason)
		d.sendPrivateMessage(s, m, message)
		// Kick user
		if err := s.GuildMemberDeleteWithReason(m.GuildID, m.Author.ID, reason); err != nil {
			d.logError(m.GuildID, "takeAutomoderationAction()", "Failed to kick user %s: %s", m.Author.ID, err)
			return
		}
	case model.ActionBan:
		message := i18n.T(lang, "action.ban_dm", guildName, reason)
		d.sendPrivateMessage(s, m, message)
		// Permanent ban with message deletion
		if err := s.GuildBanCreateWithReason(m.GuildID, m.Author.ID, reason, 1); err != nil {
			d.logError(m.GuildID, "takeAutomoderationAction()", "Failed to ban user %s: %s", m.Author.ID, err)
			return
		}
	}
}
