package api

import (
	"fmt"
	"strings"
	"time"

	"github.com/bwmarrin/discordgo"
	"github.com/dragonis41/discord-bot-moderation/pkg/logger"
	"github.com/dragonis41/discord-bot-moderation/pkg/model"
)

// buildReportActionButtons creates the action buttons for report messages
func buildReportActionButtons(userID string) []discordgo.MessageComponent {
	return []discordgo.MessageComponent{
		discordgo.ActionsRow{
			Components: []discordgo.MessageComponent{
				discordgo.Button{
					Label:    "Kick",
					Style:    discordgo.SecondaryButton,
					CustomID: fmt.Sprintf("report_kick_%s", userID),
					Emoji: &discordgo.ComponentEmoji{
						Name: "👢",
					},
				},
				discordgo.Button{
					Label:    "Ban",
					Style:    discordgo.DangerButton,
					CustomID: fmt.Sprintf("report_ban_%s", userID),
					Emoji: &discordgo.ComponentEmoji{
						Name: "🔨",
					},
				},
				discordgo.Button{
					Label:    "Ignore",
					Style:    discordgo.PrimaryButton,
					CustomID: fmt.Sprintf("report_ignore_%s", userID),
					Emoji: &discordgo.ComponentEmoji{
						Name: "✖️",
					},
				},
			},
		},
	}
}

func (d *Discord) handleReportActions(s *discordgo.Session, i *discordgo.InteractionCreate) {
	// Check if this is a message component interaction
	if i.Type != discordgo.InteractionMessageComponent {
		return
	}

	data := i.MessageComponentData()
	customID := data.CustomID

	// Check if this is a report action button
	if !strings.HasPrefix(customID, "report_kick_") &&
		!strings.HasPrefix(customID, "report_ban_") &&
		!strings.HasPrefix(customID, "report_ignore_") {
		return
	}

	// Defer the response
	err := s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseDeferredChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Flags: discordgo.MessageFlagsEphemeral,
		},
	})
	if err != nil {
		d.log.LogError(logger.LogModel{
			Database: d.db,
			GuildID:  i.GuildID,
			Function: "handleReportActions()",
			Message:  fmt.Sprintf("Error deferring response: %s", err),
		})
		return
	}

	// Check if user has moderation permissions
	if !d.db.CheckModerationPermissionOnInteraction(s, i) {
		return
	}

	// Extract user ID from custom ID
	var userID string
	var action string
	if strings.HasPrefix(customID, "report_kick_") {
		userID = strings.TrimPrefix(customID, "report_kick_")
		action = "kick"
	} else if strings.HasPrefix(customID, "report_ban_") {
		userID = strings.TrimPrefix(customID, "report_ban_")
		action = "ban"
	} else {
		userID = strings.TrimPrefix(customID, "report_ignore_")
		action = "ignore"
	}

	// Get user info
	ReportedUser, err := s.User(userID)
	if err != nil {
		d.log.LogError(logger.LogModel{
			Database: d.db,
			GuildID:  i.GuildID,
			Function: "handleReportActions()",
			Message:  fmt.Sprintf("Error fetching user: %s", err),
		})
		_, _ = s.FollowupMessageCreate(i.Interaction, true, &discordgo.WebhookParams{
			Embeds: []*discordgo.MessageEmbed{
				{
					Title:       "Erreur",
					Description: "Impossible de récupérer les informations de l'utilisateur.",
					Color:       model.Red.Int(),
					Timestamp:   time.Now().Format(time.RFC3339),
				},
			},
		})
		return
	}

	// Perform the action
	var actionErr error
	var successMessage string
	var logMessage string

	switch action {
	case "kick":
		actionErr = s.GuildMemberDeleteWithReason(i.GuildID, userID, fmt.Sprintf("Expulsé le %s suite à un signalement", time.Now().Format("02/01/2006 à 15:04")))
		successMessage = fmt.Sprintf("✅ L'utilisateur **%s** a été expulsé du serveur.", ReportedUser.Username)
		logMessage = fmt.Sprintf("User [%s] kicked user [%s] via report", i.Member.User.Username, ReportedUser.Username)
	case "ban":
		actionErr = s.GuildBanCreateWithReason(i.GuildID, userID, fmt.Sprintf("Banni le %s suite à un signalement", time.Now().Format("02/01/2006 à 15:04")), 1)
		successMessage = fmt.Sprintf("✅ L'utilisateur **%s** a été banni du serveur.", ReportedUser.Username)
		logMessage = fmt.Sprintf("User [%s] banned user [%s] via report", i.Member.User.Username, ReportedUser.Username)
	case "ignore":
		successMessage = fmt.Sprintf("✅ Le signalement de l'utilisateur **%s** a été ignoré.", ReportedUser.Username)
		logMessage = fmt.Sprintf("User [%s] ignored report for user [%s]", i.Member.User.Username, ReportedUser.Username)
	default:
		successMessage = fmt.Sprintf("✅ Le signalement de l'utilisateur **%s** a été ignoré.", ReportedUser.Username)
		logMessage = fmt.Sprintf("User [%s] ignored report for user [%s]", i.Member.User.Username, ReportedUser.Username)
	}

	if actionErr != nil {
		d.log.LogError(logger.LogModel{
			Database: d.db,
			GuildID:  i.GuildID,
			Function: "handleReportActions()",
			Message:  fmt.Sprintf("Error performing %s: %s", action, actionErr),
		})
		_, _ = s.FollowupMessageCreate(i.Interaction, true, &discordgo.WebhookParams{
			Embeds: []*discordgo.MessageEmbed{
				{
					Title:       "Erreur",
					Description: fmt.Sprintf("Impossible d'effectuer l'action. Vérifiez que le bot a les permissions nécessaires et que l'utilisateur est toujours sur le serveur."),
					Color:       model.Red.Int(),
					Timestamp:   time.Now().Format(time.RFC3339),
				},
			},
		})
		return
	}

	d.log.LogSuccess(logger.LogModel{
		Database: d.db,
		GuildID:  i.GuildID,
		Function: "handleReportActions()",
		Message:  logMessage,
	})

	// Send success message
	_, err = s.FollowupMessageCreate(i.Interaction, true, &discordgo.WebhookParams{
		Embeds: []*discordgo.MessageEmbed{
			{
				Title:       "Action effectuée",
				Description: successMessage,
				Color:       model.Green.Int(),
				Timestamp:   time.Now().Format(time.RFC3339),
			},
		},
	})
	if err != nil {
		d.log.LogError(logger.LogModel{
			Database: d.db,
			GuildID:  i.GuildID,
			Function: "handleReportActions()",
			Message:  fmt.Sprintf("Error sending follow-up message: %s", err),
		})
	}

	// Update the original report message to show the action was taken
	updatedEmbed := i.Message.Embeds[0]
	updatedEmbed.Footer = &discordgo.MessageEmbedFooter{
		Text: fmt.Sprintf("%s effectué par %s le %s",
			action,
			i.Member.User.Username,
			time.Now().Format("02/01/2006 à 15:04"),
		),
	}

	// Remove buttons
	_, err = s.ChannelMessageEditComplex(&discordgo.MessageEdit{
		Channel:    i.ChannelID,
		ID:         i.Message.ID,
		Embeds:     &[]*discordgo.MessageEmbed{updatedEmbed},
		Components: &[]discordgo.MessageComponent{},
	})
	if err != nil {
		d.log.LogError(logger.LogModel{
			Database: d.db,
			GuildID:  i.GuildID,
			Function: "handleReportActions()",
			Message:  fmt.Sprintf("Error updating message: %s", err),
		})
	}
}
