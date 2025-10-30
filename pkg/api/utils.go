package api

import (
	"fmt"
	"time"

	"github.com/bwmarrin/discordgo"
	"github.com/dragonis41/discord-bot-moderation/pkg/logger"
	"github.com/dragonis41/discord-bot-moderation/pkg/model"
)

type DiscordUtilsInterface interface {
	displayConnectedGuilds()
	setMaxLogRetention()
	checkForGuildSetup()
	sendPrivateMessage(s *discordgo.Session, m *discordgo.Message, message string)
	sendLogChannelsEmbed(s *discordgo.Session, guildID string, embed *discordgo.MessageEmbed)
	sendModChannelsEmbed(s *discordgo.Session, guildID string, embed *discordgo.MessageEmbed)
	getUserRecentMessagesString(guildID string, user *discordgo.User, limit int) string
	logAutomoderationAction(s *discordgo.Session, m *discordgo.Message, action model.ModerationLogAction, trigger string, reason string)
	takeAutomoderationAction(s *discordgo.Session, m *discordgo.Message, action model.ModerationLogAction, reason string)
}

func (d *Discord) displayConnectedGuilds() {
	fmt.Printf("\n============== Connected Servers ==============\n")
	for _, guild := range d.client.State.Guilds {
		// Fetch full guild data if name is empty
		if guild.Name == "" {
			fullGuild, err := d.client.Guild(guild.ID)
			if err != nil {
				guild.Name = "<error fetching>"
			}
			guild = fullGuild
		}
		fmt.Printf("- [%s] (ID: %s)\n", guild.Name, guild.ID)
	}
	fmt.Printf("===============================================\n\n")
}

func (d *Discord) setMaxLogRetention() {
	// Set default max log entries for each connected guild
	for _, guild := range d.client.State.Guilds {
		// Set default max system log entries
		systemLogIsSet, err := d.db.IsMaxSystemLogEntriesSet(guild.ID)
		if err != nil {
			d.log.LogWarning(logger.LogModel{Database: d.db, Function: "setMaxLogRetention()",
				Message: fmt.Sprintf("Failed to check if max system log entries is set for guild %s: %v", guild.ID, err),
			})
		}
		if !systemLogIsSet {
			err := d.db.SetMaxSystemLogEntries(guild.ID, 10000)
			if err != nil {
				d.log.LogWarning(logger.LogModel{Database: d.db, Function: "setMaxLogRetention()",
					Message: fmt.Sprintf("Failed to set default max log entries for guild %s: %v", guild.ID, err),
				})
			}
		}

		// Set default max moderation log entries
		moderationLogIsSet, err := d.db.IsMaxModerationLogEntriesSet(guild.ID)
		if err != nil {
			d.log.LogWarning(logger.LogModel{Database: d.db, Function: "setMaxLogRetention()",
				Message: fmt.Sprintf("Failed to check if max moderation log entries is set for guild %s: %v", guild.ID, err),
			})
		}
		if !moderationLogIsSet {
			err = d.db.SetMaxModerationLogEntries(guild.ID, 100)
			if err != nil {
				d.log.LogWarning(logger.LogModel{Database: d.db, Function: "setMaxLogRetention()",
					Message: fmt.Sprintf("Failed to set default max moderation log entries for guild %s: %v", guild.ID, err),
				})
			}
		}
	}
}

// checkForGuildSetup checks if the guilds are properly set up in the database
func (d *Discord) checkForGuildSetup() {
	for _, guild := range d.client.State.Guilds {
		if guild.Name == "" {
			fullGuild, err := d.client.Guild(guild.ID)
			if err != nil {
				guild.Name = "<error fetching>"
			}
			guild = fullGuild
		}

		modRoles, err := d.db.GetModerationRolesByGuildId(guild.ID)
		if err != nil {
			d.log.LogError(logger.LogModel{Database: d.db, GuildID: guild.ID, Function: "checkForGuildSetup()", Message: fmt.Sprintf("Failed to fetch moderation roles for guild [%s]: %s", guild.Name, err)})
			continue
		}
		if len(modRoles) == 0 {
			d.log.LogWarning(logger.LogModel{Database: d.db, GuildID: guild.ID, Function: "checkForGuildSetup()", Message: fmt.Sprintf("⚠️ No moderation roles configured for guild [%s]", guild.Name)})
		}

		logChannels, err := d.db.GetLogChannelsByGuildId(guild.ID)
		if err != nil {
			d.log.LogError(logger.LogModel{Database: d.db, GuildID: guild.ID, Function: "checkForGuildSetup()", Message: fmt.Sprintf("Failed to fetch log channels for guild [%s]: %s", guild.Name, err)})
			continue
		}
		if len(logChannels) == 0 {
			d.log.LogWarning(logger.LogModel{Database: d.db, GuildID: guild.ID, Function: "checkForGuildSetup()", Message: fmt.Sprintf("⚠️ No log channels configured for guild [%s]", guild.Name)})
		}

		modChannels, err := d.db.GetModerationChannelsByGuildId(guild.ID)
		if err != nil {
			d.log.LogError(logger.LogModel{Database: d.db, GuildID: guild.ID, Function: "checkForGuildSetup()", Message: fmt.Sprintf("Failed to fetch moderation channels for guild [%s]: %s", guild.Name, err)})
			continue
		}
		if len(modChannels) == 0 {
			d.log.LogWarning(logger.LogModel{Database: d.db, GuildID: guild.ID, Function: "checkForGuildSetup()", Message: fmt.Sprintf("⚠️ No moderation channels configured for guild [%s]", guild.Name)})
		}
	}
}

