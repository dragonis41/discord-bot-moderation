package api

import (
	"fmt"
	"time"

	"github.com/bwmarrin/discordgo"
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
		utils.LogError(fmt.Sprintf("showStatus: Error deferring response: %s", err))
		return
	}

	utils.LogInfo(fmt.Sprintf("Got command [%s] from user [%s] asking for bot status", i.ApplicationCommandData().Name, i.Member.User.Username))

	// TODO : Get roles from the database
	adminRoleNames := []string{"sudoers"}
	if !utils.UserHasRoleByName(s, i, adminRoleNames) {
		_, _ = s.FollowupMessageCreate(i.Interaction, true, &discordgo.WebhookParams{
			Embeds: []*discordgo.MessageEmbed{
				{
					Description: "❌ Vous n'avez pas la permission d'utiliser cette commande.",
					Color:       red,
				},
			},
		})
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
				fmt.Printf("Server: [<error fetching>] (ID: %s)\n", guild.ID)
				continue
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

	// TODO : Also show the last errors in the logs

	_, err = s.FollowupMessageCreate(i.Interaction, true, &discordgo.WebhookParams{
		Embeds: []*discordgo.MessageEmbed{
			{
				Title:     "Status",
				Color:     green,
				Fields:    fields,
				Footer:    defaultFooter,
				Timestamp: time.Now().Format(time.RFC3339),
			},
		},
	})
	if err != nil {
		utils.LogError(fmt.Sprintf("showStatus: Error sending follow-up message: %s", err))
		return
	}
}
