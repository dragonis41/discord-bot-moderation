package api

import (
	"fmt"
	"time"

	"github.com/bwmarrin/discordgo"
	"github.com/dragonis41/discord-bot-moderation/pkg/logger"
	"github.com/dragonis41/discord-bot-moderation/pkg/model"
	"github.com/dragonis41/discord-bot-moderation/pkg/utils"
)

type DiscordAdminFunctionInterface interface {
	showStatus(s *discordgo.Session, i *discordgo.InteractionCreate)
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
		Message: fmt.Sprintf("Got command [%s] from user [%s] asking for bot status", i.ApplicationCommandData().Name, i.Member.User.Username),
	})

	if !d.db.CheckAdminPermissionOnInteraction(s, i) {
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

	nbEntries, err := d.db.GetLogEntriesCount(i.GuildID)
	if err == nil {
		maxEntries, err := d.db.GetMaxLogEntries(i.GuildID)
		if err == nil {
			fields = append(fields, &discordgo.MessageEmbedField{
				Name:   "Log entries",
				Value:  fmt.Sprintf("%d/%d", nbEntries, maxEntries),
				Inline: false,
			})
		}
	}

	last5Errors, err := d.db.GetLogEntriesErrorsByGuildAndSystem(i.GuildID, 5)
	if err == nil && len(last5Errors) > 0 {
		errorMessages := ""
		for _, entry := range last5Errors {
			errorMessages += fmt.Sprintf("- [**%s**] `%s` : \n`%s`\n", entry.CreatedAt, entry.Function, entry.Content)
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
				Timestamp: time.Now().Format(time.RFC3339),
			},
		},
	})
	if err != nil {
		d.log.LogError(logger.LogModel{Database: d.db, GuildID: i.GuildID, Function: "showStatus()",
			Message: fmt.Sprintf("Error sending follow-up message: %s", err),
		})
	}
}
