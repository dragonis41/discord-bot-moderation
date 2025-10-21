package api

import (
	"fmt"
	"strings"
	"time"

	"github.com/bwmarrin/discordgo"
	"github.com/dragonis41/discord-bot-moderation/pkg/logger"
	"github.com/dragonis41/discord-bot-moderation/pkg/model"
)

type DiscordUserFunctionInterface interface {
	reportUser(s *discordgo.Session, i *discordgo.InteractionCreate)
	showHelp(s *discordgo.Session, i *discordgo.InteractionCreate)
}

func (d *Discord) reportUser(s *discordgo.Session, i *discordgo.InteractionCreate) {
	// Defer the response
	err := s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseDeferredChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Flags: discordgo.MessageFlagsEphemeral,
		},
	})
	if err != nil {
		d.log.LogError(logger.LogModel{Database: d.db, GuildID: i.GuildID, Function: "reportUser()",
			Message: fmt.Sprintf("Error deferring response: %s", err),
		})
		return
	}

	options := i.ApplicationCommandData().Options
	var reportedUser *discordgo.User
	reason := ""

	// Parse options
	for _, option := range options {
		switch option.Name {
		case "user":
			reportedUser = option.UserValue(s)
		case "reason":
			reason = option.StringValue()
		}
	}

	if reportedUser == nil {
		d.log.LogError(logger.LogModel{Database: d.db, GuildID: i.GuildID, Function: "reportUser()",
			Message: "Reported user is nil",
		})
		_, _ = s.FollowupMessageCreate(i.Interaction, true, &discordgo.WebhookParams{
			Embeds: []*discordgo.MessageEmbed{
				{
					Title:       "Report",
					Description: "L'utilisateur spécifié est introuvable. Veuillez vérifier l'ID et réessayer.",
					Color:       model.Red.Int(),
					Footer:      model.DefaultFooter,
					Timestamp:   time.Now().Format(time.RFC3339),
				},
			},
		})
		return
	}

	d.log.LogInfo(logger.LogModel{Database: d.db, GuildID: i.GuildID, Function: "reportUser()",
		Message: fmt.Sprintf("Got command [%s] from user [%s] to report the user [%s] for the reason [%s]",
			i.ApplicationCommandData().Name,
			i.Member.User.Username,
			reportedUser.Username,
			reason,
		),
	})
	// TODO : Log this report in the database

	// TODO : Get this list from the database
	adminRoleNames := []string{"sudoers"}

	// Get the list of all roles in the guild
	roles, err := s.GuildRoles(i.GuildID)
	if err != nil {
		d.log.LogError(logger.LogModel{Database: d.db, GuildID: i.GuildID, Function: "reportUser()",
			Message: fmt.Sprintf("Error fetching guild roles: %s", err),
		})
		_, _ = s.FollowupMessageCreate(i.Interaction, true, &discordgo.WebhookParams{
			Embeds: []*discordgo.MessageEmbed{
				{
					Title:       "Report",
					Description: "Une erreur est survenue lors du traitement de votre demande. Contactez un modérateur.",
					Color:       model.Red.Int(),
					Footer:      model.DefaultFooter,
					Timestamp:   time.Now().Format(time.RFC3339),
				},
			},
		})
		return
	}
	roleMap := make(map[string]string) // name -> id
	for _, role := range roles {
		roleMap[role.Name] = role.ID
	}
	modRoleMentions := ""
	for _, name := range adminRoleNames {
		if id, exists := roleMap[name]; exists {
			modRoleMentions = fmt.Sprintf("%s<@&%s> ", modRoleMentions, id)
		}
	}

	selectedChannels, err := d.db.GetModerationChannelsByGuildId(i.GuildID)
	if err != nil {
		d.log.LogError(logger.LogModel{Database: d.db, GuildID: i.GuildID, Function: "reportUser()",
			Message: fmt.Sprintf("Error fetching moderation channels from database: %s", err),
		})
		_, _ = s.FollowupMessageCreate(i.Interaction, true, &discordgo.WebhookParams{
			Embeds: []*discordgo.MessageEmbed{
				{
					Title:       "Report",
					Description: "Une erreur est survenue lors du traitement de votre demande. Contactez un modérateur.",
					Color:       model.Red.Int(),
					Footer:      model.DefaultFooter,
					Timestamp:   time.Now().Format(time.RFC3339),
				},
			},
		})
		return
	}

	// Send a message in the mod channel
	if len(selectedChannels) == 0 {
		d.log.LogError(logger.LogModel{Database: d.db, GuildID: i.GuildID, Function: "reportUser()",
			Message: "No moderation channels configured for this guild",
		})
		_, _ = s.FollowupMessageCreate(i.Interaction, true, &discordgo.WebhookParams{
			Embeds: []*discordgo.MessageEmbed{
				{
					Title:       "Report",
					Description: "Aucun canal de modération n'est configuré pour ce serveur. Veuillez contacter un administrateur.",
					Color:       model.Red.Int(),
					Footer:      model.DefaultFooter,
					Timestamp:   time.Now().Format(time.RFC3339),
				},
			},
		})
		return
	}
	for _, channelID := range selectedChannels {
		_, err = s.ChannelMessageSendComplex(channelID, &discordgo.MessageSend{
			Content: modRoleMentions,
			Embed: &discordgo.MessageEmbed{
				Title: fmt.Sprintf("🚨 Nouveau signalement par %s", i.Member.User.Username),
				Description: fmt.Sprintf("**Utilisateur signalé**: <@!%s> (ID : %s)\n**Salon**: <#%s>\n**Raison**: %s",
					reportedUser.ID,
					reportedUser.ID,
					i.ChannelID,
					reason,
				),
				Color:     model.Red.Int(),
				Timestamp: time.Now().Format(time.RFC3339),
			},
		})
		if err != nil {
			d.log.LogError(logger.LogModel{Database: d.db, GuildID: i.GuildID, Function: "reportUser()",
				Message: fmt.Sprintf("Error sending message to mod channel %s: %s", channelID, err),
			})
			continue
		}
	}

	// Tell the user that the report has been received
	_, err = s.FollowupMessageCreate(i.Interaction, true, &discordgo.WebhookParams{
		Embeds: []*discordgo.MessageEmbed{
			{
				Title:       "Report",
				Description: fmt.Sprintf("L'utilisateur %s a été signalé à la moderation", reportedUser.Username),
				Color:       model.Green.Int(),
				Footer:      &discordgo.MessageEmbedFooter{Text: "Merci de rendre ce serveur plus sain."},
				Timestamp:   time.Now().Format(time.RFC3339),
			},
		},
	})
	if err != nil {
		d.log.LogError(logger.LogModel{Database: d.db, GuildID: i.GuildID, Function: "reportUser()",
			Message: fmt.Sprintf("Error sending follow-up message: %s", err),
		})
	}
}

