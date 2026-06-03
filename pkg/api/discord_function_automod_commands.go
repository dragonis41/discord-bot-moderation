package api

import (
	"fmt"
	"strings"

	"github.com/bwmarrin/discordgo"
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

// maxEmbedDescriptionLength is Discord's hard limit on an embed description.
const maxEmbedDescriptionLength = 4096

func (d *Discord) addBannedWord(s *discordgo.Session, i *discordgo.InteractionCreate) {
	if !d.beginModCommand(s, i, "addBannedWord()") {
		return
	}

	wordPattern := ""
	isRegex := false
	for _, option := range i.ApplicationCommandData().Options {
		switch option.Name {
		case "word":
			wordPattern = option.StringValue()
		case "is_regex":
			isRegex = option.BoolValue()
		}
	}

	if wordPattern == "" {
		d.followup(s, i, "addBannedWord()", errorEmbed("Mot interdit", "Le mot ne peut pas être vide."))
		return
	}

	if err := d.db.AddBannedWord(i.GuildID, wordPattern, isRegex); err != nil {
		d.logError(i.GuildID, "addBannedWord()", "Error adding banned word to database: %s", err)
		d.followup(s, i, "addBannedWord()", errorEmbed("Mot interdit", "Une erreur est survenue lors de l'ajout du mot interdit."))
		return
	}

	wordType := "littéral"
	if isRegex {
		wordType = "regex"
	}

	d.followup(s, i, "addBannedWord()", successEmbed("✅ Mot interdit ajouté",
		fmt.Sprintf("Le mot `%s` (type: %s) a été ajouté à la liste des mots interdits.", wordPattern, wordType)))
}

func (d *Discord) removeBannedWord(s *discordgo.Session, i *discordgo.InteractionCreate) {
	if !d.beginModCommand(s, i, "removeBannedWord()") {
		return
	}

	wordID := int64(0)
	for _, option := range i.ApplicationCommandData().Options {
		if option.Name == "id" {
			wordID = option.IntValue()
		}
	}

	if wordID == 0 {
		d.followup(s, i, "removeBannedWord()", errorEmbed("Mot interdit", "L'ID du mot est requis."))
		return
	}

	if err := d.db.RemoveBannedWord(i.GuildID, int(wordID)); err != nil {
		d.logError(i.GuildID, "removeBannedWord()", "Error removing banned word from database: %s", err)
		d.followup(s, i, "removeBannedWord()", errorEmbed("Mot interdit", "Une erreur est survenue lors de la suppression du mot interdit."))
		return
	}

	d.followup(s, i, "removeBannedWord()", successEmbed("✅ Mot interdit supprimé",
		fmt.Sprintf("Le mot avec l'ID `%d` a été supprimé de la liste des mots interdits.", wordID)))
}

func (d *Discord) listBannedWords(s *discordgo.Session, i *discordgo.InteractionCreate) {
	if !d.beginModCommand(s, i, "listBannedWords()") {
		return
	}

	bannedWords, err := d.db.GetBannedWordsByGuildId(i.GuildID)
	if err != nil {
		d.logError(i.GuildID, "listBannedWords()", "Error fetching banned words from database: %s", err)
		d.followup(s, i, "listBannedWords()", errorEmbed("Mots interdits", "Une erreur est survenue lors de la récupération des mots interdits."))
		return
	}

	if len(bannedWords) == 0 {
		d.followup(s, i, "listBannedWords()", infoEmbed("📝 Mots interdits", "Aucun mot interdit n'est configuré pour ce serveur."))
		return
	}

	var wordsList strings.Builder
	for _, word := range bannedWords {
		wordType := "littéral"
		if word.IsRegex {
			wordType = "regex"
		}
		wordsList.WriteString(fmt.Sprintf("• **ID %d**: `%s` (type: %s)\n", word.ID, word.WordPattern, wordType))
	}

	description := fmt.Sprintf("**Total**: %d mot(s) interdit(s)\n\n%s", len(bannedWords), wordsList.String())
	d.followup(s, i, "listBannedWords()", infoEmbed("📝 Liste des mots interdits", truncate(description, maxEmbedDescriptionLength)))
}

func (d *Discord) addBannedWebsite(s *discordgo.Session, i *discordgo.InteractionCreate) {
	if !d.beginModCommand(s, i, "addBannedWebsite()") {
		return
	}

	websiteURL := ""
	for _, option := range i.ApplicationCommandData().Options {
		if option.Name == "url" {
			websiteURL = option.StringValue()
		}
	}

	if websiteURL == "" {
		d.followup(s, i, "addBannedWebsite()", errorEmbed("Site web interdit", "L'URL ne peut pas être vide."))
		return
	}

	if err := d.db.AddBannedWebsite(i.GuildID, websiteURL); err != nil {
		d.logError(i.GuildID, "addBannedWebsite()", "Error adding banned website to database: %s", err)
		d.followup(s, i, "addBannedWebsite()", errorEmbed("Site web interdit", "Une erreur est survenue lors de l'ajout du site web interdit."))
		return
	}

	d.followup(s, i, "addBannedWebsite()", successEmbed("✅ Site web interdit ajouté",
		fmt.Sprintf("Le site `%s` a été ajouté à la liste des sites web interdits.", websiteURL)))
}

func (d *Discord) removeBannedWebsite(s *discordgo.Session, i *discordgo.InteractionCreate) {
	if !d.beginModCommand(s, i, "removeBannedWebsite()") {
		return
	}

	websiteID := int64(0)
	for _, option := range i.ApplicationCommandData().Options {
		if option.Name == "id" {
			websiteID = option.IntValue()
		}
	}

	if websiteID == 0 {
		d.followup(s, i, "removeBannedWebsite()", errorEmbed("Site web interdit", "L'ID du site est requis."))
		return
	}

	if err := d.db.RemoveBannedWebsite(i.GuildID, int(websiteID)); err != nil {
		d.logError(i.GuildID, "removeBannedWebsite()", "Error removing banned website from database: %s", err)
		d.followup(s, i, "removeBannedWebsite()", errorEmbed("Site web interdit", "Une erreur est survenue lors de la suppression du site web interdit."))
		return
	}

	d.followup(s, i, "removeBannedWebsite()", successEmbed("✅ Site web interdit supprimé",
		fmt.Sprintf("Le site avec l'ID `%d` a été supprimé de la liste des sites web interdits.", websiteID)))
}

func (d *Discord) listBannedWebsites(s *discordgo.Session, i *discordgo.InteractionCreate) {
	if !d.beginModCommand(s, i, "listBannedWebsites()") {
		return
	}

	bannedWebsites, err := d.db.GetBannedWebsitesByGuildId(i.GuildID)
	if err != nil {
		d.logError(i.GuildID, "listBannedWebsites()", "Error fetching banned websites from database: %s", err)
		d.followup(s, i, "listBannedWebsites()", errorEmbed("Sites web interdits", "Une erreur est survenue lors de la récupération des sites web interdits."))
		return
	}

	if len(bannedWebsites) == 0 {
		d.followup(s, i, "listBannedWebsites()", infoEmbed("🌐 Sites web interdits", "Aucun site web interdit n'est configuré pour ce serveur."))
		return
	}

	var websitesList strings.Builder
	for _, website := range bannedWebsites {
		websitesList.WriteString(fmt.Sprintf("• **ID %d**: `%s`\n", website.ID, website.WebsiteURL))
	}

	description := fmt.Sprintf("**Total**: %d site(s) web interdit(s)\n\n%s", len(bannedWebsites), websitesList.String())
	d.followup(s, i, "listBannedWebsites()", infoEmbed("🌐 Liste des sites web interdits", truncate(description, maxEmbedDescriptionLength)))
}

func (d *Discord) configureAutomod(s *discordgo.Session, i *discordgo.InteractionCreate) {
	if !d.beginModCommand(s, i, "configureAutomod()") {
		return
	}

	items, err := d.getAutomoderationFeaturesAsItems(s, i.GuildID)
	if err != nil {
		d.sendErrorMessage(s, i.Interaction, "Configuration Automodération", "Une erreur est survenue lors de la récupération des fonctionnalités.")
		return
	}

	config, dbOps := d.getAutomodSettingsConfig()
	d.sendSelectPage(s, i.Interaction, items, 0, config, dbOps)
}
