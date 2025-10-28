package api

import (
	"fmt"
	"time"

	"github.com/bwmarrin/discordgo"
	"github.com/dragonis41/discord-bot-moderation/pkg/logger"
	"github.com/dragonis41/discord-bot-moderation/pkg/model"
)

type DiscordUtilsInterface interface {
	sendPrivateMessage(s *discordgo.Session, m *discordgo.Message, message string)
	splitMessage(message string, limit int) []string
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

// splitMessage splits a long message into multiple parts based on the limit
func (d *Discord) splitMessage(message string, limit int) []string {
	if len(message) <= limit {
		return []string{message}
	}

	var messages []string
	runes := []rune(message)

	for len(runes) > 0 {
		if len(runes) <= limit {
			messages = append(messages, string(runes))
			break
		}

		splitIndex := limit

		// Try to find a good split point (newline, space, or punctuation)
		for i := limit - 1; i > limit-200 && i > 0; i-- {
			if runes[i] == '\n' || runes[i] == ' ' || runes[i] == '.' || runes[i] == ',' {
				splitIndex = i + 1
				break
			}
		}

		messages = append(messages, string(runes[:splitIndex]))
		runes = runes[splitIndex:]
	}

	return messages
}

// logSpamBan sends an alert to moderators about the spam ban
// TODO : This function is temporary and will be replaced by a proper logging system
func (d *Discord) logSpamBan(s *discordgo.Session, m *discordgo.Message, duplicateCount int) {
	logChannels, err := d.db.GetLogChannelsByGuildId(m.GuildID)
	if err != nil {
		d.log.LogError(logger.LogModel{
			Database: d.db,
			GuildID:  m.GuildID,
			Function: "logSpamBan()",
			Message:  fmt.Sprintf("Failed to fetch log channels: %s", err),
		})
		return
	}

	if len(logChannels) == 0 {
		return
	}

	// Get moderator roles to ping
	modRoles, err := d.db.GetModerationRolesByGuildId(m.GuildID)
	if err != nil {
		d.log.LogError(logger.LogModel{
			Database: d.db,
			GuildID:  m.GuildID,
			Function: "logSpamBan()",
			Message:  fmt.Sprintf("Failed to fetch moderation roles: %s", err),
		})
	}

	modRoleMentions := ""
	for _, roleID := range modRoles {
		modRoleMentions += fmt.Sprintf("<@&%s> ", roleID)
	}

	// Truncate message content if too long for embed
	messageContent := m.Content
	if len(messageContent) > 500 {
		messageContent = messageContent[:497] + "..."
	}

	description := fmt.Sprintf(
		"**Utilisateur**: <@%s> (ID: %s)\n"+
			"**Action**: Ban automatique\n"+
			"**Raison**: Spam (message répété %d fois en %s)\n"+
			"**Message répété**:\n```\n%s\n```",
		m.Author.ID,
		m.Author.ID,
		duplicateCount,
		d.cache.GetViolationWindow(),
		messageContent,
	)

	// Send alert to all moderation channels
	for _, channelID := range logChannels {
		_, err = s.ChannelMessageSendComplex(channelID, &discordgo.MessageSend{
			Content: modRoleMentions,
			Embed: &discordgo.MessageEmbed{
				Title:       "🚨 Spam détecté",
				Description: description,
				Color:       model.Red.Int(),
				Timestamp:   time.Now().Format(time.RFC3339),
			},
		})
		if err != nil {
			d.log.LogError(logger.LogModel{
				Database: d.db,
				GuildID:  m.GuildID,
				Function: "logSpamBan()",
				Message:  fmt.Sprintf("Failed to send alert to channel %s: %s", channelID, err),
			})
		}
	}
}

// logAutomoderationAction sends an alert to moderators about repeated violations
// TODO : This function is temporary and will be replaced by a proper logging system
func (d *Discord) logAutomoderationAction(s *discordgo.Session, m *discordgo.Message, bannedWord string, triggeredRule string) {
	logChannels, err := d.db.GetLogChannelsByGuildId(m.GuildID)
	if err != nil {
		d.log.LogError(logger.LogModel{
			Database: d.db,
			GuildID:  m.GuildID,
			Function: "logAutomoderationAction()",
			Message:  fmt.Sprintf("Failed to fetch log channels: %s", err),
		})
		return
	}
	if len(logChannels) == 0 {
		d.log.LogWarning(logger.LogModel{
			Database: d.db,
			GuildID:  m.GuildID,
			Function: "logAutomoderationAction()",
			Message:  "No moderation channels configured to send alerts",
		})
		return
	}

	// Get moderator roles to ping
	modRoles, err := d.db.GetModerationRolesByGuildId(m.GuildID)
	if err != nil {
		d.log.LogError(logger.LogModel{
			Database: d.db,
			GuildID:  m.GuildID,
			Function: "logAutomoderationAction()",
			Message:  fmt.Sprintf("Failed to fetch moderation roles: %s", err),
		})
	}

	modRoleMentions := ""
	for _, roleID := range modRoles {
		modRoleMentions += fmt.Sprintf("<@&%s> ", roleID)
	}

	// Get recent messages from this user
	recentMessages := d.cache.GetUserRecentMessages(m.GuildID, m.Author.ID, 10)
	messagesText := ""
	const maxRecentMessagesLength = 800 // Leave room for other content in the description

	for i, msg := range recentMessages {
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

		line := fmt.Sprintf("%d. `%s` in <#%s>\n", i+1, content, msg.ChannelID)

		// Check if adding this line would exceed the limit
		if len(messagesText)+len(line) > maxRecentMessagesLength {
			messagesText += fmt.Sprintf("... et %d message(s) de plus", len(recentMessages)-i)
			break
		}

		messagesText += line
	}

	// If no messages were added, show a placeholder
	if messagesText == "" {
		messagesText = "*Aucun message récent disponible*"
	}

	// Build description with length check
	description := fmt.Sprintf(
		"**Utilisateur**: <@%s> (ID: %s)\n"+
			"**Violations**: %d en %s\n"+
			"**Triggered rule**: `%s`\n"+
			"**Dernier mot interdit**: `%s`\n\n"+
			"**Messages récents**:\n%s",
		m.Author.ID,
		m.Author.ID,
		d.cache.violationThreshold,
		d.cache.violationWindow,
		triggeredRule,
		bannedWord,
		messagesText,
	)

	// Truncate description if it exceeds Discord's limit (4096 chars)
	const maxDescriptionLength = 4096
	if len(description) > maxDescriptionLength {
		description = description[:maxDescriptionLength-3] + "..."
	}

	// Send alert to all moderation channels
	for _, channelID := range logChannels {
		_, err = s.ChannelMessageSendComplex(channelID, &discordgo.MessageSend{
			Content: modRoleMentions,
			Embed: &discordgo.MessageEmbed{
				Title:       "🚨 Alerte : Violations répétées",
				Description: description,
				Color:       model.Red.Int(),
				Timestamp:   time.Now().Format(time.RFC3339),
			},
		})
		if err != nil {
			d.log.LogError(logger.LogModel{
				Database: d.db,
				GuildID:  m.GuildID,
				Function: "logAutomoderationAction()",
				Message:  fmt.Sprintf("Failed to send alert to channel %s: %s", channelID, err),
			})
		}
	}
}
