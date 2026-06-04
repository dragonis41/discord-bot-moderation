package api

import (
	"fmt"
	"time"

	"github.com/bwmarrin/discordgo"
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
		d.followup(s, i, "reportUser()", errorEmbed("Report", "L'utilisateur spécifié est introuvable. Veuillez vérifier l'ID et réessayer."))
		return
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
		d.followup(s, i, "reportUser()", errorEmbed("Report", "Aucun rôle de modération n'est configuré pour ce serveur. Veuillez contacter un administrateur."))
		return
	}
	modRoleMentions := ""
	for _, id := range modRoles {
		modRoleMentions += fmt.Sprintf("<@&%s> ", id)
	}

	selectedChannels, err := d.db.GetModerationChannelsByGuildId(i.GuildID)
	if err != nil {
		d.logError(i.GuildID, "reportUser()", "Error fetching moderation channels from database: %s", err)
		d.followup(s, i, "reportUser()", errorEmbed("Report", "Une erreur est survenue lors du traitement de votre demande. Contactez un modérateur."))
		return
	}
	if len(selectedChannels) == 0 {
		d.logError(i.GuildID, "reportUser()", "No moderation channels configured for this guild")
		d.followup(s, i, "reportUser()", errorEmbed("Report", "Aucun canal de modération n'est configuré pour ce serveur. Veuillez contacter un administrateur."))
		return
	}

	// Get recent messages from the reported user
	messages := d.getUserRecentMessagesString(i.GuildID, reportedUser, 10)

	description := truncate(fmt.Sprintf(
		"**Utilisateur signalé**: <@!%s> (%s | %s)\n"+
			"**Salon**: <#%s>\n"+
			"**Raison**: %s\n\n"+
			"**Messages récents**:\n%s",
		reportedUser.ID, reportedUser.Username, reportedUser.ID, i.ChannelID, reason, messages,
	), maxEmbedDescriptionLength)

	for _, channelID := range selectedChannels {
		_, err = s.ChannelMessageSendComplex(channelID, &discordgo.MessageSend{
			Content:    modRoleMentions,
			Components: buildReportActionButtons(reportedUser.ID),
			Embed: &discordgo.MessageEmbed{
				Title:       fmt.Sprintf("🚨 Nouveau signalement par %s", i.Member.User.Username),
				Description: description,
				Color:       model.Red.Int(),
				Footer:      &discordgo.MessageEmbedFooter{Text: "Aucune action n'a encore été prise."},
			},
		})
		if err != nil {
			d.logError(i.GuildID, "reportUser()", "Error sending message to mod channel %s: %s", channelID, err)
			continue
		}
	}

	// Tell the user that the report has been received
	d.followup(s, i, "reportUser()", &discordgo.MessageEmbed{
		Title:       "Report",
		Description: fmt.Sprintf("L'utilisateur %s a été signalé à la moderation", reportedUser.Username),
		Color:       model.Green.Int(),
		Footer:      &discordgo.MessageEmbedFooter{Text: "Merci de rendre ce serveur plus sain."},
		Timestamp:   time.Now().UTC().Format(time.RFC3339),
	})
}

func (d *Discord) showHelp(s *discordgo.Session, i *discordgo.InteractionCreate) {
	if !d.deferEphemeral(s, i, "showHelp()") {
		return
	}

	d.logCommand(i, "showHelp()")

	commands, err := s.ApplicationCommands(d.client.State.User.ID, i.GuildID)
	if err != nil {
		d.logError(i.GuildID, "showHelp()", "Error fetching application commands: %s", err)
	}

	var fields []*discordgo.MessageEmbedField
	for _, cmd := range commands {
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

		fields = append(fields, &discordgo.MessageEmbedField{
			Name:   fmt.Sprintf("----------------------------------------\n**`/%s`**\n", cmd.Name),
			Value:  value,
			Inline: false,
		})
	}

	d.followup(s, i, "showHelp()", &discordgo.MessageEmbed{
		Title:       "💡 Help",
		Description: "Voici la liste des commandes disponibles :",
		Color:       model.Blue.Int(),
		Fields:      fields,
		Timestamp:   time.Now().UTC().Format(time.RFC3339),
	})
}
