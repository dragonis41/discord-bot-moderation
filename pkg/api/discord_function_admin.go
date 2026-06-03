package api

import (
	"bytes"
	"fmt"
	"time"

	"github.com/bwmarrin/discordgo"
	"github.com/dragonis41/discord-bot-moderation/pkg/model"
	"github.com/dragonis41/discord-bot-moderation/pkg/utils"
)

type DiscordAdminFunctionInterface interface {
	showStatus(s *discordgo.Session, i *discordgo.InteractionCreate)
	getMessageHistory(s *discordgo.Session, i *discordgo.InteractionCreate)
	getBotLogs(s *discordgo.Session, i *discordgo.InteractionCreate)
	getModerationLogs(s *discordgo.Session, i *discordgo.InteractionCreate)
}

func (d *Discord) showStatus(s *discordgo.Session, i *discordgo.InteractionCreate) {
	if !d.beginModCommand(s, i, "showStatus()") {
		return
	}

	var fields []*discordgo.MessageEmbedField

	// Uptime
	fields = append(fields, &discordgo.MessageEmbedField{
		Name:   "Uptime",
		Value:  utils.GetUptime(),
		Inline: false,
	})

	// Connected servers
	connectedServers := ""
	for _, guild := range s.State.Guilds {
		connectedServers += fmt.Sprintf("- [%s] (ID: %s)\n", d.guildDisplayName(guild), guild.ID)
	}
	fields = append(fields, &discordgo.MessageEmbedField{
		Name:   "Serveurs connectés",
		Value:  fmt.Sprintf("%d serveurs :\n%s", len(s.State.Guilds), connectedServers),
		Inline: false,
	})

	// CPU and RAM Usage
	fields = append(fields, &discordgo.MessageEmbedField{
		Name:   "Utilisation CPU",
		Value:  utils.GetCPUUsage(),
		Inline: true,
	})

	fields = append(fields, &discordgo.MessageEmbedField{
		Name:   "Utilisation mémoire",
		Value:  utils.GetMemoryUsage(),
		Inline: false,
	})

	// Get detailed stats and add them as inline fields
	detailedStats := utils.GetDetailedStats()

	// Add specific detailed stats as separate fields
	if goroutines, ok := detailedStats["Goroutines"]; ok {
		fields = append(fields, &discordgo.MessageEmbedField{
			Name:   "Goroutines",
			Value:  goroutines,
			Inline: true,
		})
	}

	if rss, ok := detailedStats["RSS Memory"]; ok {
		fields = append(fields, &discordgo.MessageEmbedField{
			Name:   "Mémoire RSS",
			Value:  rss,
			Inline: true,
		})
	}

	nbEntries, err := d.db.GetSystemLogEntriesCount(i.GuildID)
	if err == nil {
		maxEntries, err := d.db.GetMaxSystemLogEntries(i.GuildID)
		if err == nil {
			fields = append(fields, &discordgo.MessageEmbedField{
				Name:   "Log entries",
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
				Name:   "5 dernières erreurs",
				Value:  errorMessages,
				Inline: false,
			})
		}
	}

	d.followup(s, i, "showStatus()", &discordgo.MessageEmbed{
		Title:     "Status",
		Color:     model.Green.Int(),
		Fields:    fields,
		Footer:    model.DefaultFooter,
		Timestamp: time.Now().UTC().Format(time.RFC3339),
	})
}

