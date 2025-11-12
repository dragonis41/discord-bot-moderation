package api

import (
	"fmt"
	"strings"
	"time"

	"github.com/bwmarrin/discordgo"
	"github.com/dragonis41/discord-bot-moderation/pkg/logger"
	"github.com/dragonis41/discord-bot-moderation/pkg/model"
)

type DiscordAutomodCommandsInterface interface {
	addBannedWord(s *discordgo.Session, i *discordgo.InteractionCreate)
	removeBannedWord(s *discordgo.Session, i *discordgo.InteractionCreate)
	listBannedWords(s *discordgo.Session, i *discordgo.InteractionCreate)
	addBannedWebsite(s *discordgo.Session, i *discordgo.InteractionCreate)
	removeBannedWebsite(s *discordgo.Session, i *discordgo.InteractionCreate)
	listBannedWebsites(s *discordgo.Session, i *discordgo.InteractionCreate)
	configureAutomod(s *discordgo.Session, i *discordgo.InteractionCreate)
}

func (d *Discord) addBannedWord(s *discordgo.Session, i *discordgo.InteractionCreate) {
	// Defer the response
	err := s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseDeferredChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Flags: discordgo.MessageFlagsEphemeral,
		},
	})
	if err != nil {
		d.log.LogError(logger.LogModel{Database: d.db, GuildID: i.GuildID, Function: "addBannedWord()",
			Message: fmt.Sprintf("Error deferring response: %s", err),
		})
		return
	}

	d.log.LogInfo(logger.LogModel{Database: d.db, GuildID: i.GuildID, Function: "addBannedWord()",
		Message: fmt.Sprintf("Got command [%s] from user [%s]", i.ApplicationCommandData().Name, i.Member.User.Username),
	})

	if !d.db.CheckModerationPermissionOnInteraction(s, i) {
		return
	}

	options := i.ApplicationCommandData().Options
	wordPattern := ""
	isRegex := false

	// Parse options
	for _, option := range options {
		switch option.Name {
		case "word":
			wordPattern = option.StringValue()
		case "is_regex":
			isRegex = option.BoolValue()
		}
	}

	if wordPattern == "" {
		_, _ = s.FollowupMessageCreate(i.Interaction, true, &discordgo.WebhookParams{
			Embeds: []*discordgo.MessageEmbed{
				{
					Title:       "Mot interdit",
					Description: "Le mot ne peut pas être vide.",
					Color:       model.Red.Int(),
					Footer:      model.DefaultFooter,
					Timestamp:   time.Now().Format(time.DateTime),
				},
			},
		})
		return
	}

	// Add to database
	err = d.db.AddBannedWord(i.GuildID, wordPattern, isRegex)
	if err != nil {
		d.log.LogError(logger.LogModel{Database: d.db, GuildID: i.GuildID, Function: "addBannedWord()",
			Message: fmt.Sprintf("Error adding banned word to database: %s", err),
		})
		_, _ = s.FollowupMessageCreate(i.Interaction, true, &discordgo.WebhookParams{
			Embeds: []*discordgo.MessageEmbed{
				{
					Title:       "Mot interdit",
					Description: "Une erreur est survenue lors de l'ajout du mot interdit.",
					Color:       model.Red.Int(),
					Footer:      model.DefaultFooter,
					Timestamp:   time.Now().Format(time.DateTime),
				},
			},
		})
		return
	}

	wordType := "littéral"
	if isRegex {
		wordType = "regex"
	}

	_, err = s.FollowupMessageCreate(i.Interaction, true, &discordgo.WebhookParams{
		Embeds: []*discordgo.MessageEmbed{
			{
				Title:       "✅ Mot interdit ajouté",
				Description: fmt.Sprintf("Le mot `%s` (type: %s) a été ajouté à la liste des mots interdits.", wordPattern, wordType),
				Color:       model.Green.Int(),
				Footer:      model.DefaultFooter,
				Timestamp:   time.Now().Format(time.DateTime),
			},
		},
	})
	if err != nil {
		d.log.LogError(logger.LogModel{Database: d.db, GuildID: i.GuildID, Function: "addBannedWord()",
			Message: fmt.Sprintf("Error sending follow-up message: %s", err),
		})
	}
}

