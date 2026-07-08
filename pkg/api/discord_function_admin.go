package api

import (
	"bytes"
	"fmt"
	"time"

	"github.com/bwmarrin/discordgo"
	"github.com/dragonis41/discord-bot-moderation/pkg/i18n"
	"github.com/dragonis41/discord-bot-moderation/pkg/model"
	"github.com/dragonis41/discord-bot-moderation/pkg/utils"
)

type DiscordAdminFunctionInterface interface {
	showStatus(s *discordgo.Session, i *discordgo.InteractionCreate)
	getMessageHistory(s *discordgo.Session, i *discordgo.InteractionCreate)
	getBotLogs(s discordClient, i *discordgo.InteractionCreate)
	getModerationLogs(s discordClient, i *discordgo.InteractionCreate)
}

// logDisplayLimit is how many recent log entries /get-bot-logs and
// /get-moderation-logs fetch. It is generous on purpose: the renderer fills the
// embed description up to Discord's limit and stops, so this just needs to be
// large enough to fill that space.
const logDisplayLimit = 50

func (d *Discord) showStatus(s *discordgo.Session, i *discordgo.InteractionCreate) {
	if !d.beginModCommand(s, i, "showStatus()") {
		return
	}

	lang := d.lang(i.GuildID)

	var fields []*discordgo.MessageEmbedField

	// Uptime
	fields = append(fields, &discordgo.MessageEmbedField{
		Name:   i18n.T(lang, "status.uptime"),
		Value:  utils.GetUptime(),
		Inline: false,
	})

	// Connected servers
	connectedServers := ""
	for _, guild := range s.State.Guilds {
		connectedServers += fmt.Sprintf("- [%s] (ID: %s)\n", d.guildDisplayName(guild), guild.ID)
	}
	fields = append(fields, &discordgo.MessageEmbedField{
		Name:   i18n.T(lang, "status.servers"),
		Value:  i18n.T(lang, "status.servers_value", len(s.State.Guilds), connectedServers),
		Inline: false,
	})

	// CPU and RAM Usage
	fields = append(fields, &discordgo.MessageEmbedField{
		Name:   i18n.T(lang, "status.cpu"),
		Value:  utils.GetCPUUsage(),
		Inline: true,
	})

	fields = append(fields, &discordgo.MessageEmbedField{
		Name:   i18n.T(lang, "status.memory"),
		Value:  utils.GetMemoryUsage(),
		Inline: false,
	})

	// Get detailed stats and add them as inline fields
	detailedStats := utils.GetDetailedStats()

	// Add specific detailed stats as separate fields
	if goroutines, ok := detailedStats["Goroutines"]; ok {
		fields = append(fields, &discordgo.MessageEmbedField{
			Name:   i18n.T(lang, "status.goroutines"),
			Value:  goroutines,
			Inline: true,
		})
	}

	if rss, ok := detailedStats["RSS Memory"]; ok {
		fields = append(fields, &discordgo.MessageEmbedField{
			Name:   i18n.T(lang, "status.rss"),
			Value:  rss,
			Inline: true,
		})
	}

	nbEntries, err := d.db.GetSystemLogEntriesCount(i.GuildID)
	if err == nil {
		maxEntries, err := d.db.GetMaxSystemLogEntries(i.GuildID)
		if err == nil {
			fields = append(fields, &discordgo.MessageEmbedField{
				Name:   i18n.T(lang, "status.log_entries"),
				Value:  fmt.Sprintf("%d/%d", nbEntries, maxEntries),
				Inline: false,
			})
		}
	}

	last5Errors, err := d.db.GetSystemLogEntriesErrorsByGuildAndSystem(i.GuildID, 5)
	if err == nil && len(last5Errors) > 0 {
		var lines []string
		for _, entry := range last5Errors {
			lines = append(lines, fmt.Sprintf("- [%s] `%s` : \n`%s`\n", formatDiscordTimestamp(entry.CreatedAt), entry.Function, entry.Content))
		}

		if errorMessages := joinLinesWithLimit(lines, 950); errorMessages != "" {
			fields = append(fields, &discordgo.MessageEmbedField{
				Name:   i18n.T(lang, "status.last_errors"),
				Value:  errorMessages,
				Inline: false,
			})
		}
	}

	d.followup(s, i, "showStatus()", &discordgo.MessageEmbed{
		Title:     i18n.T(lang, "status.title"),
		Color:     model.Green.Int(),
		Fields:    fields,
		Footer:    hintFooter(lang),
		Timestamp: time.Now().UTC().Format(time.RFC3339),
	})
}

