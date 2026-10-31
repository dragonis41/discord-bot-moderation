package api

import (
	"fmt"
	"strings"
	"time"

	"github.com/bwmarrin/discordgo"
	"github.com/dragonis41/discord-bot-moderation/pkg/i18n"
	"github.com/dragonis41/discord-bot-moderation/pkg/model"
)

type DiscordUserFunctionInterface interface {
	reportUser(s discordClient, i *discordgo.InteractionCreate)
	showHelp(s *discordgo.Session, i *discordgo.InteractionCreate)
}

func resolveUserOption(s discordClient, opt *discordgo.ApplicationCommandInteractionDataOption) *discordgo.User {
	if session, ok := s.(*discordgo.Session); ok && session != nil {
		return opt.UserValue(session)
	}
	u := opt.UserValue(nil)
	if u == nil || u.ID == "" {
		return nil
	}
	if full, err := s.User(u.ID); err == nil && full != nil {
		return full
	}
	return u
}

func buildReportAlertEmbed(lang i18n.Lang, pending *PendingReport, reportedUser *discordgo.User, messages string) *discordgo.MessageEmbed {
	reasonText := pending.InitialReason
	for _, entry := range pending.AdditionalReasons {
		reasonText += i18n.T(lang, "report.additional_reason", entry.Reason, entry.Reporter)
	}

	description := truncate(i18n.T(lang, "report.description",
		reportedUser.ID, reportedUser.Username, reportedUser.ID, pending.OriginChannelID, reasonText, messages,
	), maxEmbedDescriptionLength)

	return &discordgo.MessageEmbed{
		Title:       i18n.T(lang, "report.alert_title", pending.InitialReporter),
		Description: description,
		Color:       model.Red.Int(),
		Footer:      &discordgo.MessageEmbedFooter{Text: i18n.T(lang, "report.alert_footer")},
	}
}

func (d *Discord) reportUser(s discordClient, i *discordgo.InteractionCreate) {
	if !d.deferEphemeral(s, i, "reportUser()") {
		return
	}

	lang := d.lang(i.GuildID)

	var reportedUser *discordgo.User
	rawReason := ""
	for _, option := range i.ApplicationCommandData().Options {
		switch option.Name {
		case "user":
			reportedUser = resolveUserOption(s, option)
		case "reason":
			rawReason = strings.TrimSpace(option.StringValue())
		}
	}

	if reportedUser == nil {
		d.logError(i.GuildID, "reportUser()", "Reported user is nil")
		d.followup(s, i, "reportUser()", errorEmbed(lang, i18n.T(lang, "report.title"), i18n.T(lang, "report.user_not_found")))
		return
	}

	reason := rawReason
	if reason == "" {
		reason = i18n.T(lang, "report.no_reason")
	}

	reporterName := interactionUsername(i)
	d.logInfo(i.GuildID, "reportUser()", "Got command [%s] from user [%s] to report the user [%s] for the reason [%s]",
		i.ApplicationCommandData().Name, reporterName, reportedUser.Username, reason)

	// Log the report in the database
	if err := d.db.AddModerationLogEntry(i.GuildID, model.ActionReport, reportedUser.ID, reportedUser.Username, "report_user_command", fmt.Sprintf("Reported by %s for reason: %s", reporterName, reason)); err != nil {
		d.logError(i.GuildID, "reportUser()", "Error logging report to database: %s", err)
	}

	modRoles, err := d.db.GetModerationRolesByGuildId(i.GuildID)
	if err != nil {
		d.logError(i.GuildID, "reportUser()", "Error fetching moderation roles from database: %s", err)
	} else if len(modRoles) == 0 {
		d.logError(i.GuildID, "reportUser()", "No moderation roles configured for this guild")
		d.followup(s, i, "reportUser()", errorEmbed(lang, i18n.T(lang, "report.title"), i18n.T(lang, "report.no_mod_roles")))
		return
	}
	modRoleMentions := ""
	for _, id := range modRoles {
		modRoleMentions += fmt.Sprintf("<@&%s> ", id)
	}

	selectedChannels, err := d.db.GetModerationChannelsByGuildId(i.GuildID)
	if err != nil {
		d.logError(i.GuildID, "reportUser()", "Error fetching moderation channels from database: %s", err)
		d.followup(s, i, "reportUser()", errorEmbed(lang, i18n.T(lang, "report.title"), i18n.T(lang, "report.error_generic")))
		return
	}
	if len(selectedChannels) == 0 {
		d.logError(i.GuildID, "reportUser()", "No moderation channels configured for this guild")
		d.followup(s, i, "reportUser()", errorEmbed(lang, i18n.T(lang, "report.title"), i18n.T(lang, "report.no_mod_channels")))
		return
	}

	// Get recent messages from the reported user
	messages := d.getUserRecentMessagesString(i.GuildID, reportedUser, 10)

	// If a moderation message is already pending for this user, update it instead of creating a new one
	if d.cache != nil && d.cache.GetPendingReport(i.GuildID, reportedUser.ID) != nil {
		if rawReason != "" {
			pending := d.cache.AddPendingReportReason(i.GuildID, reportedUser.ID, reporterName, rawReason)
			if pending != nil {
				updatedEmbed := buildReportAlertEmbed(lang, pending, reportedUser, messages)
				for _, msgRef := range pending.Messages {
					_, err := s.ChannelMessageEditComplex(&discordgo.MessageEdit{
						Channel: msgRef.ChannelID,
						ID:      msgRef.MessageID,
						Embeds:  &[]*discordgo.MessageEmbed{updatedEmbed},
					})
					if err != nil {
						d.logError(i.GuildID, "reportUser()", "Error updating report message %s in mod channel %s: %s", msgRef.MessageID, msgRef.ChannelID, err)
					}
				}
			}
		}

		d.followup(s, i, "reportUser()", &discordgo.MessageEmbed{
			Title:       i18n.T(lang, "report.title"),
			Description: i18n.T(lang, "report.received", reportedUser.Username),
			Color:       model.Green.Int(),
			Footer:      &discordgo.MessageEmbedFooter{Text: i18n.T(lang, "report.received_footer")},
			Timestamp:   time.Now().UTC().Format(time.RFC3339),
		})
		return
	}

	pending := PendingReport{
		GuildID:          i.GuildID,
		ReportedUserID:   reportedUser.ID,
		ReportedUsername: reportedUser.Username,
		OriginChannelID:  i.ChannelID,
		InitialReporter:  reporterName,
		InitialReason:    reason,
	}
	alertEmbed := buildReportAlertEmbed(lang, &pending, reportedUser, messages)

	for _, channelID := range selectedChannels {
		sentMsg, err := s.ChannelMessageSendComplex(channelID, &discordgo.MessageSend{
			Content:    modRoleMentions,
			Components: buildReportActionButtons(reportedUser.ID),
			Embed:      alertEmbed,
		})
		if err != nil {
			d.logError(i.GuildID, "reportUser()", "Error sending message to mod channel %s: %s", channelID, err)
			continue
		}
		if sentMsg != nil && sentMsg.ID != "" {
			pending.Messages = append(pending.Messages, ReportMessageRef{
				ChannelID: channelID,
				MessageID: sentMsg.ID,
			})
		}
	}

	if d.cache != nil && len(pending.Messages) > 0 {
		d.cache.SetPendingReport(pending)
	}

	// Tell the user that the report has been received
	d.followup(s, i, "reportUser()", &discordgo.MessageEmbed{
		Title:       i18n.T(lang, "report.title"),
		Description: i18n.T(lang, "report.received", reportedUser.Username),
		Color:       model.Green.Int(),
		Footer:      &discordgo.MessageEmbedFooter{Text: i18n.T(lang, "report.received_footer")},
		Timestamp:   time.Now().UTC().Format(time.RFC3339),
	})
}

