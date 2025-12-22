package api

import (
	"fmt"
	"regexp"
	"strings"
	"time"

	"github.com/bwmarrin/discordgo"
	"github.com/dragonis41/discord-bot-moderation/pkg/logger"
	"github.com/dragonis41/discord-bot-moderation/pkg/model"
	"github.com/dragonis41/discord-bot-moderation/pkg/utils"
)

// messageCreateHandler handles new messages created in guilds
//
//	It ignores messages from bots and itself, as well as private messages (DMs).
//	It adds the message to cache and calls the common moderation function.
func (d *Discord) messageCreateHandler(s *discordgo.Session, m *discordgo.MessageCreate) {
	// Ignore messages from bots and itself
	if m.Author != nil && m.Author.Bot || m.Author.ID == s.State.User.ID {
		return
	}
	// Ignore if this is a private message (DM)
	if m.GuildID == "" {
		return
	}

	// Add message to cache
	d.cache.AddMessage(m)
	// Use the common moderation function
	d.moderateMessage(s, m.Message)
}

// messageUpdateHandler handles edited messages in guilds
//
//	It ignores messages from bots and itself, as well as private messages (DMs).
//	It updates the message in cache and calls the common moderation function.
func (d *Discord) messageUpdateHandler(s *discordgo.Session, m *discordgo.MessageUpdate) {
	// Ignore messages from bots and itself
	if m.Author != nil && m.Author.Bot || m.Author.ID == s.State.User.ID {
		return
	}
	// Ignore if this is a private message (DM)
	if m.GuildID == "" {
		return
	}

	// Update the message in cache
	d.cache.UpdateMessage(m)
	// Use the common moderation function
	d.moderateMessage(s, m.Message)
}

// Common function that handles moderation logic
//
//	This function is called by both message create and update handlers.
//	It is used to call the various moderation checks.
//
//	It ignores messages from moderators and messages sent in excluded channels. The checks are here because we still want to update the cache for those messages.
func (d *Discord) moderateMessage(s *discordgo.Session, m *discordgo.Message) {
	// Ignore messages from moderators
	if d.db.UserHasModerationRole(m.GuildID, m.Member) {
		return
	}
	// Ignore if the message is sent in an excluded channel
	if d.db.IsExcludedChannel(m.GuildID, m.ChannelID) {
		return
	}

	if d.checkMessageSpam(s, m) {
		return // Stop processing if spam detected
	}
	if d.checkBannedWords(s, m) {
		return // Stop processing if banned word detected
	}
	if d.checkBannedWebsites(s, m) {
		return // Stop processing if banned website detected
	}
}

// checkBannedWords checks if the message contains any banned words
//
//	The check is case-insensitive and matches whole words / sentences.
//	It supports both literal words and regex patterns.
//	It also triggers if the word / sentence is surrounded by whitespace or punctuation.
func (d *Discord) checkBannedWords(s *discordgo.Session, m *discordgo.Message) bool {
	// Check if banned words feature is enabled
	settings, err := d.db.GetAutomoderationSettings(m.GuildID)
	if err != nil {
		d.log.LogError(logger.LogModel{
			Database: d.db,
			GuildID:  m.GuildID,
			Function: "checkBannedWords()",
			Message:  fmt.Sprintf("Failed to fetch automoderation settings: %s", err),
		})
		return false
	}

	if !settings.BannedWordsEnabled {
		return false
	}

	// Get banned words from database
	bannedWords, err := d.db.GetBannedWordsByGuildId(m.GuildID)
	if err != nil {
		d.log.LogError(logger.LogModel{
			Database: d.db,
			GuildID:  m.GuildID,
			Function: "checkBannedWords()",
			Message:  fmt.Sprintf("Failed to fetch banned words from database: %s", err),
		})
		return false
	}

	// Skip if no banned words are configured
	if len(bannedWords) == 0 {
		return false
	}

	messageLower := strings.ToLower(m.Content)

	for _, bannedWord := range bannedWords {
		var pattern string
		// Create regex pattern that matches the word only if it's surrounded by:
		// - start/end of string
		// - whitespace
		// - punctuation

		// If IsRegex is true, treat the word pattern as a regex
		word := bannedWord.WordPattern
		if !bannedWord.IsRegex {
			// Otherwise, escape the word to treat it as a literal string
			word = regexp.QuoteMeta(bannedWord.WordPattern)
		}
		pattern = fmt.Sprintf(`(?i)(?:^|[\s\p{Z}\p{Cf}\p{P}])(%s)(?:$|[\s\p{Z}\p{Cf}\p{P}])`, word)
		matched, err := regexp.MatchString(pattern, messageLower)
		if err != nil {
			d.log.LogError(logger.LogModel{
				Database: d.db,
				GuildID:  m.GuildID,
				Function: "checkBannedWords()",
				Message:  fmt.Sprintf("Regex error for word '%s': %s", bannedWord.WordPattern, err),
			})
			continue
		}

		if matched {
			// Increment violation count
			violationCount := d.cache.IncrementViolation(m.GuildID, m.Author.ID)

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
					"Voici une copie de votre message :\n",
				violationCount, d.cache.violationThreshold, guildName, bannedWord.WordPattern)

			d.sendPrivateMessage(s, m, warningMessage)

			message := utils.SplitMessage(m.Content, 1900)
			for _, msgPart := range message {
				d.sendPrivateMessage(s, m, fmt.Sprintf("```\n%s\n```", msgPart))
			}

			if violationCount < d.cache.violationThreshold {
				// Log and delete the message
				d.logAutomoderationAction(s, m, model.ActionDeleteMessage, "banned_word_check", fmt.Sprintf("Le mot `%s` est banni", bannedWord.WordPattern))
				d.takeAutomoderationAction(s, m, model.ActionDeleteMessage, fmt.Sprintf("Le mot `%s` est banni (violation %d/%d)", bannedWord.WordPattern, violationCount, d.cache.violationThreshold))
			}

			// Check if threshold is reached
			if violationCount >= d.cache.violationThreshold {
				// Ban the user
				d.logAutomoderationAction(s, m, model.ActionBan, "banned_word_check", fmt.Sprintf("Le mot `%s` est banni", bannedWord.WordPattern))
				d.takeAutomoderationAction(s, m, model.ActionBan, fmt.Sprintf("Le mot `%s` est banni\n%d automod violations", bannedWord.WordPattern, violationCount))
			}

			d.log.LogInfo(logger.LogModel{
				Database: d.db,
				GuildID:  m.GuildID,
				Function: "sendPrivateMessage()",
				Message: fmt.Sprintf("Deleted message from %s containing banned word: [%s] (violation %d/%d)",
					m.Author.Username, bannedWord.WordPattern, violationCount, d.cache.violationThreshold),
			})

			return true
		}
	}

	return false
}

