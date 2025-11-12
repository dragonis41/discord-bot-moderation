package api

import (
	"bytes"
	"fmt"
	"time"

	"github.com/bwmarrin/discordgo"
	"github.com/dragonis41/discord-bot-moderation/pkg/logger"
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
	// Defer the response
	err := s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseDeferredChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Flags: discordgo.MessageFlagsEphemeral,
		},
	})
	if err != nil {
		d.log.LogError(logger.LogModel{Database: d.db, GuildID: i.GuildID, Function: "showStatus()",
			Message: fmt.Sprintf("Error deferring response: %s", err),
		})
		return
	}

	d.log.LogInfo(logger.LogModel{Database: d.db, GuildID: i.GuildID, Function: "showStatus()",
		Message: fmt.Sprintf("Got command [%s] from user [%s]", i.ApplicationCommandData().Name, i.Member.User.Username),
	})

	if !d.db.CheckModerationPermissionOnInteraction(s, i) {
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
		// Fetch full guild data if name is empty
		if guild.Name == "" {
			fullGuild, err := d.client.Guild(guild.ID)
			if err != nil {
				guild.Name = "<error fetching>"
			}
			guild = fullGuild
		}
		connectedServers += fmt.Sprintf("- [%s] (ID: %s)\n", guild.Name, guild.ID)
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
		errorMessages := ""
		const maxLength = 950 // Leave some margin below 1024

		for _, entry := range last5Errors {
			logEntry := fmt.Sprintf("- [**%s**] `%s` : \n`%s`\n", entry.CreatedAt, entry.Function, entry.Content)

			// Check if adding this entry would exceed the limit
			if len(errorMessages)+len(logEntry) > maxLength {
				errorMessages += "\n... (logs truncated)"
				break
			}
			errorMessages += logEntry
		}

		if errorMessages != "" {
			fields = append(fields, &discordgo.MessageEmbedField{
				Name:   "5 dernières erreurs",
				Value:  errorMessages,
				Inline: false,
			})
		}
	}

	_, err = s.FollowupMessageCreate(i.Interaction, true, &discordgo.WebhookParams{
		Embeds: []*discordgo.MessageEmbed{
			{
				Title:     "Status",
				Color:     model.Green.Int(),
				Fields:    fields,
				Footer:    model.DefaultFooter,
				Timestamp: time.Now().Format(time.DateTime),
			},
		},
	})
	if err != nil {
		d.log.LogError(logger.LogModel{Database: d.db, GuildID: i.GuildID, Function: "showStatus()",
			Message: fmt.Sprintf("Error sending follow-up message: %s", err),
		})
	}
}

func (d *Discord) getMessageHistory(s *discordgo.Session, i *discordgo.InteractionCreate) {
	// Defer the response
	err := s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseDeferredChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Flags: discordgo.MessageFlagsEphemeral,
		},
	})
	if err != nil {
		d.log.LogError(logger.LogModel{Database: d.db, GuildID: i.GuildID, Function: "getMessageHistory()",
			Message: fmt.Sprintf("Error deferring response: %s", err),
		})
		return
	}

	d.log.LogInfo(logger.LogModel{Database: d.db, GuildID: i.GuildID, Function: "getMessageHistory()",
		Message: fmt.Sprintf("Got command [%s] from user [%s]", i.ApplicationCommandData().Name, i.Member.User.Username),
	})

	if !d.db.CheckModerationPermissionOnInteraction(s, i) {
		return
	}

	options := i.ApplicationCommandData().Options
	var limit = 10
	// Parse options
	for _, option := range options {
		switch option.Name {
		case "limit":
			limit = int(option.IntValue())
		}
	}
	if limit <= 0 || limit > 100 {
		// Send error message
		_, err = s.FollowupMessageCreate(i.Interaction, true, &discordgo.WebhookParams{
			Embeds: []*discordgo.MessageEmbed{
				{Title: "Message History", Color: model.Red.Int(), Description: "La limite doit être comprise entre 1 et 100.", Footer: model.DefaultFooter, Timestamp: time.Now().Format(time.DateTime)},
			},
		})
		if err != nil {
			d.log.LogError(logger.LogModel{Database: d.db, GuildID: i.GuildID, Function: "getMessageHistory()",
				Message: fmt.Sprintf("Error sending follow-up error message: %s", err),
			})
		}
		return
	}

	// Get the message history from the cache
	historyMessages := d.cache.GetGuildRecentMessages(i.GuildID, limit)
	if len(historyMessages) == 0 {
		_, err = s.FollowupMessageCreate(i.Interaction, true, &discordgo.WebhookParams{
			Embeds: []*discordgo.MessageEmbed{
				{Title: "Message History", Color: model.Red.Int(), Description: "Aucun message en cache pour ce serveur.", Footer: model.DefaultFooter, Timestamp: time.Now().Format(time.DateTime)},
			},
		})
		if err != nil {
			d.log.LogError(logger.LogModel{Database: d.db, GuildID: i.GuildID, Function: "getMessageHistory()",
				Message: fmt.Sprintf("Error sending follow-up error message: %s", err),
			})
		}
		return
	}

	// Format the messages and send them as an attachment
	messageContent := fmt.Sprintf("Derniers %d messages en cache :\n\n", limit)
	for _, msg := range historyMessages {
		var user *discordgo.User
		member, err := s.State.Member(i.GuildID, msg.AuthorID)
		if err != nil {
			// Fallback to API call
			member, err = s.GuildMember(i.GuildID, msg.AuthorID)
			if err != nil {
				user = &discordgo.User{Username: "<Unknown User>"}
			} else {
				user = member.User
			}
		} else {
			user = member.User
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
		}
		messageContent += fmt.Sprintf("----------------------------------------\n[%s] in #%s\n%s (ID: %s) : %s\n%s\n",
			msg.Timestamp.Format(time.DateTime),
			channel.Name,
			user.Username,
			msg.AuthorID,
			msg.Content,
			attachments,
		)
	}

	_, err = s.FollowupMessageCreate(i.Interaction, true, &discordgo.WebhookParams{
		Files: []*discordgo.File{
			{
				Name:        "message_history.txt",
				ContentType: "text/plain",
				Reader:      bytes.NewReader([]byte(messageContent)),
			},
		},
	})
	if err != nil {
		d.log.LogError(logger.LogModel{Database: d.db, GuildID: i.GuildID, Function: "getMessageHistory()",
			Message: fmt.Sprintf("Error sending follow-up message: %s", err),
		})
	}
}