func (d *Discord) removeBannedWord(s *discordgo.Session, i *discordgo.InteractionCreate) {
	// Defer the response
	err := s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseDeferredChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Flags: discordgo.MessageFlagsEphemeral,
		},
	})
	if err != nil {
		d.log.LogError(logger.LogModel{Database: d.db, GuildID: i.GuildID, Function: "removeBannedWord()",
			Message: fmt.Sprintf("Error deferring response: %s", err),
		})
		return
	}

	d.log.LogInfo(logger.LogModel{Database: d.db, GuildID: i.GuildID, Function: "removeBannedWord()",
		Message: fmt.Sprintf("Got command [%s] from user [%s]", i.ApplicationCommandData().Name, i.Member.User.Username),
	})

	if !d.db.CheckModerationPermissionOnInteraction(s, i) {
		return
	}

	options := i.ApplicationCommandData().Options
	wordID := int64(0)

	// Parse options
	for _, option := range options {
		if option.Name == "id" {
			wordID = option.IntValue()
		}
	}

	if wordID == 0 {
		_, _ = s.FollowupMessageCreate(i.Interaction, true, &discordgo.WebhookParams{
			Embeds: []*discordgo.MessageEmbed{
				{
					Title:       "Mot interdit",
					Description: "L'ID du mot est requis.",
					Color:       model.Red.Int(),
					Footer:      model.DefaultFooter,
					Timestamp:   time.Now().Format(time.DateTime),
				},
			},
		})
		return
	}

	// Remove from database
	err = d.db.RemoveBannedWord(i.GuildID, int(wordID))
	if err != nil {
		d.log.LogError(logger.LogModel{Database: d.db, GuildID: i.GuildID, Function: "removeBannedWord()",
			Message: fmt.Sprintf("Error removing banned word from database: %s", err),
		})
		_, _ = s.FollowupMessageCreate(i.Interaction, true, &discordgo.WebhookParams{
			Embeds: []*discordgo.MessageEmbed{
				{
					Title:       "Mot interdit",
					Description: "Une erreur est survenue lors de la suppression du mot interdit.",
					Color:       model.Red.Int(),
					Footer:      model.DefaultFooter,
					Timestamp:   time.Now().Format(time.DateTime),
				},
			},
		})
		return
	}

	_, err = s.FollowupMessageCreate(i.Interaction, true, &discordgo.WebhookParams{
		Embeds: []*discordgo.MessageEmbed{
			{
				Title:       "✅ Mot interdit supprimé",
				Description: fmt.Sprintf("Le mot avec l'ID `%d` a été supprimé de la liste des mots interdits.", wordID),
				Color:       model.Green.Int(),
				Footer:      model.DefaultFooter,
				Timestamp:   time.Now().Format(time.DateTime),
			},
		},
	})
	if err != nil {
		d.log.LogError(logger.LogModel{Database: d.db, GuildID: i.GuildID, Function: "removeBannedWord()",
			Message: fmt.Sprintf("Error sending follow-up message: %s", err),
		})
	}
}