func (d *Discord) getUserRecentMessagesString(guildID string, user *discordgo.User, limit int) string {
	recentMessages := d.cache.GetUserRecentMessages(guildID, user.ID, limit)
	messagesText := ""
	const maxRecentMessagesLength = 800

	for idx, msg := range recentMessages {
		content := msg.Content

		// Handle messages with attachments but no text
		if content == "" {
			if msg.AttachmentCount > 0 && msg.HasEmbeds {
				content = fmt.Sprintf("<fichier(s): %d + embed(s)>", msg.AttachmentCount)
			} else if msg.AttachmentCount > 0 {
				if msg.AttachmentCount == 1 {
					content = "<fichier>"
				} else {
					content = fmt.Sprintf("<%d fichiers>", msg.AttachmentCount)
				}
			} else if msg.HasEmbeds {
				content = "<embed>"
			} else {
				content = "<message vide>"
			}
		} else {
			// Truncate long messages
			if len(content) > 50 {
				content = content[:47] + "..."
			}
			// Add attachment indicator if message has both text and attachments
			if msg.AttachmentCount > 0 {
				if msg.AttachmentCount == 1 {
					content += " [+fichier]"
				} else {
					content += fmt.Sprintf(" [+%d fichiers]", msg.AttachmentCount)
				}
			}
		}

		line := fmt.Sprintf("%d. `%s` in <#%s>\n", idx+1, content, msg.ChannelID)

		if len(messagesText)+len(line) > maxRecentMessagesLength {
			messagesText += fmt.Sprintf("... et %d message(s) de plus", len(recentMessages)-idx)
			break
		}

		messagesText += line
	}

	if messagesText == "" {
		messagesText = "*Aucun message récent disponible*"
	}

	return messagesText
}

func (d *Discord) sendPrivateMessage(s *discordgo.Session, m *discordgo.Message, message string) {
	channel, err := s.UserChannelCreate(m.Author.ID)
	if err != nil {
		d.log.LogError(logger.LogModel{
			Database: d.db,
			GuildID:  m.GuildID,
			Function: "sendPrivateMessage()",
			Message:  fmt.Sprintf("Failed to create DM channel: %s", err),
		})
		return
	}

	_, err = s.ChannelMessageSend(channel.ID, message)
	if err != nil {
		d.log.LogError(logger.LogModel{
			Database: d.db,
			GuildID:  m.GuildID,
			Function: "sendPrivateMessage()",
			Message:  fmt.Sprintf("Failed to send DM: %s", err),
		})
	}
}

func (d *Discord) sendLogChannelsEmbed(s *discordgo.Session, guildID string, embed *discordgo.MessageEmbed) {
	logChannels, err := d.db.GetLogChannelsByGuildId(guildID)
	if err != nil {
		d.log.LogError(logger.LogModel{
			Database: d.db,
			GuildID:  guildID,
			Function: "sendLogChannelsEmbed()",
			Message:  fmt.Sprintf("Failed to fetch log channels: %s", err),
		})
		return
	}

	for _, channelID := range logChannels {
		_, err = s.ChannelMessageSendEmbed(channelID, embed)
		if err != nil {
			d.log.LogError(logger.LogModel{
				Database: d.db,
				GuildID:  guildID,
				Function: "sendLogChannelsEmbed()",
				Message:  fmt.Sprintf("Failed to send embed to channel %s: %s", channelID, err),
			})
		}
	}
}

