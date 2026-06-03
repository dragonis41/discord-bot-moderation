package api

import (
	"fmt"
	"strings"
	"time"

	"github.com/bwmarrin/discordgo"
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
	if !d.deferEphemeral(s, i, "handleReportActions()") {
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
		d.logError(i.GuildID, "handleReportActions()", "Error fetching user: %s", err)
		d.followup(s, i, "handleReportActions()", errorEmbed("Erreur", "Impossible de récupérer les informations de l'utilisateur."))
		return
	}

	guildName := guildNameByID(s, i.GuildID)

	// Perform the action
	var actionErr error
	var successMessage string
	var logMessage string

	switch action {
	case "kick":
		reason := fmt.Sprintf("Expulsé le %s suite à un signalement", time.Now().UTC().Format("02/01/2006 à 15:04 UTC"))
		d.sendPrivateMessageOnInteraction(s, i, ReportedUser, fmt.Sprintf("👢 **Expulsion**\n\nVous avez été expulsé du serveur [%s] le <t:%d:f> pour la raison suivante : \n`Expulsé suite à un signalement`", guildName, time.Now().Unix()))
		actionErr = s.GuildMemberDeleteWithReason(i.GuildID, userID, reason)
		successMessage = fmt.Sprintf("✅ L'utilisateur **%s** a été expulsé du serveur.", ReportedUser.Username)
		logMessage = fmt.Sprintf("User [%s] kicked user [%s] via report", i.Member.User.Username, ReportedUser.Username)
	case "ban":
		reason := fmt.Sprintf("Banni le %s suite à un signalement", time.Now().UTC().Format("02/01/2006 à 15:04 UTC"))
		d.sendPrivateMessageOnInteraction(s, i, ReportedUser, fmt.Sprintf("🔨 **Banissement**\n\nVous avez été banni du serveur [%s] le <t:%d:f> pour la raison suivante : \n`Banni suite à un signalement`", guildName, time.Now().Unix()))
		actionErr = s.GuildBanCreateWithReason(i.GuildID, userID, reason, 1)
		successMessage = fmt.Sprintf("✅ L'utilisateur **%s** a été banni du serveur.", ReportedUser.Username)
		logMessage = fmt.Sprintf("User [%s] banned user [%s] via report", i.Member.User.Username, ReportedUser.Username)
	default:
		successMessage = fmt.Sprintf("✅ Le signalement de l'utilisateur **%s** a été ignoré.", ReportedUser.Username)
		logMessage = fmt.Sprintf("User [%s] ignored report for user [%s]", i.Member.User.Username, ReportedUser.Username)
	}

	if actionErr != nil {
		d.logError(i.GuildID, "handleReportActions()", "Error performing %s: %s", action, actionErr)
		d.followup(s, i, "handleReportActions()", errorEmbed("Erreur", "Impossible d'effectuer l'action. Vérifiez que le bot a les permissions nécessaires et que l'utilisateur est toujours sur le serveur."))
		return
	}

	d.logSuccess(i.GuildID, "handleReportActions()", "%s", logMessage)

	// Send success message
	d.followup(s, i, "handleReportActions()", successEmbed("Action effectuée", successMessage))

	// Update the original report message to show the action was taken
	updatedEmbed := i.Message.Embeds[0]
	// Use Discord's timestamp in embed description for local time display
	if updatedEmbed.Description != "" {
		updatedEmbed.Description += fmt.Sprintf("\n\n%s effectué par %s le <t:%d:f>", action, i.Member.User.Username, time.Now().Unix())
	} else {
		updatedEmbed.Description = fmt.Sprintf("%s effectué par %s le <t:%d:f>", action, i.Member.User.Username, time.Now().Unix())
	}
	updatedEmbed.Footer = nil

	// Remove buttons
	_, err = s.ChannelMessageEditComplex(&discordgo.MessageEdit{
		Channel:    i.ChannelID,
		ID:         i.Message.ID,
		Embeds:     &[]*discordgo.MessageEmbed{updatedEmbed},
		Components: &[]discordgo.MessageComponent{},
	})
	if err != nil {
		d.logError(i.GuildID, "handleReportActions()", "Error updating message: %s", err)
	}
}
