package api

import (
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/bwmarrin/discordgo"
	"github.com/dragonis41/discord-bot-moderation/pkg/utils"
)

type DiscordFunctionInterface interface {
	displayConnectedGuilds()
	reportUser(s *discordgo.Session, i *discordgo.InteractionCreate)
	showHelp(s *discordgo.Session, i *discordgo.InteractionCreate)
}

var (
	red   = 0xff0000
	green = 0x00dd00
	blue  = 0x0099ff

	defaultFooter = &discordgo.MessageEmbedFooter{Text: "💡 Hint: Utilisez /help pour lister les commandes disponibles."}
)

func (d *Discord) displayConnectedGuilds() {
	fmt.Printf("\n======= Connected Servers =======\n")
	for _, guild := range d.client.State.Guilds {
		fmt.Printf("Server: (ID: %s)\n", guild.ID)
	}
	fmt.Printf("=================================\n\n")
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
		utils.LogError(fmt.Sprintf("reportUser: Error deferring response: %s", err))
		return
	}

	options := i.ApplicationCommandData().Options
	var reportedUser *discordgo.User
	reason := ""

	// Parse options
	for _, option := range options {
		switch option.Name {
		case "user":
			reportedUser, err = s.User(option.StringValue())
			if err != nil {
				utils.LogError(fmt.Sprintf("reportUser: Error fetching reported user: %s", err))
				_, _ = s.FollowupMessageCreate(i.Interaction, true, &discordgo.WebhookParams{
					Embeds: []*discordgo.MessageEmbed{
						{
							Title:       "Report",
							Description: "Une erreur est survenue lors du traitement de votre demande. Contactez un modérateur.",
							Color:       red,
							Timestamp:   time.Now().Format(time.RFC3339),
						},
					},
				})
				return
			}
		case "reason":
			reason = option.StringValue()
		}
	}

	if reportedUser == nil {
		utils.LogError("reportUser: Reported user is nil")
		_, _ = s.FollowupMessageCreate(i.Interaction, true, &discordgo.WebhookParams{
			Embeds: []*discordgo.MessageEmbed{
				{
					Title:       "Report",
					Description: "L'utilisateur spécifié est introuvable. Veuillez vérifier l'ID et réessayer.",
					Color:       red,
					Timestamp:   time.Now().Format(time.RFC3339),
				},
			},
		})
		return
	}

	utils.LogInfo(fmt.Sprintf("Got command [%s] from user [%s] to report the user [%s] for the reason [%s]",
		i.ApplicationCommandData().Name,
		i.Member.User.Username,
		reportedUser.Username,
		reason,
	))

	// TODO : Get this list from the database
	roleNames := []string{"sudoers"}

	// Get the list of all roles in the guild
	roles, err := s.GuildRoles(i.GuildID)
	if err != nil {
		utils.LogError(fmt.Sprintf("reportUser: Error fetching guild roles: %s", err))
		_, _ = s.FollowupMessageCreate(i.Interaction, true, &discordgo.WebhookParams{
			Embeds: []*discordgo.MessageEmbed{
				{
					Title:       "Report",
					Description: "Une erreur est survenue lors du traitement de votre demande. Contactez un modérateur.",
					Color:       red,
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
	for _, name := range roleNames {
		if id, exists := roleMap[name]; exists {
			modRoleMentions = fmt.Sprintf("%s<@&%s> ", modRoleMentions, id)
		}
	}

	// Send a message in the mod channel
	// TODO : Get the mod channel ID from the database
	modChannelID := os.Getenv("DISCORD_MOD_CHANNEL_ID")

	_, err = s.ChannelMessageSendEmbed(modChannelID, &discordgo.MessageEmbed{
		Title:       fmt.Sprintf("🚨 Nouveau signalement par <@%s>", i.Member.User.ID),
		Description: fmt.Sprintf("%s\n**Utilisateur**: <@&%s> (ID : %s)\n**Salon**: <#%s>\n**Raison** : %s", modRoleMentions, reportedUser.ID, reportedUser.ID, i.ChannelID, reason),
		Color:       red,
		Timestamp:   time.Now().Format(time.RFC3339),
	})
	if err != nil {
		utils.LogError(fmt.Sprintf("reportUser: Error sending message to mod channel: %s", err))
		_, _ = s.FollowupMessageCreate(i.Interaction, true, &discordgo.WebhookParams{
			Embeds: []*discordgo.MessageEmbed{
				{
					Title:       "Report",
					Description: "Une erreur est survenue lors du traitement de votre demande. Contactez un modérateur.",
					Color:       red,
					Timestamp:   time.Now().Format(time.RFC3339),
				},
			},
		})
		return
	}

	// Tell the user that the report has been received
	_, err = s.FollowupMessageCreate(i.Interaction, true, &discordgo.WebhookParams{
		Embeds: []*discordgo.MessageEmbed{
			{
				Title:       "Report",
				Description: fmt.Sprintf("L'utilisateur %s a été signalé à la moderation", reportedUser.Username),
				Color:       green,
				Footer:      &discordgo.MessageEmbedFooter{Text: "Merci de rendre ce serveur meilleur."},
				Timestamp:   time.Now().Format(time.RFC3339),
			},
		},
	})
	if err != nil {
		utils.LogError(fmt.Sprintf("reportUser: Error sending followup message: %s", err))
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
		utils.LogError(fmt.Sprintf("showHelp: Error deferring response: %s", err))
		return
	}

	utils.LogInfo(fmt.Sprintf("Got command [%s] from user [%s] asking for help", i.ApplicationCommandData().Name, i.Member.User.Username))

	var fields []*discordgo.MessageEmbedField
	commands, err := s.ApplicationCommands(d.client.State.User.ID, i.GuildID)
	if err != nil {
		utils.LogError(fmt.Sprintf("showHelp: Error fetching application commands: %s", err))
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
		Color:       blue,
		Fields:      fields,
		Footer:      defaultFooter,
		Timestamp:   time.Now().Format(time.RFC3339),
	}

	_, err = s.FollowupMessageCreate(i.Interaction, true, &discordgo.WebhookParams{
		Embeds: []*discordgo.MessageEmbed{embed},
	})
	if err != nil {
		utils.LogError(fmt.Sprintf("showHelp: Error sending followup message: %s", err))
	}
}