func (d *Discord) listBannedWords(s *discordgo.Session, i *discordgo.InteractionCreate) {
	// Defer the response
	err := s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseDeferredChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Flags: discordgo.MessageFlagsEphemeral,
		},
	})
	if err != nil {
		d.log.LogError(logger.LogModel{Database: d.db, GuildID: i.GuildID, Function: "listBannedWords()",
			Message: fmt.Sprintf("Error deferring response: %s", err),
		})
		return
	}

	d.log.LogInfo(logger.LogModel{Database: d.db, GuildID: i.GuildID, Function: "listBannedWords()",
		Message: fmt.Sprintf("Got command [%s] from user [%s]", i.ApplicationCommandData().Name, i.Member.User.Username),
	})

	if !d.db.CheckModerationPermissionOnInteraction(s, i) {
		return
	}

	// Get banned words from database
	bannedWords, err := d.db.GetBannedWordsByGuildId(i.GuildID)
	if err != nil {
		d.log.LogError(logger.LogModel{Database: d.db, GuildID: i.GuildID, Function: "listBannedWords()",
			Message: fmt.Sprintf("Error fetching banned words from database: %s", err),
		})
		_, _ = s.FollowupMessageCreate(i.Interaction, true, &discordgo.WebhookParams{
			Embeds: []*discordgo.MessageEmbed{
				{
					Title:       "Mots interdits",
					Description: "Une erreur est survenue lors de la récupération des mots interdits.",
					Color:       model.Red.Int(),
					Footer:      model.DefaultFooter,
					Timestamp:   time.Now().Format(time.DateTime),
				},
			},
		})
		return
	}

	if len(bannedWords) == 0 {
		_, _ = s.FollowupMessageCreate(i.Interaction, true, &discordgo.WebhookParams{
			Embeds: []*discordgo.MessageEmbed{
				{
					Title:       "📝 Mots interdits",
					Description: "Aucun mot interdit n'est configuré pour ce serveur.",
					Color:       model.Blue.Int(),
					Footer:      model.DefaultFooter,
					Timestamp:   time.Now().Format(time.DateTime),
				},
			},
		})
		return
	}

	// Build the list
	var wordsList strings.Builder
	for _, word := range bannedWords {
		wordType := "littéral"
		if word.IsRegex {
			wordType = "regex"
		}
		wordsList.WriteString(fmt.Sprintf("• **ID %d**: `%s` (type: %s)\n", word.ID, word.WordPattern, wordType))
	}

	description := fmt.Sprintf("**Total**: %d mot(s) interdit(s)\n\n%s", len(bannedWords), wordsList.String())

	// Truncate if too long
	const maxDescriptionLength = 4096
	if len(description) > maxDescriptionLength {
		description = description[:maxDescriptionLength-3] + "..."
	}

	_, err = s.FollowupMessageCreate(i.Interaction, true, &discordgo.WebhookParams{
		Embeds: []*discordgo.MessageEmbed{
			{
				Title:       "📝 Liste des mots interdits",
				Description: description,
				Color:       model.Blue.Int(),
				Footer:      model.DefaultFooter,
				Timestamp:   time.Now().Format(time.DateTime),
			},
		},
	})
	if err != nil {
		d.log.LogError(logger.LogModel{Database: d.db, GuildID: i.GuildID, Function: "listBannedWords()",
			Message: fmt.Sprintf("Error sending follow-up message: %s", err),
		})
	}
}

func (d *Discord) addBannedWebsite(s *discordgo.Session, i *discordgo.InteractionCreate) {
	// Defer the response
	err := s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseDeferredChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Flags: discordgo.MessageFlagsEphemeral,
		},
	})
	if err != nil {
		d.log.LogError(logger.LogModel{Database: d.db, GuildID: i.GuildID, Function: "addBannedWebsite()",
			Message: fmt.Sprintf("Error deferring response: %s", err),
		})
		return
	}

	d.log.LogInfo(logger.LogModel{Database: d.db, GuildID: i.GuildID, Function: "addBannedWebsite()",
		Message: fmt.Sprintf("Got command [%s] from user [%s]", i.ApplicationCommandData().Name, i.Member.User.Username),
	})

	if !d.db.CheckModerationPermissionOnInteraction(s, i) {
		return
	}

	options := i.ApplicationCommandData().Options
	websiteURL := ""

	// Parse options
	for _, option := range options {
		if option.Name == "url" {
			websiteURL = option.StringValue()
		}
	}

	if websiteURL == "" {
		_, _ = s.FollowupMessageCreate(i.Interaction, true, &discordgo.WebhookParams{
			Embeds: []*discordgo.MessageEmbed{
				{
					Title:       "Site web interdit",
					Description: "L'URL ne peut pas être vide.",
					Color:       model.Red.Int(),
					Footer:      model.DefaultFooter,
					Timestamp:   time.Now().Format(time.DateTime),
				},
			},
		})
		return
	}

	// Add to database
	err = d.db.AddBannedWebsite(i.GuildID, websiteURL)
	if err != nil {
		d.log.LogError(logger.LogModel{Database: d.db, GuildID: i.GuildID, Function: "addBannedWebsite()",
			Message: fmt.Sprintf("Error adding banned website to database: %s", err),
		})
		_, _ = s.FollowupMessageCreate(i.Interaction, true, &discordgo.WebhookParams{
			Embeds: []*discordgo.MessageEmbed{
				{
					Title:       "Site web interdit",
					Description: "Une erreur est survenue lors de l'ajout du site web interdit.",
					Color:       model.Red.Int(),
					Footer:      model.DefaultFooter,
					Timestamp:   time.Now().Format(time.DateTime),
				},
			},
		})
		return
	}

	_, err = s.FollowupMessageCreate(i.Interaction, true, &discordgo.WebhookParams{
		Embeds: []*discordgo.MessageEmbed{
			{
				Title:       "✅ Site web interdit ajouté",
				Description: fmt.Sprintf("Le site `%s` a été ajouté à la liste des sites web interdits.", websiteURL),
				Color:       model.Green.Int(),
				Footer:      model.DefaultFooter,
				Timestamp:   time.Now().Format(time.DateTime),
			},
		},
	})
	if err != nil {
		d.log.LogError(logger.LogModel{Database: d.db, GuildID: i.GuildID, Function: "addBannedWebsite()",
			Message: fmt.Sprintf("Error sending follow-up message: %s", err),
		})
	}
}

