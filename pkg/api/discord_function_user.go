package api

import (
	"fmt"
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

	// Log the report in the database
	err = d.db.AddModerationLogEntry(i.GuildID, model.ActionReport, reportedUser.ID, "report_user_command", fmt.Sprintf("Reported by %s for reason: %s", i.Member.User.Username, reason))
	if err != nil {
		d.log.LogError(logger.LogModel{Database: d.db, GuildID: i.GuildID, Function: "reportUser()",
			Message: fmt.Sprintf("Error logging report to database: %s", err),
		})
	}

	modRoles, err := d.db.GetModerationRolesByGuildId(i.GuildID)
	if err != nil {
		d.log.LogError(logger.LogModel{Database: d.db, GuildID: i.GuildID, Function: "reportUser()",
			Message: fmt.Sprintf("Error fetching moderation roles from database: %s", err),
		})
	} else if len(modRoles) == 0 {
		d.log.LogError(logger.LogModel{Database: d.db, GuildID: i.GuildID, Function: "reportUser()",
			Message: "No moderation roles configured for this guild",
		})
		_, _ = s.FollowupMessageCreate(i.Interaction, true, &discordgo.WebhookParams{
			Embeds: []*discordgo.MessageEmbed{
				{
					Title:       "Report",
					Description: "Aucun rôle de modération n'est configuré pour ce serveur. Veuillez contacter un administrateur.",
					Color:       model.Red.Int(),
					Footer:      model.DefaultFooter,
					Timestamp:   time.Now().Format(time.RFC3339),
				},
			},
		})
		return
	}
	modRoleMentions := ""
	for _, id := range modRoles {
		modRoleMentions = fmt.Sprintf("%s<@&%s> ", modRoleMentions, id)
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

	// Get recent messages from the reported user
	messages := d.getUserRecentMessagesString(i.GuildID, reportedUser, 10)

	description := fmt.Sprintf(
		"**Utilisateur signalé**: <@!%s> (ID : %s)\n"+
			"**Salon**: <#%s>\n"+
			"**Raison**: %s\n\n"+
			"**Messages récents**:\n%s",
		reportedUser.ID,
		reportedUser.ID,
		i.ChannelID,
		reason,
		messages,
	)

	// Truncate description if it exceeds Discord's limit
	const maxDescriptionLength = 4096
	if len(description) > maxDescriptionLength {
		description = description[:maxDescriptionLength-3] + "..."
	}

	for _, channelID := range selectedChannels {
		_, err = s.ChannelMessageSendComplex(channelID, &discordgo.MessageSend{
			Content:    modRoleMentions,
			Components: buildReportActionButtons(reportedUser.ID),
			Embed: &discordgo.MessageEmbed{
				Title:       fmt.Sprintf("🚨 Nouveau signalement par %s", i.Member.User.Username),
				Description: description,
				Color:       model.Red.Int(),
				Footer: &discordgo.MessageEmbedFooter{
					Text: "Aucune action n'a encore été prise.",
				},
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

		field.Name = "----------------------------------------\n"
		field.Name += fmt.Sprintf("**`/%s`**\n", cmd.Name)

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