func (d *Discord) getMessageHistory(s *discordgo.Session, i *discordgo.InteractionCreate) {
	if !d.beginModCommand(s, i, "getMessageHistory()") {
		return
	}

	lang := d.lang(i.GuildID)

	limit := 10
	for _, option := range i.ApplicationCommandData().Options {
		if option.Name == "limit" {
			limit = int(option.IntValue())
		}
	}
	if limit <= 0 || limit > 1000 {
		d.followup(s, i, "getMessageHistory()", errorEmbed(lang, i18n.T(lang, "history.title"), i18n.T(lang, "history.limit_error")))
		return
	}

	// Get the message history from the cache
	historyMessages := d.cache.GetGuildRecentMessages(i.GuildID, limit)
	if len(historyMessages) == 0 {
		d.followup(s, i, "getMessageHistory()", errorEmbed(lang, i18n.T(lang, "history.title"), i18n.T(lang, "history.empty")))
		return
	}

	// Format the messages and send them as an attachment
	messageContent := i18n.T(lang, "history.header", limit)
	for _, msg := range historyMessages {
		// Use cached username
		username := msg.AuthorUsername
		if username == "" {
			username = "<Unknown User>"
		}

		channel, err := s.State.Channel(msg.ChannelID)
		if err != nil {
			channel, err = s.Channel(msg.ChannelID)
			if err != nil {
				channel = &discordgo.Channel{Name: "<Unknown Channel>"}
			}
		}

		attachments := ""
		if msg.AttachmentCount > 0 {
			attachments = fmt.Sprintf("\nAttachments : <%d Attachments>", msg.AttachmentCount)
			for _, name := range msg.AttachmentNames {
				attachments += "\n- " + name
			}
		}
		stickers := ""
		if len(msg.Stickers) > 0 {
			stickers = fmt.Sprintf("\nStickers : <%d Sticker(s)>", len(msg.Stickers))
			for _, s := range msg.Stickers {
				stickers += fmt.Sprintf("\n- %s (%s)", s.Name, s.ID)
			}
		}

		messageContent += fmt.Sprintf("----------------------------------------\n[%s UTC] @%s (ID: %s) in #%s\nMessage : %s%s%s\n",
			msg.Timestamp.UTC().Format(time.DateTime),
			username,
			msg.AuthorID,
			channel.Name,
			msg.Content,
			attachments,
			stickers,
		)
	}

	_, err := s.FollowupMessageCreate(i.Interaction, true, &discordgo.WebhookParams{
		Files: []*discordgo.File{
			{
				Name:        fmt.Sprintf("message_history_%s_%s.txt", i.GuildID, time.Now().Format("20060102_150405")),
				ContentType: "text/plain",
				Reader:      bytes.NewReader([]byte(messageContent)),
			},
		},
	})
	if err != nil {
		d.logError(i.GuildID, "getMessageHistory()", "Error sending follow-up message: %s", err)
	}
}

func (d *Discord) getBotLogs(s discordClient, i *discordgo.InteractionCreate) {
	if !d.beginModCommand(s, i, "getBotLogs()") {
		return
	}

	lang := d.lang(i.GuildID)

	nbEntries, err := d.db.GetSystemLogEntriesCount(i.GuildID)
	if err != nil {
		d.logError(i.GuildID, "getBotLogs()", "Error getting system log entries count: %s", err)
		d.followup(s, i, "getBotLogs()", errorEmbed(lang, i18n.T(lang, "logs.bot_title"), i18n.T(lang, "logs.bot_error")))
		return
	}

	maxEntries, err := d.db.GetMaxSystemLogEntries(i.GuildID)
	if err != nil {
		d.logError(i.GuildID, "getBotLogs()", "Error getting max system log entries: %s", err)
		d.followup(s, i, "getBotLogs()", errorEmbed(lang, i18n.T(lang, "logs.bot_title"), i18n.T(lang, "logs.bot_error")))
		return
	}

	entries, _ := d.db.GetSystemLogEntriesByGuild(i.GuildID, logDisplayLimit)
	var lines []string
	for _, entry := range entries {
		lines = append(lines, fmt.Sprintf("----------------------------------------\n- [%s] `%s` : \n`%s`\n",
			formatDiscordTimestamp(entry.CreatedAt), entry.Function, entry.Content))
	}

	d.followupLogEmbed(s, i, lang, "getBotLogs()", i18n.T(lang, "logs.bot_title"), nbEntries, maxEntries, lines, i18n.T(lang, "logs.bot_empty"))
}

