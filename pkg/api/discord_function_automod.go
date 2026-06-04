package api

import (
	"fmt"
	"regexp"
	"strings"
	"time"

	"github.com/bwmarrin/discordgo"
	"github.com/dragonis41/discord-bot-moderation/pkg/model"
	"github.com/dragonis41/discord-bot-moderation/pkg/utils"
)

// messageCreateHandler handles new messages created in guilds
//
//	It ignores messages from bots and itself, as well as private messages (DMs).
//	It adds the message to cache and calls the common moderation function.
func (d *Discord) messageCreateHandler(s *discordgo.Session, m *discordgo.MessageCreate) {
	// Ignore if Author is nil
	if m == nil || s == nil || m.Author == nil || m.Member == nil || s.State == nil {
		return
	}
	// Ignore messages from bots and itself
	if m.Author.Bot || m.Author.ID == s.State.User.ID {
		return
	}
	// Ignore webhook messages
	if m.WebhookID != "" {
		return
	}
	// Ignore if this is a private message (DM)
	if m.GuildID == "" {
		return
	}
	// Ignore system messages (thread starters, pins, join notifications, etc.)
	if m.Type != discordgo.MessageTypeDefault && m.Type != discordgo.MessageTypeReply {
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
	// Ignore if Author is nil (can happen with some message update events)
	// Also check if Message is nil - update events can have incomplete data
	if m == nil || s == nil || m.Message == nil || m.Author == nil || m.Member == nil || s.State == nil {
		return
	}
	// Ignore messages from bots and itself
	if m.Author.Bot || m.Author.ID == s.State.User.ID {
		return
	}
	// Ignore webhook messages (webhooks like news bots can trigger update events)
	if m.WebhookID != "" {
		return
	}
	// Ignore if this is a private message (DM)
	if m.GuildID == "" {
		return
	}
	// Ignore update events that don't contain actual content (e.g., embed updates, reactions)
	// These events fire when Discord loads link previews or other metadata changes
	if m.Content == "" && len(m.Attachments) == 0 {
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
	// Safety check: ensure database is available
	if d.db == nil {
		return
	}
	// Ignore if user is already pending ban to prevent duplicate actions
	if d.cache.IsUserPendingBan(m.GuildID, m.Author.ID) {
		return
	}
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

// websiteURLPattern matches http(s):// links and bare domains. It is compiled
// once at package load instead of on every message.
var websiteURLPattern = regexp.MustCompile(`(?i)(?:https?://)?(?:www\.)?([a-zA-Z0-9][-a-zA-Z0-9]*\.)+[a-zA-Z]{2,}(?:/[^\s]*)?`)

// escalateContentViolation applies the shared warn → delete → ban escalation
// used whenever a message breaks a content rule (a banned word or website).
//
//	It warns the author by DM with a copy of their (soon to be deleted) message,
//	then either deletes the message (below the violation threshold) or bans the
//	author (at or above it). It returns true once handled so the caller stops
//	running further checks.
//
//	cause completes the sentence "...a été supprimé car {cause}", ruleReason is
//	the reason logged/sent for the action, and logSubject describes the match in
//	the internal info log.
func (d *Discord) escalateContentViolation(s *discordgo.Session, m *discordgo.Message, checkName, cause, ruleReason, logSubject string) bool {
	// Skip if the user is already being banned by a concurrent handler.
	if d.cache.IsUserPendingBan(m.GuildID, m.Author.ID) {
		return true
	}

	violationCount := d.cache.IncrementViolation(m.GuildID, m.Author.ID)
	guildName := guildNameByID(s, m.GuildID)

	// Warn the user and send back a copy of their message.
	warning := fmt.Sprintf(
		"⚠️ **Avertissement %d/%d**\n"+
			"Votre message sur le serveur [%s] a été supprimé car %s\n\n"+
			"Voici une copie de votre message :\n",
		violationCount, d.cache.violationThreshold, guildName, cause)
	d.sendPrivateMessage(s, m, warning)
	for _, part := range utils.SplitMessage(m.Content, 1900) {
		d.sendPrivateMessage(s, m, fmt.Sprintf("```\n%s\n```", part))
	}

	if violationCount < d.cache.violationThreshold {
		// Below threshold: log and delete the message.
		d.logAutomoderationAction(s, m, model.ActionDeleteMessage, checkName, ruleReason)
		d.takeAutomoderationAction(s, m, model.ActionDeleteMessage, fmt.Sprintf("%s (violation %d/%d)", ruleReason, violationCount, d.cache.violationThreshold))
	} else {
		// Threshold reached: ban the user, guarding against concurrent bans.
		if !d.cache.MarkUserAsPendingBan(m.GuildID, m.Author.ID) {
			return true
		}
		d.logAutomoderationAction(s, m, model.ActionBan, checkName, ruleReason)
		d.takeAutomoderationAction(s, m, model.ActionBan, fmt.Sprintf("%s\n%d automod violations", ruleReason, violationCount))
	}

	d.logInfo(m.GuildID, checkName, "Deleted message from %s containing %s (violation %d/%d)",
		m.Author.Username, logSubject, violationCount, d.cache.violationThreshold)
	return true
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
		d.logError(m.GuildID, "checkBannedWords()", "Failed to fetch automoderation settings: %s", err)
		return false
	}
	if !settings.BannedWordsEnabled {
		return false
	}

	// Get banned words from database
	bannedWords, err := d.db.GetBannedWordsByGuildId(m.GuildID)
	if err != nil {
		d.logError(m.GuildID, "checkBannedWords()", "Failed to fetch banned words from database: %s", err)
		return false
	}
	if len(bannedWords) == 0 {
		return false
	}

	messageLower := strings.ToLower(m.Content)

	for _, bannedWord := range bannedWords {
		// Match the word only when bounded by start/end of string, whitespace or
		// punctuation. Literal words are escaped; regex patterns are used as-is.
		word := bannedWord.WordPattern
		if !bannedWord.IsRegex {
			word = regexp.QuoteMeta(bannedWord.WordPattern)
		}
		pattern := fmt.Sprintf(`(?i)(?:^|[\s\p{Z}\p{Cf}\p{P}])(%s)(?:$|[\s\p{Z}\p{Cf}\p{P}])`, word)
		matched, err := regexp.MatchString(pattern, messageLower)
		if err != nil {
			d.logError(m.GuildID, "checkBannedWords()", "Regex error for word '%s': %s", bannedWord.WordPattern, err)
			continue
		}

		if matched {
			return d.escalateContentViolation(s, m, "banned_word_check",
				fmt.Sprintf("il contient le mot `%s`", bannedWord.WordPattern),
				fmt.Sprintf("Le mot `%s` est banni", bannedWord.WordPattern),
				fmt.Sprintf("banned word: [%s]", bannedWord.WordPattern))
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
		d.logError(m.GuildID, "checkBannedWebsites()", "Failed to fetch automoderation settings: %s", err)
		return false
	}
	if !settings.BannedWebsitesEnabled {
		return false
	}

	// Get banned websites from database
	bannedWebsites, err := d.db.GetBannedWebsitesByGuildId(m.GuildID)
	if err != nil {
		d.logError(m.GuildID, "checkBannedWebsites()", "Failed to fetch banned websites from database: %s", err)
		return false
	}
	if len(bannedWebsites) == 0 {
		return false
	}

	// Skip messages that contain no URL-like token at all.
	if len(websiteURLPattern.FindAllString(m.Content, -1)) == 0 {
		return false
	}

	messageLower := strings.ToLower(m.Content)
	for _, bannedWebsite := range bannedWebsites {
		if strings.Contains(messageLower, strings.ToLower(bannedWebsite.WebsiteURL)) {
			return d.escalateContentViolation(s, m, "banned_website_check",
				fmt.Sprintf("il contient un lien interdit : `%s`", bannedWebsite.WebsiteURL),
				fmt.Sprintf("Le site `%s` est banni", bannedWebsite.WebsiteURL),
				fmt.Sprintf("banned website: [%s]", bannedWebsite.WebsiteURL))
		}
	}

	return false
}

// checkMessageSpam detects if a user is sending the exact same message repeatedly across multiple channels
func (d *Discord) checkMessageSpam(s *discordgo.Session, m *discordgo.Message) bool {
	// Check if spam detection feature is enabled
	settings, err := d.db.GetAutomoderationSettings(m.GuildID)
	if err != nil {
		d.logError(m.GuildID, "checkMessageSpam()", "Failed to fetch automoderation settings: %s", err)
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
			if msg.Content == m.Content && msg.AttachmentCount == len(m.Attachments) && stickerIDsFromCache(msg.Stickers) == stickerIDsFromItems(m.StickerItems) {
				duplicateCount++
				channelsUsed[msg.ChannelID] = true
			}
		}
	}

	// Only consider it spam if:
	// 1. The threshold is reached
	// 2. The messages were sent across multiple channels (more than x unique channel)
	if duplicateCount >= d.cache.GetViolationThreshold() && len(channelsUsed) >= d.cache.GetViolationThreshold() {
		// Mark user as pending ban to prevent duplicate actions from concurrent handlers
		if !d.cache.MarkUserAsPendingBan(m.GuildID, m.Author.ID) {
			// User is already being banned, skip
			return true
		}
		// Log and ban the user
		d.logAutomoderationAction(s, m, model.ActionBan, "spam_detection", fmt.Sprintf("Spam (message répété %d fois dans %d salons en %s)", duplicateCount, len(channelsUsed), d.cache.GetViolationWindow()))
		d.takeAutomoderationAction(s, m, model.ActionBan, fmt.Sprintf("Spam (message répété %d fois dans %d salons en %s)", duplicateCount, len(channelsUsed), d.cache.GetViolationWindow()))

		d.logSuccess(m.GuildID, "checkMessageSpam()", "Banned user %s for cross-channel spam (%d duplicate messages across %d channels)", m.Author.Username, duplicateCount, len(channelsUsed))

		return true
	}

	return false
}
