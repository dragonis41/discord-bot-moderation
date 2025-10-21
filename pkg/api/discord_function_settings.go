package api

import (
	"fmt"

	"github.com/bwmarrin/discordgo"
	"github.com/dragonis41/discord-bot-moderation/pkg/logger"
)

type DiscordSettingsFunctionInterface interface {
	selectLogChannels(s *discordgo.Session, i *discordgo.InteractionCreate)
	selectModeratorChannels(s *discordgo.Session, i *discordgo.InteractionCreate)
	selectModeratorRoles(s *discordgo.Session, i *discordgo.InteractionCreate)
}

func (d *Discord) selectLogChannels(s *discordgo.Session, i *discordgo.InteractionCreate) {
	// Defer the response
	if err := s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseDeferredChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{Flags: discordgo.MessageFlagsEphemeral},
	}); err != nil {
		d.log.LogError(logger.LogModel{Database: d.db, GuildID: i.GuildID, Function: "selectLogChannels()",
			Message: fmt.Sprintf("Error deferring response: %s", err),
		})
		return
	}

	d.log.LogInfo(logger.LogModel{Database: d.db, GuildID: i.GuildID, Function: "selectLogChannels()",
		Message: fmt.Sprintf("Got command [%s] from user [%s]", i.ApplicationCommandData().Name, i.Member.User.Username),
	})

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
		d.log.LogError(logger.LogModel{Database: d.db, GuildID: i.GuildID, Function: "selectModeratorChannels()",
			Message: fmt.Sprintf("Error deferring response: %s", err),
		})
		return
	}

	d.log.LogInfo(logger.LogModel{Database: d.db, GuildID: i.GuildID, Function: "selectModeratorChannels()",
		Message: fmt.Sprintf("Got command [%s] from user [%s]", i.ApplicationCommandData().Name, i.Member.User.Username),
	})

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
		d.log.LogError(logger.LogModel{Database: d.db, GuildID: i.GuildID, Function: "selectModeratorRoles()",
			Message: fmt.Sprintf("Error deferring response: %s", err),
		})
		return
	}

	d.log.LogInfo(logger.LogModel{Database: d.db, GuildID: i.GuildID, Function: "selectModeratorRoles()",
		Message: fmt.Sprintf("Got command [%s] from user [%s]", i.ApplicationCommandData().Name, i.Member.User.Username),
	})

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