// checkBannedWebsites checks if the message contains any banned website URLs
//
//	The check extracts URLs from the message and compares them against the banned websites list.
//	It matches if the URL contains the banned website pattern.
func (d *Discord) checkBannedWebsites(s *discordgo.Session, m *discordgo.Message) bool {
	// Check if banned websites feature is enabled
	settings, err := d.db.GetAutomoderationSettings(m.GuildID)
	if err != nil {
		d.log.LogError(logger.LogModel{
			Database: d.db,
			GuildID:  m.GuildID,
			Function: "checkBannedWebsites()",
			Message:  fmt.Sprintf("Failed to fetch automoderation settings: %s", err),
		})
		return false
	}

	if !settings.BannedWebsitesEnabled {
		return false
	}

	// Get banned websites from database
	bannedWebsites, err := d.db.GetBannedWebsitesByGuildId(m.GuildID)
	if err != nil {
		d.log.LogError(logger.LogModel{
			Database: d.db,
			GuildID:  m.GuildID,
			Function: "checkBannedWebsites()",
			Message:  fmt.Sprintf("Failed to fetch banned websites from database: %s", err),
		})
		return false
	}

	// Skip if no banned websites are configured
	if len(bannedWebsites) == 0 {
		return false
	}

	// Regex pattern to extract URLs from message
	// This matches http://, https://, and bare domains
	urlPattern := regexp.MustCompile(`(?i)(?:https?://)?(?:www\.)?([a-zA-Z0-9][-a-zA-Z0-9]*\.)+[a-zA-Z]{2,}(?:/[^\s]*)?`)
	urls := urlPattern.FindAllString(m.Content, -1)

	// Skip if no URLs found in message
	if len(urls) == 0 {
		return false
	}

	messageLower := strings.ToLower(m.Content)

	// Check each banned website against the message
	for _, bannedWebsite := range bannedWebsites {
		websiteLower := strings.ToLower(bannedWebsite.WebsiteURL)

		// Check if the banned website pattern appears in the message
		if strings.Contains(messageLower, websiteLower) {
			// Increment violation count
			violationCount := d.cache.IncrementViolation(m.GuildID, m.Author.ID)

			// Send a warning to the user
			guildName := "<unknown>"
			guild, err := d.client.Guild(m.GuildID)
			if err != nil {
				d.log.LogError(logger.LogModel{
					Database: d.db,
					GuildID:  m.GuildID,
					Function: "checkBannedWebsites()",
					Message:  fmt.Sprintf("Failed to fetch guild info: %s", err),
				})
			}
			if guild != nil {
				guildName = guild.Name
			}

			d.logAutomoderationAction(s, m, model.ActionWarn, "banned_website_check", fmt.Sprintf("Site banni: `%s`", bannedWebsite.WebsiteURL))
			warningMessage := fmt.Sprintf(
				"⚠️ **Avertissement %d/%d**\n"+
					"Votre message sur le serveur [%s] a été supprimé car il contient un lien interdit : `%s`\n\n"+
					"Voici une copie de votre message :\n",
				violationCount, d.cache.violationThreshold, guildName, bannedWebsite.WebsiteURL)
			d.takeAutomoderationAction(s, m, model.ActionWarn, warningMessage)

			// Send a copy of the deleted message
			message := utils.SplitMessage(m.Content, 1900)
			for _, msgPart := range message {
				d.sendPrivateMessage(s, m, fmt.Sprintf("```\n%s\n```", msgPart))
			}

			if violationCount < d.cache.violationThreshold {
				// Log and delete the message
				d.logAutomoderationAction(s, m, model.ActionDeleteMessage, "banned_website_check", fmt.Sprintf("Le site `%s` est banni", bannedWebsite.WebsiteURL))
				d.takeAutomoderationAction(s, m, model.ActionDeleteMessage, fmt.Sprintf("Le site `%s` est banni (violation %d/%d)", bannedWebsite.WebsiteURL, violationCount, d.cache.violationThreshold))
			}

			// Check if threshold is reached
			if violationCount >= d.cache.violationThreshold {
				// Ban the user
				d.logAutomoderationAction(s, m, model.ActionBan, "banned_website_check", fmt.Sprintf("Le site `%s` est banni", bannedWebsite.WebsiteURL))
				d.takeAutomoderationAction(s, m, model.ActionBan, fmt.Sprintf("Le site `%s` est banni\n%d automod violations", bannedWebsite.WebsiteURL, violationCount))
			}

			d.log.LogInfo(logger.LogModel{
				Database: d.db,
				GuildID:  m.GuildID,
				Function: "checkBannedWebsites()",
				Message: fmt.Sprintf("Deleted message from %s containing banned website: [%s] (violation %d/%d)",
					m.Author.Username, bannedWebsite.WebsiteURL, violationCount, d.cache.violationThreshold),
			})

			return true
		}
	}

	return false
}