func (d *Discord) showHelp(s *discordgo.Session, i *discordgo.InteractionCreate) {
	// Defer the response
	err := s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseDeferredChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Flags: discordgo.MessageFlagsEphemeral,
		},
	})
	if err != nil {
		d.log.LogError(logger.LogModel{Database: d.db, GuildID: i.GuildID, Function: "showHelp()",
			Message: fmt.Sprintf("Error deferring response: %s", err),
		})
		return
	}

	d.log.LogInfo(logger.LogModel{Database: d.db, GuildID: i.GuildID, Function: "showHelp()",
		Message: fmt.Sprintf("Got command [%s] from user [%s]", i.ApplicationCommandData().Name, i.Member.User.Username),
	})

	var fields []*discordgo.MessageEmbedField
	commands, err := s.ApplicationCommands(d.client.State.User.ID, i.GuildID)
	if err != nil {
		d.log.LogError(logger.LogModel{Database: d.db, GuildID: i.GuildID, Function: "showHelp()",
			Message: fmt.Sprintf("Error fetching application commands: %s", err),
		})
	}
	for _, cmd := range commands {
		var field discordgo.MessageEmbedField
		field.Inline = false

		switch {
		case strings.HasPrefix(cmd.Name, "help"):
			field.Name = "💡 `/" + cmd.Name + "`"
		default:
			field.Name = "`/" + cmd.Name + "`"
		}

		value := fmt.Sprintf("%s\n", cmd.Description)
		if len(cmd.Options) > 0 {
			value += "**Options**:\n"
			var requiredOptions string
			var optionalOptions string
			for _, option := range cmd.Options {
				if option.Required {
					requiredOptions += fmt.Sprintf("• `%s`: %s\n", option.Name, option.Description)
				} else {
					optionalOptions += fmt.Sprintf("• `%s`: %s\n", option.Name, option.Description)
				}
			}
			if len(requiredOptions) > 0 {
				value += fmt.Sprintf("**Requis :**\n%s", requiredOptions)
			}
			if len(optionalOptions) > 0 {
				value += fmt.Sprintf("**Optionel :**\n%s", optionalOptions)
			}
		}
		field.Value = value
		fields = append(fields, &field)
	}

	// Create the embed
	embed := &discordgo.MessageEmbed{
		Title:       "💡 Help",
		Description: "Voici la liste des commandes disponibles :",
		Color:       model.Blue.Int(),
		Fields:      fields,
		Timestamp:   time.Now().Format(time.RFC3339),
	}

	_, err = s.FollowupMessageCreate(i.Interaction, true, &discordgo.WebhookParams{
		Embeds: []*discordgo.MessageEmbed{embed},
	})
	if err != nil {
		d.log.LogError(logger.LogModel{Database: d.db, GuildID: i.GuildID, Function: "showHelp()",
			Message: fmt.Sprintf("Error sending follow-up message: %s", err),
		})
	}
}