func (d *Discord) removeBannedWebsite(s *discordgo.Session, i *discordgo.InteractionCreate) {
	// Defer the response
	err := s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseDeferredChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Flags: discordgo.MessageFlagsEphemeral,
		},
	})
	if err != nil {
		d.log.LogError(logger.LogModel{Database: d.db, GuildID: i.GuildID, Function: "removeBannedWebsite()",
			Message: fmt.Sprintf("Error deferring response: %s", err),
		})
		return
	}

	d.log.LogInfo(logger.LogModel{Database: d.db, GuildID: i.GuildID, Function: "removeBannedWebsite()",
		Message: fmt.Sprintf("Got command [%s] from user [%s]", i.ApplicationCommandData().Name, i.Member.User.Username),
	})

	if !d.db.CheckModerationPermissionOnInteraction(s, i) {
		return
	}

	options := i.ApplicationCommandData().Options
	websiteID := int64(0)

	// Parse options
	for _, option := range options {
		if option.Name == "id" {
			websiteID = option.IntValue()
		}
	}

	if websiteID == 0 {
		_, _ = s.FollowupMessageCreate(i.Interaction, true, &discordgo.WebhookParams{
			Embeds: []*discordgo.MessageEmbed{
				{
					Title:       "Site web interdit",
					Description: "L'ID du site est requis.",
					Color:       model.Red.Int(),
					Footer:      model.DefaultFooter,
					Timestamp:   time.Now().Format(time.DateTime),
				},
			},
		})
		return
	}

	// Remove from database
	err = d.db.RemoveBannedWebsite(i.GuildID, int(websiteID))
	if err != nil {
		d.log.LogError(logger.LogModel{Database: d.db, GuildID: i.GuildID, Function: "removeBannedWebsite()",
			Message: fmt.Sprintf("Error removing banned website from database: %s", err),
		})
		_, _ = s.FollowupMessageCreate(i.Interaction, true, &discordgo.WebhookParams{
			Embeds: []*discordgo.MessageEmbed{
				{
					Title:       "Site web interdit",
					Description: "Une erreur est survenue lors de la suppression du site web interdit.",
					Color:       model.Red.Int(),
					Footer:      model.DefaultFooter,
					Timestamp:   time.Now().Format(time.DateTime),
				},
			},
		})
		return
	}

	_, err = s.FollowupMessageCreate(i.Interaction, true, &discordgo.WebhookParams{
		Embeds: []*discordgo.MessageEmbed{
			{
				Title:       "✅ Site web interdit supprimé",
				Description: fmt.Sprintf("Le site avec l'ID `%d` a été supprimé de la liste des sites web interdits.", websiteID),
				Color:       model.Green.Int(),
				Footer:      model.DefaultFooter,
				Timestamp:   time.Now().Format(time.DateTime),
			},
		},
	})
	if err != nil {
		d.log.LogError(logger.LogModel{Database: d.db, GuildID: i.GuildID, Function: "removeBannedWebsite()",
			Message: fmt.Sprintf("Error sending follow-up message: %s", err),
		})
	}
}

