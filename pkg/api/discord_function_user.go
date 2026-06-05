package api

import (
	"fmt"
	"time"

	"github.com/bwmarrin/discordgo"
	"github.com/dragonis41/discord-bot-moderation/pkg/i18n"
	"github.com/dragonis41/discord-bot-moderation/pkg/model"
)

type DiscordUserFunctionInterface interface {
	reportUser(s *discordgo.Session, i *discordgo.InteractionCreate)
	showHelp(s *discordgo.Session, i *discordgo.InteractionCreate)
}

func (d *Discord) reportUser(s *discordgo.Session, i *discordgo.InteractionCreate) {
	if !d.deferEphemeral(s, i, "reportUser()") {
		return
	}

	lang := d.lang(i.GuildID)

	var reportedUser *discordgo.User
	reason := ""
	for _, option := range i.ApplicationCommandData().Options {
		switch option.Name {
		case "user":
			reportedUser = option.UserValue(s)
		case "reason":
			reason = option.StringValue()
		}
	}

	if reportedUser == nil {
		d.logError(i.GuildID, "reportUser()", "Reported user is nil")
		d.followup(s, i, "reportUser()", errorEmbed(lang, i18n.T(lang, "report.title"), i18n.T(lang, "report.user_not_found")))
		return
	}

	if reason == "" {
		reason = i18n.T(lang, "report.no_reason")
	}

	d.logInfo(i.GuildID, "reportUser()", "Got command [%s] from user [%s] to report the user [%s] for the reason [%s]",
		i.ApplicationCommandData().Name, i.Member.User.Username, reportedUser.Username, reason)

	// Log the report in the database
	if err := d.db.AddModerationLogEntry(i.GuildID, model.ActionReport, reportedUser.ID, reportedUser.Username, "report_user_command", fmt.Sprintf("Reported by %s for reason: %s", i.Member.User.Username, reason)); err != nil {
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

	description := truncate(i18n.T(lang, "report.description",
		reportedUser.ID, reportedUser.Username, reportedUser.ID, i.ChannelID, reason, messages,
	), maxEmbedDescriptionLength)

	for _, channelID := range selectedChannels {
		_, err = s.ChannelMessageSendComplex(channelID, &discordgo.MessageSend{
			Content:    modRoleMentions,
			Components: buildReportActionButtons(reportedUser.ID),
			Embed: &discordgo.MessageEmbed{
				Title:       i18n.T(lang, "report.alert_title", i.Member.User.Username),
				Description: description,
				Color:       model.Red.Int(),
				Footer:      &discordgo.MessageEmbedFooter{Text: i18n.T(lang, "report.alert_footer")},
			},
		})
		if err != nil {
			d.logError(i.GuildID, "reportUser()", "Error sending message to mod channel %s: %s", channelID, err)
			continue
		}
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

	commands, err := s.ApplicationCommands(d.client.State.User.ID, i.GuildID)
	if err != nil {
		d.logError(i.GuildID, "showHelp()", "Error fetching application commands: %s", err)
	}

	var fields []*discordgo.MessageEmbedField
	for _, cmd := range commands {
		value := fmt.Sprintf("%s\n", cmd.Description)
		if len(cmd.Options) > 0 {
			value += i18n.T(lang, "help.options_label") + "\n"
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
