package api

import (
	"fmt"

	"github.com/bwmarrin/discordgo"
	"github.com/dragonis41/discord-bot-moderation/pkg/logger"
	"github.com/dragonis41/discord-bot-moderation/pkg/model"
)

type DiscordLanguageFunctionInterface interface {
	handleLanguageCommand(s *discordgo.Session, i *discordgo.InteractionCreate)
	setLanguage(s *discordgo.Session, i *discordgo.InteractionCreate)
	showLanguage(s *discordgo.Session, i *discordgo.InteractionCreate)
}

// handleLanguageCommand routes language subcommands
func (d *Discord) handleLanguageCommand(s *discordgo.Session, i *discordgo.InteractionCreate) {
	options := i.ApplicationCommandData().Options
	if len(options) == 0 {
		return
	}

	// Route to appropriate subcommand
	switch options[0].Name {
	case "set":
		d.setLanguage(s, i)
	case "show":
		d.showLanguage(s, i)
	}
}

// setLanguage sets the language preference for the guild
func (d *Discord) setLanguage(s *discordgo.Session, i *discordgo.InteractionCreate) {
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
			Function: "setLanguage()",
			Message:  fmt.Sprintf("Error deferring response: %s", err),
		})
		return
	}

	d.log.LogInfo(logger.LogModel{
		Database: d.db,
		GuildID:  i.GuildID,
		Function: "setLanguage()",
		Message:  fmt.Sprintf("Got command [language set] from user [%s]", i.Member.User.Username),
	})

	// Check moderation permissions (it already sends error message)
	if !d.db.CheckModerationPermissionOnInteraction(s, i) {
		return
	}

	// Get the language code from subcommand options
	options := i.ApplicationCommandData().Options[0].Options
	if len(options) == 0 {
		return
	}

	languageCode := options[0].StringValue()

	// Validate language code
	validLanguages := map[string]bool{
		"en_US": true,
		"fr_FR": true,
	}

	if !validLanguages[languageCode] {
		// Get localized invalid language message
		msg, msgErr := d.localizer.GetMessage(i.GuildID, "language.invalid", nil)
		if msgErr != nil {
			msg = "❌ Invalid language code. Available languages: en_US, fr_FR"
		}

		_, err = s.InteractionResponseEdit(i.Interaction, &discordgo.WebhookEdit{
			Content: &msg,
		})
		if err != nil {
			d.log.LogError(logger.LogModel{
				Database: d.db,
				GuildID:  i.GuildID,
				Function: "setLanguage()",
				Message:  fmt.Sprintf("Error editing response: %s", err),
			})
		}
		return
	}

	// Set the language preference
	err = d.localizer.SetGuildLanguage(i.GuildID, languageCode)
	if err != nil {
		d.log.LogError(logger.LogModel{
			Database: d.db,
			GuildID:  i.GuildID,
			Function: "setLanguage()",
			Message:  fmt.Sprintf("Failed to set guild language: %s", err),
		})

		// Get localized error message (use the OLD language since new one might have failed)
		msg, msgErr := d.localizer.GetMessage(i.GuildID, "language.error", map[string]interface{}{
			"error": err.Error(),
		})
		if msgErr != nil {
			msg = fmt.Sprintf("❌ Failed to change language: %s", err.Error())
		}

		_, err = s.InteractionResponseEdit(i.Interaction, &discordgo.WebhookEdit{
			Content: &msg,
		})
		if err != nil {
			d.log.LogError(logger.LogModel{
				Database: d.db,
				GuildID:  i.GuildID,
				Function: "setLanguage()",
				Message:  fmt.Sprintf("Error editing response: %s", err),
			})
		}
		return
	}

	// Log success
	d.log.LogSuccess(logger.LogModel{
		Database: d.db,
		GuildID:  i.GuildID,
		Function: "setLanguage()",
		Message:  fmt.Sprintf("Language set to %s by user %s", languageCode, i.Member.User.Username),
	})

	// Get localized success message (in the NEW language!)
	languageName := map[string]string{
		"en_US": "English",
		"fr_FR": "Français",
	}

	msg, msgErr := d.localizer.GetMessage(i.GuildID, "language.changed", map[string]interface{}{
		"language": languageName[languageCode],
	})
	if msgErr != nil {
		msg = fmt.Sprintf("✅ Language changed to %s", languageName[languageCode])
	}

	// Create embed for better visual
	embed := &discordgo.MessageEmbed{
		Title:       "🌍 Language Settings",
		Description: msg,
		Color:       model.Green.Int(),
		Footer: &discordgo.MessageEmbedFooter{
			Text: "All user-facing messages will now be displayed in the selected language.",
		},
	}

	_, err = s.InteractionResponseEdit(i.Interaction, &discordgo.WebhookEdit{
		Embeds: &[]*discordgo.MessageEmbed{embed},
	})
	if err != nil {
		d.log.LogError(logger.LogModel{
			Database: d.db,
			GuildID:  i.GuildID,
			Function: "setLanguage()",
			Message:  fmt.Sprintf("Error editing response: %s", err),
		})
	}
}

// showLanguage shows the current language preference for the guild
func (d *Discord) showLanguage(s *discordgo.Session, i *discordgo.InteractionCreate) {
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
			Function: "showLanguage()",
			Message:  fmt.Sprintf("Error deferring response: %s", err),
		})
		return
	}

	d.log.LogInfo(logger.LogModel{
		Database: d.db,
		GuildID:  i.GuildID,
		Function: "showLanguage()",
		Message:  fmt.Sprintf("Got command [language show] from user [%s]", i.Member.User.Username),
	})

	// Get the current language
	language, err := d.localizer.GetGuildLanguage(i.GuildID)
	if err != nil {
		d.log.LogError(logger.LogModel{
			Database: d.db,
			GuildID:  i.GuildID,
			Function: "showLanguage()",
			Message:  fmt.Sprintf("Failed to get guild language: %s", err),
		})
		language = "en_US" // Default fallback
	}

	// Map language codes to display names
	languageNames := map[string]string{
		"en_US": "English (en_US)",
		"fr_FR": "Français (fr_FR)",
	}

	displayName, exists := languageNames[language]
	if !exists {
		displayName = language
	}

	// Get localized message
	msg, msgErr := d.localizer.GetMessage(i.GuildID, "language.current", map[string]interface{}{
		"language": displayName,
	})
	if msgErr != nil {
		msg = fmt.Sprintf("📢 Current language: %s", displayName)
	}

	// Create embed
	embed := &discordgo.MessageEmbed{
		Title:       "🌍 Language Settings",
		Description: msg,
		Color:       model.Blue.Int(),
		Fields: []*discordgo.MessageEmbedField{
			{
				Name:   "Available Languages",
				Value:  "• `en_US` - English\n• `fr_FR` - Français",
				Inline: false,
			},
		},
		Footer: &discordgo.MessageEmbedFooter{
			Text: "Use /language set to change the language",
		},
	}

	_, err = s.InteractionResponseEdit(i.Interaction, &discordgo.WebhookEdit{
		Embeds: &[]*discordgo.MessageEmbed{embed},
	})
	if err != nil {
		d.log.LogError(logger.LogModel{
			Database: d.db,
			GuildID:  i.GuildID,
			Function: "showLanguage()",
			Message:  fmt.Sprintf("Error editing response: %s", err),
		})
	}
}