// checkMessageSpam detects if a user is sending the exact same message repeatedly across multiple channels
func (d *Discord) checkMessageSpam(s *discordgo.Session, m *discordgo.Message) bool {
	// Check if spam detection feature is enabled
	settings, err := d.db.GetAutomoderationSettings(m.GuildID)
	if err != nil {
		d.log.LogError(logger.LogModel{
			Database: d.db,
			GuildID:  m.GuildID,
			Function: "checkMessageSpam()",
			Message:  fmt.Sprintf("Failed to fetch automoderation settings: %s", err),
		})
		return false
	}

	if !settings.SpamDetectionEnabled {
		return false
	}

	// Get recent messages from this user
	recentMessages := d.cache.GetUserRecentMessages(m.GuildID, m.Author.ID, d.cache.GetMaxCacheSize())

	if len(recentMessages) < d.cache.GetViolationThreshold() {
		return false
	}

	// Count how many times this exact message appears within the violation window and track which channels they were sent to
	duplicateCount := 0
	channelsUsed := make(map[string]bool)
	now := time.Now()

	for _, msg := range recentMessages {
		// Check if message is within the violation window
		if now.Sub(msg.Timestamp) <= d.cache.GetViolationWindow() {
			// Exact match comparison
			if msg.Content == m.Content && msg.AttachmentCount == len(m.Attachments) {
				duplicateCount++
				channelsUsed[msg.ChannelID] = true
			}
		}
	}

	// Only consider it spam if:
	// 1. The threshold is reached
	// 2. The messages were sent across multiple channels (more than x unique channel)
	if duplicateCount >= d.cache.GetViolationThreshold() && len(channelsUsed) >= d.cache.GetViolationThreshold() {
		// Log and ban the user
		d.logAutomoderationAction(s, m, model.ActionBan, "spam_detection", fmt.Sprintf("Spam (message répété %d fois dans %d salons en %s)", duplicateCount, len(channelsUsed), d.cache.GetViolationWindow()))
		d.takeAutomoderationAction(s, m, model.ActionBan, fmt.Sprintf("Spam (message répété %d fois dans %d salons en %s)", duplicateCount, len(channelsUsed), d.cache.GetViolationWindow()))

		d.log.LogSuccess(logger.LogModel{
			Database: d.db,
			GuildID:  m.GuildID,
			Function: "checkMessageSpam()",
			Message:  fmt.Sprintf("Banned user %s for cross-channel spam (%d duplicate messages across %d channels)", m.Author.Username, duplicateCount, len(channelsUsed)),
		})

		return true
	}

	return false
}