func (d *Discord) listBannedWebsites(s *discordgo.Session, i *discordgo.InteractionCreate) {
	// Defer the response
	err := s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseDeferredChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Flags: discordgo.MessageFlagsEphemeral,
		},
	})
	if err != nil {
		d.log.LogError(logger.LogModel{Database: d.db, GuildID: i.GuildID, Function: "listBannedWebsites()",
			Message: fmt.Sprintf("Error deferring response: %s", err),
		})
		return
	}

	d.log.LogInfo(logger.LogModel{Database: d.db, GuildID: i.GuildID, Function: "listBannedWebsites()",
		Message: fmt.Sprintf("Got command [%s] from user [%s]", i.ApplicationCommandData().Name, i.Member.User.Username),
	})

	if !d.db.CheckModerationPermissionOnInteraction(s, i) {
		return
	}

	// Get banned websites from database
	bannedWebsites, err := d.db.GetBannedWebsitesByGuildId(i.GuildID)
	if err != nil {
		d.log.LogError(logger.LogModel{Database: d.db, GuildID: i.GuildID, Function: "listBannedWebsites()",
			Message: fmt.Sprintf("Error fetching banned websites from database: %s", err),
		})
		_, _ = s.FollowupMessageCreate(i.Interaction, true, &discordgo.WebhookParams{
			Embeds: []*discordgo.MessageEmbed{
				{
					Title:       "Sites web interdits",
					Description: "Une erreur est survenue lors de la récupération des sites web interdits.",
					Color:       model.Red.Int(),
					Footer:      model.DefaultFooter,
					Timestamp:   time.Now().Format(time.DateTime),
				},
			},
		})
		return
	}

	if len(bannedWebsites) == 0 {
		_, _ = s.FollowupMessageCreate(i.Interaction, true, &discordgo.WebhookParams{
			Embeds: []*discordgo.MessageEmbed{
				{
					Title:       "🌐 Sites web interdits",
					Description: "Aucun site web interdit n'est configuré pour ce serveur.",
					Color:       model.Blue.Int(),
					Footer:      model.DefaultFooter,
					Timestamp:   time.Now().Format(time.DateTime),
				},
			},
		})
		return
	}

	// Build the list
	var websitesList strings.Builder
	for _, website := range bannedWebsites {
		websitesList.WriteString(fmt.Sprintf("• **ID %d**: `%s`\n", website.ID, website.WebsiteURL))
	}

	description := fmt.Sprintf("**Total**: %d site(s) web interdit(s)\n\n%s", len(bannedWebsites), websitesList.String())

	// Truncate if too long
	const maxDescriptionLength = 4096
	if len(description) > maxDescriptionLength {
		description = description[:maxDescriptionLength-3] + "..."
	}

	_, err = s.FollowupMessageCreate(i.Interaction, true, &discordgo.WebhookParams{
		Embeds: []*discordgo.MessageEmbed{
			{
				Title:       "🌐 Liste des sites web interdits",
				Description: description,
				Color:       model.Blue.Int(),
				Footer:      model.DefaultFooter,
				Timestamp:   time.Now().Format(time.DateTime),
			},
		},
	})
	if err != nil {
		d.log.LogError(logger.LogModel{Database: d.db, GuildID: i.GuildID, Function: "listBannedWebsites()",
			Message: fmt.Sprintf("Error sending follow-up message: %s", err),
		})
	}
}

func (d *Discord) configureAutomod(s *discordgo.Session, i *discordgo.InteractionCreate) {
	// Defer the response
	err := s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseDeferredChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Flags: discordgo.MessageFlagsEphemeral,
		},
	})
	if err != nil {
		d.log.LogError(logger.LogModel{Database: d.db, GuildID: i.GuildID, Function: "configureAutomod()",
			Message: fmt.Sprintf("Error deferring response: %s", err),
		})
		return
	}

	d.log.LogInfo(logger.LogModel{Database: d.db, GuildID: i.GuildID, Function: "configureAutomod()",
		Message: fmt.Sprintf("Got command [%s] from user [%s]", i.ApplicationCommandData().Name, i.Member.User.Username),
	})

	if !d.db.CheckModerationPermissionOnInteraction(s, i) {
		return
	}

	// Get automoderation features
	items, err := d.getAutomoderationFeaturesAsItems(s, i.GuildID)
	if err != nil {
		d.sendErrorMessage(s, i.Interaction, "Configuration Automodération", "Une erreur est survenue lors de la récupération des fonctionnalités.")
		return
	}

	// Start with page 0
	config, dbOps := d.getAutomodSettingsConfig()
	d.sendSelectPage(s, i.Interaction, items, 0, config, dbOps)
}