func (d *Discord) getBotLogs(s *discordgo.Session, i *discordgo.InteractionCreate) {
	// Defer the response
	err := s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseDeferredChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Flags: discordgo.MessageFlagsEphemeral,
		},
	})
	if err != nil {
		d.log.LogError(logger.LogModel{Database: d.db, GuildID: i.GuildID, Function: "getBotLogs()",
			Message: fmt.Sprintf("Error deferring response: %s", err),
		})
		return
	}

	d.log.LogInfo(logger.LogModel{Database: d.db, GuildID: i.GuildID, Function: "getBotLogs()",
		Message: fmt.Sprintf("Got command [%s] from user [%s]", i.ApplicationCommandData().Name, i.Member.User.Username),
	})

	if !d.db.CheckModerationPermissionOnInteraction(s, i) {
		return
	}

	nbEntries, err := d.db.GetSystemLogEntriesCount(i.GuildID)
	if err != nil {
		d.log.LogError(logger.LogModel{Database: d.db, GuildID: i.GuildID, Function: "getBotLogs()",
			Message: fmt.Sprintf("Error getting system log entries count: %s", err),
		})
		_, err = s.FollowupMessageCreate(i.Interaction, true, &discordgo.WebhookParams{
			Embeds: []*discordgo.MessageEmbed{
				{Title: "Bot logs", Color: model.Red.Int(), Description: "Erreur lors de la récupération des logs du bot.", Footer: model.DefaultFooter, Timestamp: time.Now().Format(time.DateTime)},
			},
		})
		if err != nil {
			d.log.LogError(logger.LogModel{Database: d.db, GuildID: i.GuildID, Function: "getBotLogs()",
				Message: fmt.Sprintf("Error sending follow-up error message: %s", err),
			})
		}
		return
	}

	maxEntries, err := d.db.GetMaxSystemLogEntries(i.GuildID)
	if err != nil {
		d.log.LogError(logger.LogModel{Database: d.db, GuildID: i.GuildID, Function: "getBotLogs()",
			Message: fmt.Sprintf("Error getting max system log entries: %s", err),
		})
		_, err = s.FollowupMessageCreate(i.Interaction, true, &discordgo.WebhookParams{
			Embeds: []*discordgo.MessageEmbed{
				{Title: "Bot logs", Color: model.Red.Int(), Description: "Erreur lors de la récupération des logs du bot.", Footer: model.DefaultFooter, Timestamp: time.Now().Format(time.DateTime)},
			},
		})
		if err != nil {
			d.log.LogError(logger.LogModel{Database: d.db, GuildID: i.GuildID, Function: "getBotLogs()",
				Message: fmt.Sprintf("Error sending follow-up error message: %s", err),
			})
		}
		return
	}

	var fields []*discordgo.MessageEmbedField
	last10Errors, err := d.db.GetSystemLogEntriesByGuild(i.GuildID, 10)
	errorMessages := "Aucun log trouvé."
	if err == nil && len(last10Errors) > 0 {
		errorMessages = ""
		const maxLength = 950 // Leave some margin below 1024

		for _, entry := range last10Errors {
			logEntry := fmt.Sprintf("----------------------------------------\n- [**%s**] `%s` : \n`%s`\n",
				entry.CreatedAt, entry.Function, entry.Content)

			// Check if adding this entry would exceed the limit
			if len(errorMessages)+len(logEntry) > maxLength {
				errorMessages += "\n... (logs truncated)"
				break
			}
			errorMessages += logEntry
		}
	}

	fields = append(fields, &discordgo.MessageEmbedField{
		Name:   "10 derniers logs",
		Value:  fmt.Sprintf("%d/%d\n\n%s", nbEntries, maxEntries, errorMessages),
		Inline: false,
	})

	_, err = s.FollowupMessageCreate(i.Interaction, true, &discordgo.WebhookParams{
		Embeds: []*discordgo.MessageEmbed{
			{Title: "Bot logs", Color: model.Green.Int(), Fields: fields, Footer: model.DefaultFooter, Timestamp: time.Now().Format(time.DateTime)},
		},
	})
	if err != nil {
		d.log.LogError(logger.LogModel{Database: d.db, GuildID: i.GuildID, Function: "getBotLogs()",
			Message: fmt.Sprintf("Error sending follow-up message: %s", err),
		})
	}
}