func (d *Discord) getMessageHistory(s *discordgo.Session, i *discordgo.InteractionCreate) {
	if !d.beginModCommand(s, i, "getMessageHistory()") {
		return
	}

	limit := 10
	for _, option := range i.ApplicationCommandData().Options {
		if option.Name == "limit" {
			limit = int(option.IntValue())
		}
	}
	if limit <= 0 || limit > 100 {
		d.followup(s, i, "getMessageHistory()", errorEmbed("Message History", "La limite doit être comprise entre 1 et 100."))
		return
	}

	// Get the message history from the cache
	historyMessages := d.cache.GetGuildRecentMessages(i.GuildID, limit)
	if len(historyMessages) == 0 {
		d.followup(s, i, "getMessageHistory()", errorEmbed("Message History", "Aucun message en cache pour ce serveur."))
		return
	}

	// Format the messages and send them as an attachment
	messageContent := fmt.Sprintf("Derniers %d messages en cache :\n\n", limit)
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

		attachments := "<No Attachments>"
		if msg.AttachmentCount > 0 {
			attachments = fmt.Sprintf("<%d Attachments>", msg.AttachmentCount)
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

		messageContent += fmt.Sprintf("----------------------------------------\n[%s UTC] @%s (ID: %s) in #%s\nMessage : %s\nAttachments : %s%s\n",
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

func (d *Discord) getBotLogs(s *discordgo.Session, i *discordgo.InteractionCreate) {
	if !d.beginModCommand(s, i, "getBotLogs()") {
		return
	}

	nbEntries, err := d.db.GetSystemLogEntriesCount(i.GuildID)
	if err != nil {
		d.logError(i.GuildID, "getBotLogs()", "Error getting system log entries count: %s", err)
		d.followup(s, i, "getBotLogs()", errorEmbed("Bot logs", "Erreur lors de la récupération des logs du bot."))
		return
	}

	maxEntries, err := d.db.GetMaxSystemLogEntries(i.GuildID)
	if err != nil {
		d.logError(i.GuildID, "getBotLogs()", "Error getting max system log entries: %s", err)
		d.followup(s, i, "getBotLogs()", errorEmbed("Bot logs", "Erreur lors de la récupération des logs du bot."))
		return
	}

	entries, _ := d.db.GetSystemLogEntriesByGuild(i.GuildID, 10)
	var lines []string
	for _, entry := range entries {
		lines = append(lines, fmt.Sprintf("----------------------------------------\n- [%s] `%s` : \n`%s`\n",
			formatDiscordTimestamp(entry.CreatedAt), entry.Function, entry.Content))
	}

	d.followupLogEmbed(s, i, "getBotLogs()", "Bot logs", nbEntries, maxEntries, lines, "Aucun log trouvé.")
}

func (d *Discord) getModerationLogs(s *discordgo.Session, i *discordgo.InteractionCreate) {
	if !d.beginModCommand(s, i, "getModerationLogs()") {
		return
	}

	nbEntries, err := d.db.GetModerationLogEntriesCount(i.GuildID)
	if err != nil {
		d.logError(i.GuildID, "getModerationLogs()", "Error getting mod log entries count: %s", err)
		d.followup(s, i, "getModerationLogs()", errorEmbed("Logs de modération", "Erreur lors de la récupération des logs de modération."))
		return
	}

	maxEntries, err := d.db.GetMaxModerationLogEntries(i.GuildID)
	if err != nil {
		d.logError(i.GuildID, "getModerationLogs()", "Error getting max moderation log entries: %s", err)
		d.followup(s, i, "getModerationLogs()", errorEmbed("Logs de modération", "Erreur lors de la récupération des logs de modération."))
		return
	}

	entries, _ := d.db.GetModerationLogEntriesByGuild(i.GuildID, 10)
	var lines []string
	for _, entry := range entries {
		lines = append(lines, fmt.Sprintf("----------------------------------------\n- [%s] Action : `%s` <@%s> (ID: %s) : \n`%s`\n",
			formatDiscordTimestamp(entry.CreatedAt), entry.Action, entry.UserID, entry.UserID, entry.Reason))
	}

	d.followupLogEmbed(s, i, "getModerationLogs()", "Logs de modération", nbEntries, maxEntries, lines, "Aucun log de modération trouvé.")
}

// followupLogEmbed sends the standard log-listing embed: a green embed with a
// single "10 derniers logs" field showing count/max followed by the log lines,
// trimmed to stay below Discord's per-field length limit. emptyMessage is shown
// when there are no log lines.
func (d *Discord) followupLogEmbed(s *discordgo.Session, i *discordgo.InteractionCreate, function, title string, count, max int, lines []string, emptyMessage string) {
	const maxFieldLength = 950 // Leave some margin below Discord's 1024 field limit

	body := emptyMessage
	if len(lines) > 0 {
		body = joinLinesWithLimit(lines, maxFieldLength)
	}

	d.followup(s, i, function, &discordgo.MessageEmbed{
		Title: title,
		Color: model.Green.Int(),
		Fields: []*discordgo.MessageEmbedField{{
			Name:   "10 derniers logs",
			Value:  fmt.Sprintf("%d/%d\n\n%s", count, max, body),
			Inline: false,
		}},
		Footer:    model.DefaultFooter,
		Timestamp: time.Now().UTC().Format(time.RFC3339),
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