func (d *Discord) getModerationLogs(s discordClient, i *discordgo.InteractionCreate) {
	if !d.beginModCommand(s, i, "getModerationLogs()") {
		return
	}

	lang := d.lang(i.GuildID)

	nbEntries, err := d.db.GetModerationLogEntriesCount(i.GuildID)
	if err != nil {
		d.logError(i.GuildID, "getModerationLogs()", "Error getting mod log entries count: %s", err)
		d.followup(s, i, "getModerationLogs()", errorEmbed(lang, i18n.T(lang, "logs.mod_title"), i18n.T(lang, "logs.mod_error")))
		return
	}

	maxEntries, err := d.db.GetMaxModerationLogEntries(i.GuildID)
	if err != nil {
		d.logError(i.GuildID, "getModerationLogs()", "Error getting max moderation log entries: %s", err)
		d.followup(s, i, "getModerationLogs()", errorEmbed(lang, i18n.T(lang, "logs.mod_title"), i18n.T(lang, "logs.mod_error")))
		return
	}

	entries, _ := d.db.GetModerationLogEntriesByGuild(i.GuildID, logDisplayLimit)
	var lines []string
	for _, entry := range entries {

		lines = append(lines, fmt.Sprintf("----------------------------------------\n- [%s] Action : `%s` <@%s> (%s | %s) : \n`%s`\n",
			formatDiscordTimestamp(entry.CreatedAt), entry.Action, entry.UserID, entry.Username, entry.UserID, entry.Reason))
	}

	d.followupLogEmbed(s, i, lang, "getModerationLogs()", i18n.T(lang, "logs.mod_title"), nbEntries, maxEntries, lines, i18n.T(lang, "logs.mod_empty"))
}

// followupLogEmbed sends the standard log-listing embed: a green embed whose
// description shows count/max followed by as many log lines as fit. The list is
// rendered into the embed description (4096 chars) rather than a field (1024),
// so it holds roughly four times more before truncating. emptyMessage is shown
// when there are no log lines.
func (d *Discord) followupLogEmbed(s discordClient, i *discordgo.InteractionCreate, lang i18n.Lang, function, title string, count, max int, lines []string, emptyMessage string) {
	// Reserve a little room for the "count/max" header so the joined log lines
	// can use the rest of the 4096-char description budget.
	const headerMargin = 96

	body := emptyMessage
	if len(lines) > 0 {
		body = joinLinesWithLimit(lines, maxEmbedDescriptionLength-headerMargin)
	}

	description := truncate(i18n.T(lang, "logs.latest", count, max, body), maxEmbedDescriptionLength)

	d.followup(s, i, function, &discordgo.MessageEmbed{
		Title:       title,
		Color:       model.Green.Int(),
		Description: description,
		Footer:      hintFooter(lang),
		Timestamp:   time.Now().UTC().Format(time.RFC3339),
	})
}

// formatDiscordTimestamp converts a string timestamp to Discord's timestamp format
// It tries multiple common formats and returns a Discord timestamp that displays in user's local time
func formatDiscordTimestamp(timestamp string) string {
	formats := []string{
		time.RFC3339,
		"2006-01-02T15:04:05Z",
		"2006-01-02 15:04:05",
		time.DateTime,
	}

	for _, format := range formats {
		if t, err := time.Parse(format, timestamp); err == nil {
			return fmt.Sprintf("<t:%d:f>", t.Unix())
		}
	}

	// If parsing fails, return the original timestamp
	return timestamp
}