func (d *Discord) sendModChannelsEmbed(s *discordgo.Session, guildID string, embed *discordgo.MessageEmbed) {
	modChannels, err := d.db.GetModerationChannelsByGuildId(guildID)
	if err != nil {
		d.log.LogError(logger.LogModel{
			Database: d.db,
			GuildID:  guildID,
			Function: "sendModChannelsEmbed()",
			Message:  fmt.Sprintf("Failed to fetch moderation channels: %s", err),
		})
		return
	}

	for _, channelID := range modChannels {
		_, err = s.ChannelMessageSendEmbed(channelID, embed)
		if err != nil {
			d.log.LogError(logger.LogModel{
				Database: d.db,
				GuildID:  guildID,
				Function: "sendModChannelsEmbed()",
				Message:  fmt.Sprintf("Failed to send embed to channel %s: %s", channelID, err),
			})
		}
	}
}

// logAutomoderationAction sends an alert to moderators and save it in DB
func (d *Discord) logAutomoderationAction(s *discordgo.Session, m *discordgo.Message, action model.ModerationLogAction, trigger string, reason string) {
	// Log in the database
	err := d.db.AddModerationLogEntry(m.GuildID, action, m.Author.ID, trigger, reason)
	if err != nil {
		d.log.LogError(logger.LogModel{Database: d.db, GuildID: m.GuildID, Function: "logAutomoderationAction()",
			Message: fmt.Sprintf("Error logging automod action to database: %s", err),
		})
	}

	// Get last 10 recent messages from the user
	recentMessages := d.getUserRecentMessagesString(m.GuildID, m.Author, 10)

	description := fmt.Sprintf(
		"**Utilisateur**: <@%s> (ID: %s)\n"+
			"**Action**: `%s`\n"+
			"**Triggered rule**: `%s`\n"+
			"**Raison**: %s\n\n"+
			"**Messages récents**: \n%s",
		m.Author.ID, m.Author.ID, action, trigger, reason, recentMessages,
	)

	embed := &discordgo.MessageEmbed{
		Title:       "🚨 Action d'automodération déclenchée",
		Description: description,
		Color:       model.Violet.Int(),
		Timestamp:   time.Now().Format(time.RFC3339),
	}

	// Send alert to all log channels
	d.sendLogChannelsEmbed(s, m.GuildID, embed)
}

func (d *Discord) takeAutomoderationAction(s *discordgo.Session, m *discordgo.Message, action model.ModerationLogAction, reason string) {
	switch action {
	case model.ActionDeleteMessage:
		// Delete the message
		err := s.ChannelMessageDelete(m.ChannelID, m.ID)
		if err != nil {
			d.log.LogError(logger.LogModel{
				Database: d.db,
				GuildID:  m.GuildID,
				Function: "checkBannedWords()",
				Message:  fmt.Sprintf("Failed to delete message: %s", err),
			})
			return
		}
	case model.ActionWarn:
		// Send a private warning message to the user
		d.sendPrivateMessage(s, m, reason)
		return
	case model.ActionKick:
		guild, err := s.Guild(m.GuildID)
		if err != nil {
			d.log.LogError(logger.LogModel{
				Database: d.db,
				GuildID:  m.GuildID,
				Function: "takeAutomoderationAction()",
				Message:  fmt.Sprintf("Failed to fetch guild info for kick message: %s", err),
			})
			guild.Name = "<unknown>"
		}
		message := fmt.Sprintf("Vous avez été expulsé du serveur [%s] pour la raison suivante : %s", guild.Name, reason)
		d.sendPrivateMessage(s, m, message)
		// Kick user
		err = s.GuildMemberDeleteWithReason(m.GuildID, m.Author.ID, reason)
		if err != nil {
			d.log.LogError(logger.LogModel{
				Database: d.db,
				GuildID:  m.GuildID,
				Function: "takeAutomoderationAction()",
				Message:  fmt.Sprintf("Failed to kick user %s: %s", m.Author.ID, err),
			})
			return
		}
	case model.ActionBan:
		guild, err := s.Guild(m.GuildID)
		if err != nil {
			d.log.LogError(logger.LogModel{
				Database: d.db,
				GuildID:  m.GuildID,
				Function: "takeAutomoderationAction()",
				Message:  fmt.Sprintf("Failed to fetch guild info for ban message: %s", err),
			})
			guild.Name = "<unknown>"
		}
		message := fmt.Sprintf("Vous avez été banni du serveur [%s] pour la raison suivante : %s", guild.Name, reason)
		d.sendPrivateMessage(s, m, message)
		// Permanent ban with message deletion
		err = s.GuildBanCreateWithReason(m.GuildID, m.Author.ID, reason, 1)
		if err != nil {
			d.log.LogError(logger.LogModel{
				Database: d.db,
				GuildID:  m.GuildID,
				Function: "takeAutomoderationAction()",
				Message:  fmt.Sprintf("Failed to ban user %s: %s", m.Author.ID, err),
			})
			return
		}
	}
}