func (d *Discord) getModerationLogs(s *discordgo.Session, i *discordgo.InteractionCreate) {
	// Defer the response
	err := s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseDeferredChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Flags: discordgo.MessageFlagsEphemeral,
		},
	})
	if err != nil {
		d.log.LogError(logger.LogModel{Database: d.db, GuildID: i.GuildID, Function: "getModerationLogs()",
			Message: fmt.Sprintf("Error deferring response: %s", err),
		})
		return
	}

	d.log.LogInfo(logger.LogModel{Database: d.db, GuildID: i.GuildID, Function: "getModerationLogs()",
		Message: fmt.Sprintf("Got command [%s] from user [%s]", i.ApplicationCommandData().Name, i.Member.User.Username),
	})

	if !d.db.CheckModerationPermissionOnInteraction(s, i) {
		return
	}

	nbEntries, err := d.db.GetModerationLogEntriesCount(i.GuildID)
	if err != nil {
		d.log.LogError(logger.LogModel{Database: d.db, GuildID: i.GuildID, Function: "getModerationLogs()",
			Message: fmt.Sprintf("Error getting mod log entries count: %s", err),
		})
		_, err = s.FollowupMessageCreate(i.Interaction, true, &discordgo.WebhookParams{
			Embeds: []*discordgo.MessageEmbed{
				{Title: "Logs de modération", Color: model.Red.Int(), Description: "Erreur lors de la récupération des logs de modération.", Footer: model.DefaultFooter, Timestamp: time.Now().Format(time.DateTime)},
			},
		})
		if err != nil {
			d.log.LogError(logger.LogModel{Database: d.db, GuildID: i.GuildID, Function: "getModerationLogs()",
				Message: fmt.Sprintf("Error sending follow-up error message: %s", err),
			})
		}
		return
	}

	maxEntries, err := d.db.GetMaxModerationLogEntries(i.GuildID)
	if err != nil {
		d.log.LogError(logger.LogModel{Database: d.db, GuildID: i.GuildID, Function: "getModerationLogs()",
			Message: fmt.Sprintf("Error getting max system log entries: %s", err),
		})
		_, err = s.FollowupMessageCreate(i.Interaction, true, &discordgo.WebhookParams{
			Embeds: []*discordgo.MessageEmbed{
				{Title: "Logs de modération", Color: model.Red.Int(), Description: "Erreur lors de la récupération des logs de modération.", Footer: model.DefaultFooter, Timestamp: time.Now().Format(time.DateTime)},
			},
		})
		if err != nil {
			d.log.LogError(logger.LogModel{Database: d.db, GuildID: i.GuildID, Function: "getModerationLogs()",
				Message: fmt.Sprintf("Error sending follow-up error message: %s", err),
			})
		}
		return
	}

	var fields []*discordgo.MessageEmbedField
	errorMessages := "Aucun log de modération trouvé."
	last10Errors, err := d.db.GetModerationLogEntriesByGuild(i.GuildID, 10)
	if err == nil && len(last10Errors) > 0 {
		errorMessages = ""
		const maxLength = 950 // Leave some margin below 1024

		for _, entry := range last10Errors {
			logEntry := fmt.Sprintf("----------------------------------------\n- [**%s**] Action : `%s` <@%s> (ID: %s) : \n`%s`\n",
				entry.CreatedAt, entry.Action, entry.UserID, entry.UserID, entry.Reason)

			// Check if adding this entry would exceed the limit
			if len(errorMessages)+len(logEntry) > maxLength {
				errorMessages += "\n... (logs truncated)"
				break
			}
			errorMessages += logEntry
		}
	}

	fields = append(fields, &discordgo.MessageEmbedField{
		Name:   "10 derniers logs",
		Value:  fmt.Sprintf("%d/%d\n\n%s", nbEntries, maxEntries, errorMessages),
		Inline: false,
	})

	_, err = s.FollowupMessageCreate(i.Interaction, true, &discordgo.WebhookParams{
		Embeds: []*discordgo.MessageEmbed{
			{Title: "Logs de modération", Color: model.Green.Int(), Fields: fields, Footer: model.DefaultFooter, Timestamp: time.Now().Format(time.DateTime)},
		},
	})
	if err != nil {
		d.log.LogError(logger.LogModel{Database: d.db, GuildID: i.GuildID, Function: "getModerationLogs()",
			Message: fmt.Sprintf("Error sending follow-up message: %s", err),
		})
	}
}
