package api

import (
	"fmt"
	"regexp"
	"strings"
	"time"

	"github.com/bwmarrin/discordgo"
	"github.com/dragonis41/discord-bot-moderation/pkg/logger"
	"github.com/dragonis41/discord-bot-moderation/pkg/model"
)

func (d *Discord) messageCreateHandler(s *discordgo.Session, m *discordgo.MessageCreate) {
	// Ignore messages from bots and itself
	if m.Author != nil && m.Author.Bot || m.Author.ID == s.State.User.ID {
		return
	}
	// Ignore if this is a private message (DM)
	if m.GuildID == "" {
		return
	}
	// Ignore messages from moderators
	if d.db.UserHasModerationRole(m.GuildID, m.Member) {
		return
	}
	// TODO : Check if the message is sent in an excluded channel

	// Add message to cache
	d.cache.AddMessage(m)

	// Use the common moderation function
	d.moderateMessage(s, m.Message)
}

func (d *Discord) messageUpdateHandler(s *discordgo.Session, m *discordgo.MessageUpdate) {
	// Ignore messages from bots and itself
	if m.Author != nil && m.Author.Bot || m.Author.ID == s.State.User.ID {
		return
	}
	// Ignore if this is a private message (DM)
	if m.GuildID == "" {
		return
	}
	// Ignore messages from moderators
	if d.db.UserHasModerationRole(m.GuildID, m.Member) {
		return
	}
	// TODO : Check if the message is sent in an excluded channel

	// Update the message in cache
	d.cache.UpdateMessage(m)

	// Use the common moderation function
	d.moderateMessage(s, m.Message)
}

// Common function that handles moderation logic
func (d *Discord) moderateMessage(s *discordgo.Session, m *discordgo.Message) {
	d.checkBannedWords(s, m)
	d.checkMessageSpam(s, m)
	// TODO : Add more moderation functions
}

// checkBannedWords checks if the message contains any banned words
//
//	The check is case-insensitive and matches whole words / sentences.
//	It supports both literal words and regex patterns (prefixed with "regex:").
//	It also triggers if the word is surrounded by whitespace or punctuation.
func (d *Discord) checkBannedWords(s *discordgo.Session, m *discordgo.Message) {
	// TODO: Get this list from database
	bannedWords := []string{"badword1", "badword2", "spam", `regex:spam[0-9]+`}

	messageLower := strings.ToLower(m.Content)

	for _, word := range bannedWords {
		var pattern string
		// Create regex pattern that matches the word only if it's surrounded by:
		// - start/end of string
		// - whitespace
		// - punctuation
		if strings.HasPrefix(word, "regex:") {
			// If the word starts with "regex:", treat the rest as a regex pattern
			regexPattern := strings.TrimPrefix(word, "regex:")
			pattern = fmt.Sprintf(`(?i)(^|[\s\p{P}])(%s)($|[\s\p{P}])`, regexPattern)
		} else {
			// Otherwise, escape the word to treat it as a literal string
			pattern = fmt.Sprintf(`(?i)(^|[\s\p{P}])%s($|[\s\p{P}])`, regexp.QuoteMeta(word))
		}
		matched, err := regexp.MatchString(pattern, messageLower)

		if err != nil {
			d.log.LogError(logger.LogModel{
				Database: d.db,
				GuildID:  m.GuildID,
				Function: "checkBannedWords()",
				Message:  fmt.Sprintf("Regex error for word '%s': %s", word, err),
			})
			continue
		}

		if matched {
			// Increment violation count
			violationCount := d.cache.IncrementViolation(m.GuildID, m.Author.ID)

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

			// Check if threshold is reached
			if violationCount >= d.cache.violationThreshold {
				d.logAutomoderationAction(s, m, word, "Banned Words")
				// TODO : Log this and take further actions (mute, kick, ban, etc.)
			}

			// Send a warning to the user
			guildName := "<unknown>"
			guild, err := d.client.Guild(m.GuildID)
			if err != nil {
				d.log.LogError(logger.LogModel{
					Database: d.db,
					GuildID:  m.GuildID,
					Function: "checkBannedWords()",
					Message:  fmt.Sprintf("Failed to fetch guild info: %s", err),
				})
			}
			if guild != nil {
				guildName = guild.Name
			}

			warningMessage := fmt.Sprintf(
				"⚠️ **Avertissement %d/%d**\n"+
					"Votre message sur le serveur [%s] a été supprimé car il contient le mot `%s`\n\n"+
					"Voici une copie de votre message :\n```\n%s\n```",
				violationCount, d.cache.violationThreshold, guildName, word, m.Content)

			if violationCount >= d.cache.violationThreshold {
				warningMessage += "\n🚨 **Les modérateurs ont été alertés.**"
			}
			d.sendPrivateMessage(s, m, warningMessage)

			d.log.LogInfo(logger.LogModel{
				Database: d.db,
				GuildID:  m.GuildID,
				Function: "sendPrivateMessage()",
				Message: fmt.Sprintf("Deleted message from %s containing banned word: %s (violation %d/%d)",
					m.Author.Username, word, violationCount, d.cache.violationThreshold),
			})

			return
		}
	}
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

// checkMessageSpam detects if a user is sending the exact same message repeatedly (in a channel or across multiples channels)
func (d *Discord) checkMessageSpam(s *discordgo.Session, m *discordgo.Message) {
	// Get recent messages from this user
	recentMessages := d.cache.GetUserRecentMessages(m.GuildID, m.Author.ID, d.cache.GetMaxCacheSize())

	if len(recentMessages) < d.cache.GetViolationThreshold() {
		return
	}

	// Count how many times this exact message appears within the violation window
	duplicateCount := 0
	now := time.Now()

	for _, msg := range recentMessages {
		// Check if message is within the violation window
		if now.Sub(msg.Timestamp) <= d.cache.GetViolationWindow() {
			// Exact match comparison
			if msg.Content == m.Content {
				duplicateCount++
			}
		}
	}

	// If threshold is reached, ban the user
	if duplicateCount >= d.cache.GetViolationThreshold() {
		// Permanent ban with message deletion
		err := s.GuildBanCreateWithReason(m.GuildID, m.Author.ID, fmt.Sprintf("Spam (message répété %d fois en %s)", duplicateCount, d.cache.GetViolationWindow()), 1)
		if err != nil {
			d.log.LogError(logger.LogModel{
				Database: d.db,
				GuildID:  m.GuildID,
				Function: "checkMessageSpam()",
				Message:  fmt.Sprintf("Failed to ban user %s: %s", m.Author.ID, err),
			})
			return
		}

		// Log the action to moderation channels
		d.logSpamBan(s, m, duplicateCount)

		d.log.LogSuccess(logger.LogModel{
			Database: d.db,
			GuildID:  m.GuildID,
			Function: "checkMessageSpam()",
			Message:  fmt.Sprintf("Banned user %s for spam (%d duplicate messages)", m.Author.Username, duplicateCount),
		})
	}
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
		// Truncate individual message content if too long
		content := msg.Content
		if len(content) > 100 {
			content = content[:97] + "..."
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
