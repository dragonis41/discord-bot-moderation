package api

import (
	"fmt"
	"strings"
	"time"

	"github.com/bwmarrin/discordgo"
	"github.com/dragonis41/discord-bot-moderation/pkg/i18n"
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
	if !d.deferEphemeral(s, i, "handleReportActions()") {
		return
	}

	lang := d.lang(i.GuildID)

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
		d.followup(s, i, "handleReportActions()", errorEmbed(lang, i18n.T(lang, "report_action.error_title"), i18n.T(lang, "report_action.fetch_user_error")))
		return
	}

	guildName := guildNameByID(s, i.GuildID)
	modUsername := interactionUsername(i)

	// Perform the action
	var actionErr error
	var successMessage string
	var logMessage string
	var modAction model.ModerationLogAction
	var auditReason string

	switch action {
	case "kick":
		auditReason = i18n.T(lang, "report_action.audit_kick", time.Now().UTC().Format("02/01/2006 15:04 UTC"))
		d.sendPrivateMessageOnInteraction(s, i, ReportedUser, i18n.T(lang, "report_action.kick_dm", guildName, time.Now().Unix()))
		actionErr = s.GuildMemberDeleteWithReason(i.GuildID, userID, auditReason)
		successMessage = i18n.T(lang, "report_action.kicked", ReportedUser.Username)
		logMessage = fmt.Sprintf("User [%s] kicked user [%s] via report", modUsername, ReportedUser.Username)
		modAction = model.ActionKick
	case "ban":
		auditReason = i18n.T(lang, "report_action.audit_ban", time.Now().UTC().Format("02/01/2006 15:04 UTC"))
		d.sendPrivateMessageOnInteraction(s, i, ReportedUser, i18n.T(lang, "report_action.ban_dm", guildName, time.Now().Unix()))
		actionErr = s.GuildBanCreateWithReason(i.GuildID, userID, auditReason, 1)
		successMessage = i18n.T(lang, "report_action.banned", ReportedUser.Username)
		logMessage = fmt.Sprintf("User [%s] banned user [%s] via report", modUsername, ReportedUser.Username)
		modAction = model.ActionBan
	default:
		successMessage = i18n.T(lang, "report_action.ignored", ReportedUser.Username)
		logMessage = fmt.Sprintf("User [%s] ignored report for user [%s]", modUsername, ReportedUser.Username)
	}

	if actionErr != nil {
		d.logError(i.GuildID, "handleReportActions()", "Error performing %s: %s", action, actionErr)
		d.followup(s, i, "handleReportActions()", errorEmbed(lang, i18n.T(lang, "report_action.error_title"), i18n.T(lang, "report_action.action_error")))
		return
	}

	if modAction != "" {
		if err := d.db.AddModerationLogEntry(i.GuildID, modAction, ReportedUser.ID, ReportedUser.Username, "report_action_button", fmt.Sprintf("%s (by %s)", auditReason, modUsername)); err != nil {
			d.logError(i.GuildID, "handleReportActions()", "Error logging moderation action to database: %s", err)
		}
	}

	d.logSuccess(i.GuildID, "handleReportActions()", "%s", logMessage)

	// Send success message
	d.followup(s, i, "handleReportActions()", successEmbed(lang, i18n.T(lang, "report_action.done_title"), successMessage))

	if i.Message == nil {
		return
	}

	// Update the original report message to show the action was taken
	var updatedEmbed *discordgo.MessageEmbed
	if len(i.Message.Embeds) > 0 && i.Message.Embeds[0] != nil {
		embedCopy := *i.Message.Embeds[0]
		updatedEmbed = &embedCopy
	} else {
		updatedEmbed = &discordgo.MessageEmbed{}
	}
	// Use Discord's timestamp in embed description for local time display
	if updatedEmbed.Description != "" {
		updatedEmbed.Description += i18n.T(lang, "report_action.taken", action, modUsername, time.Now().Unix())
	} else {
		updatedEmbed.Description = strings.TrimPrefix(i18n.T(lang, "report_action.taken", action, modUsername, time.Now().Unix()), "\n\n")
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