func (d *Discord) showHelp(s *discordgo.Session, i *discordgo.InteractionCreate) {
	if !d.deferEphemeral(s, i, "showHelp()") {
		return
	}

	d.logCommand(i, "showHelp()")

	lang := d.lang(i.GuildID)

	var commands []*discordgo.ApplicationCommand
	if s != nil && s.State != nil && s.State.User != nil {
		var err error
		commands, err = s.ApplicationCommands(s.State.User.ID, i.GuildID)
		if err != nil {
			d.logError(i.GuildID, "showHelp()", "Error fetching application commands: %s", err)
		}
	}
	if len(commands) == 0 {
		commands = d.buildSlashCommands()
	}

	var fields []*discordgo.MessageEmbedField
	for _, cmd := range commands {
		var cmdLocs map[discordgo.Locale]string
		if cmd.DescriptionLocalizations != nil {
			cmdLocs = *cmd.DescriptionLocalizations
		}
		value := fmt.Sprintf("%s\n", localizedDescription(lang, cmd.Description, cmdLocs))
		if len(cmd.Options) > 0 {
			value += i18n.T(lang, "help.options_label") + "\n"
			var requiredOptions string
			var optionalOptions string
			for _, option := range cmd.Options {
				optDesc := localizedDescription(lang, option.Description, option.DescriptionLocalizations)
				if option.Required {
					requiredOptions += fmt.Sprintf("• `%s`: %s\n", option.Name, optDesc)
				} else {
					optionalOptions += fmt.Sprintf("• `%s`: %s\n", option.Name, optDesc)
				}
			}
			if len(requiredOptions) > 0 {
				value += i18n.T(lang, "help.required_label") + "\n" + requiredOptions
			}
			if len(optionalOptions) > 0 {
				value += i18n.T(lang, "help.optional_label") + "\n" + optionalOptions
			}
		}

		fields = append(fields, &discordgo.MessageEmbedField{
			Name:   fmt.Sprintf("----------------------------------------\n**`/%s`**\n", cmd.Name),
			Value:  value,
			Inline: false,
		})
	}

	d.followup(s, i, "showHelp()", &discordgo.MessageEmbed{
		Title:       i18n.T(lang, "help.title"),
		Description: i18n.T(lang, "help.description"),
		Color:       model.Blue.Int(),
		Fields:      fields,
		Timestamp:   time.Now().UTC().Format(time.RFC3339),
	})
}

func localizedDescription(lang i18n.Lang, fallback string, locs map[discordgo.Locale]string) string {
	if locs != nil {
		switch lang {
		case i18n.FR:
			if s, ok := locs[discordgo.French]; ok && s != "" {
				return s
			}
		case i18n.ES:
			if s, ok := locs[discordgo.SpanishES]; ok && s != "" {
				return s
			}
		}
	}
	return fallback
}
