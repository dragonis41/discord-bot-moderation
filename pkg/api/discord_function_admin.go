package api

import (
	"fmt"
	"time"

	"github.com/bwmarrin/discordgo"
	"github.com/dragonis41/discord-bot-moderation/pkg/utils"
)

type DiscordAdminFunctionInterface interface {
	showStatus(s *discordgo.Session, i *discordgo.InteractionCreate)
	selectModeratorChannels(s *discordgo.Session, i *discordgo.InteractionCreate)
	selectModeratorRoles(s *discordgo.Session, i *discordgo.InteractionCreate)
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

	if !d.db.CheckAdminPermission(s, i) {
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

	maxEntries, err := d.db.GetMaxLogEntries(i.GuildID)
	if err == nil {
		fields = append(fields, &discordgo.MessageEmbedField{
			Name:   "Max log entries",
			Value:  fmt.Sprintf("%d", maxEntries),
			Inline: false,
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
	}
}

func (d *Discord) selectLogChannels(s *discordgo.Session, i *discordgo.InteractionCreate) {
	// Defer the response
	if err := s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseDeferredChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{Flags: discordgo.MessageFlagsEphemeral},
	}); err != nil {
		utils.LogError(fmt.Sprintf("selectLogChannels: Error deferring response: %s", err))
		return
	}

	utils.LogInfo(fmt.Sprintf("Got command [%s] from user [%s]", i.ApplicationCommandData().Name, i.Member.User.Username))

	if !d.db.CheckAdminPermission(s, i) {
		return
	}

	// Get text channels
	textChannels, err := d.getTextChannels(s, i.GuildID)
	if err != nil {
		d.sendErrorMessage(s, i.Interaction, "Sélection des salons", "Une erreur est survenue lors de la récupération des salons.")
		return
	}

	// Start with page 0
	d.sendLogChannelSelectPage(s, i.Interaction, textChannels, 0)
}

func (d *Discord) selectModeratorChannels(s *discordgo.Session, i *discordgo.InteractionCreate) {
	// Defer the response
	if err := s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseDeferredChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{Flags: discordgo.MessageFlagsEphemeral},
	}); err != nil {
		utils.LogError(fmt.Sprintf("selectModeratorChannels: Error deferring response: %s", err))
		return
	}

	utils.LogInfo(fmt.Sprintf("Got command [%s] from user [%s]", i.ApplicationCommandData().Name, i.Member.User.Username))

	if !d.db.CheckAdminPermission(s, i) {
		return
	}

	// Get text channels
	textChannels, err := d.getTextChannels(s, i.GuildID)
	if err != nil {
		d.sendErrorMessage(s, i.Interaction, "Sélection des salons", "Une erreur est survenue lors de la récupération des salons.")
		return
	}

	// Start with page 0
	d.sendModChannelSelectPage(s, i.Interaction, textChannels, 0)
}

func (d *Discord) selectModeratorRoles(s *discordgo.Session, i *discordgo.InteractionCreate) {
	// Defer the response
	if err := s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseDeferredChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{Flags: discordgo.MessageFlagsEphemeral},
	}); err != nil {
		utils.LogError(fmt.Sprintf("selectModeratorRoles: Error deferring response: %s", err))
		return
	}

	utils.LogInfo(fmt.Sprintf("Got command [%s] from user [%s]", i.ApplicationCommandData().Name, i.Member.User.Username))

	if !d.db.CheckAdminPermission(s, i) {
		return
	}

	// Get roles
	roles, err := d.getRoles(s, i.GuildID)
	if err != nil {
		d.sendErrorMessage(s, i.Interaction, "Sélection des roles", "Une erreur est survenue lors de la récupération des roles.")
		return
	}

	// Start with page 0
	d.sendRoleSelectPage(s, i.Interaction, roles, 0)
}
