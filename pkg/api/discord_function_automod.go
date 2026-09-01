package api

import (
	"fmt"
	"regexp"
	"strings"
	"time"

	"github.com/bwmarrin/discordgo"
	"github.com/dragonis41/discord-bot-moderation/internal/database"
	"github.com/dragonis41/discord-bot-moderation/pkg/i18n"
	"github.com/dragonis41/discord-bot-moderation/pkg/model"
	"github.com/dragonis41/discord-bot-moderation/pkg/utils"
)

// shouldIgnoreMessage returns true if the message should be ignored by the moderation handlers.
//
//	It checks for nil pointers, bot/self messages, webhooks, and DMs.
func shouldIgnoreMessage(s *discordgo.Session, m *discordgo.Message) bool {
	if s == nil || s.State == nil || m == nil || m.Author == nil || m.Member == nil {
		return true
	}
	// Ignore messages from bots and itself
	if m.Author.Bot || m.Author.ID == s.State.User.ID {
		return true
	}
	// Ignore webhook messages
	if m.WebhookID != "" {
		return true
	}
	// Ignore if this is a private message (DM)
	if m.GuildID == "" {
		return true
	}
	return false
}

// messageCreateHandler handles new messages created in guilds
//
//	It ignores messages from bots and itself, as well as private messages (DMs).
//	It adds the message to cache and calls the common moderation function.
func (d *Discord) messageCreateHandler(s *discordgo.Session, m *discordgo.MessageCreate) {
	if m == nil || m.Message == nil || shouldIgnoreMessage(s, m.Message) {
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
	if m == nil || m.Message == nil || shouldIgnoreMessage(s, m.Message) {
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
func (d *Discord) moderateMessage(s discordSender, m *discordgo.Message) {
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

// Pure matching helpers ------------------------------------------------------
//
// These hold the actual content-rule logic, free of any database or Discord
// dependency, so they can be unit-tested directly. The check* methods below
// fetch data and apply escalation; these decide whether a message matches.

// bannedWordMatches reports whether content contains the given banned word.
// Matching is case-insensitive and only triggers when the word is bounded by the
// start/end of the message, whitespace or punctuation. Literal words are escaped;
// patterns flagged as regex are used as-is, so an invalid pattern returns an
// error for the caller to log.
func bannedWordMatches(content string, w database.BannedWord) (bool, error) {
	word := w.WordPattern
	if !w.IsRegex {
		word = regexp.QuoteMeta(w.WordPattern)
	}
	pattern := fmt.Sprintf(`(?i)(?:^|[\s\p{Z}\p{Cf}\p{P}])(%s)(?:$|[\s\p{Z}\p{Cf}\p{P}])`, word)
	return regexp.MatchString(pattern, strings.ToLower(content))
}

// bannedWebsiteMatches reports whether content mentions the banned website. The
// comparison is a case-insensitive substring match, matching the original
// behavior.
func bannedWebsiteMatches(content string, w database.BannedWebsite) bool {
	return strings.Contains(strings.ToLower(content), strings.ToLower(w.WebsiteURL))
}

// containsURL reports whether content contains anything that looks like a URL or
// bare domain. It is a cheap gate so the banned-website list is only scanned for
// messages that actually carry a link.
func containsURL(content string) bool {
	return websiteURLPattern.MatchString(content)
}

// escalateContentViolation applies the shared warn → delete → ban escalation
// used whenever a message breaks a content rule (a banned word or website).
//
//	It warns the author by DM with a copy of their (soon to be deleted) message,
//	then either deletes the message (below the violation threshold) or bans the
//	author (at or above it). It returns true once handled so the caller stops
//	running further checks.
//
//	lang is the guild's language; cause completes the "your message was deleted
//	because {cause}" sentence (already localized by the caller), ruleReason is the
//	reason logged/sent for the action, and logSubject describes the match in the
//	internal info log.
func (d *Discord) escalateContentViolation(s discordSender, m *discordgo.Message, lang i18n.Lang, checkName, cause, ruleReason, logSubject string) bool {
	// Skip if the user is already being banned by a concurrent handler.
	if d.cache.IsUserPendingBan(m.GuildID, m.Author.ID) {
		return true
	}

	violationCount := d.cache.IncrementViolation(m.GuildID, m.Author.ID)
	guildName := guildNameByID(s, m.GuildID)

	// Warn the user and send back a copy of their message.
	warning := i18n.T(lang, "automod.warning", violationCount, d.cache.violationThreshold, guildName, cause)
	d.sendPrivateMessage(s, m, warning)
	for _, part := range utils.SplitMessage(m.Content, 1900) {
		d.sendPrivateMessage(s, m, fmt.Sprintf("```\n%s\n```", part))
	}

	if violationCount < d.cache.violationThreshold {
		// Below threshold: log and delete the message.
		d.logAutomoderationAction(s, m, model.ActionDeleteMessage, checkName, ruleReason)
		d.takeAutomoderationAction(s, m, model.ActionDeleteMessage, i18n.T(lang, "automod.violation_suffix", ruleReason, violationCount, d.cache.violationThreshold))
	} else {
		// Threshold reached: ban the user, guarding against concurrent bans.
		if !d.cache.MarkUserAsPendingBan(m.GuildID, m.Author.ID) {
			return true
		}
		d.logAutomoderationAction(s, m, model.ActionBan, checkName, ruleReason)
		d.takeAutomoderationAction(s, m, model.ActionBan, i18n.T(lang, "automod.ban_reason", ruleReason, violationCount))
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
func (d *Discord) checkBannedWords(s discordSender, m *discordgo.Message) bool {
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

	for _, bannedWord := range bannedWords {
		matched, err := bannedWordMatches(m.Content, bannedWord)
		if err != nil {
			d.logError(m.GuildID, "checkBannedWords()", "Regex error for word '%s': %s", bannedWord.WordPattern, err)
			continue
		}

		if matched {
			lang := d.lang(m.GuildID)
			return d.escalateContentViolation(s, m, lang, "banned_word_check",
				i18n.T(lang, "automod.cause_banned_word", bannedWord.WordPattern),
				i18n.T(lang, "automod.reason_banned_word", bannedWord.WordPattern),
				fmt.Sprintf("banned word: [%s]", bannedWord.WordPattern))
		}
	}

	return false
}

// checkBannedWebsites checks if the message contains any banned website URLs
//
//	The check extracts URLs from the message and compares them against the banned websites list.
//	It matches if the URL contains the banned website pattern.
func (d *Discord) checkBannedWebsites(s discordSender, m *discordgo.Message) bool {
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
	if !containsURL(m.Content) {
		return false
	}

	for _, bannedWebsite := range bannedWebsites {
		if bannedWebsiteMatches(m.Content, bannedWebsite) {
			lang := d.lang(m.GuildID)
			return d.escalateContentViolation(s, m, lang, "banned_website_check",
				i18n.T(lang, "automod.cause_banned_website", bannedWebsite.WebsiteURL),
				i18n.T(lang, "automod.reason_banned_website", bannedWebsite.WebsiteURL),
				fmt.Sprintf("banned website: [%s]", bannedWebsite.WebsiteURL))
		}
	}

	return false
}

// checkMessageSpam detects if a user is sending the exact same message repeatedly across multiple channels
func (d *Discord) checkMessageSpam(s discordSender, m *discordgo.Message) bool {
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
		lang := d.lang(m.GuildID)
		spamReason := i18n.T(lang, "automod.spam_reason", duplicateCount, len(channelsUsed), d.cache.GetViolationWindow())
		d.logAutomoderationAction(s, m, model.ActionBan, "spam_detection", spamReason)
		d.takeAutomoderationAction(s, m, model.ActionBan, spamReason)

		d.logSuccess(m.GuildID, "checkMessageSpam()", "Banned user %s for cross-channel spam (%d duplicate messages across %d channels)", m.Author.Username, duplicateCount, len(channelsUsed))

		return true
	}

	return false
}
