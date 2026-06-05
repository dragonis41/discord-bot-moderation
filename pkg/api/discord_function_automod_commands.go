package api

import (
	"strings"

	"github.com/bwmarrin/discordgo"
	"github.com/dragonis41/discord-bot-moderation/pkg/i18n"
)

type DiscordAutomodCommandsInterface interface {
	addBannedWord(s discordClient, i *discordgo.InteractionCreate)
	removeBannedWord(s discordClient, i *discordgo.InteractionCreate)
	listBannedWords(s discordClient, i *discordgo.InteractionCreate)
	addBannedWebsite(s discordClient, i *discordgo.InteractionCreate)
	removeBannedWebsite(s discordClient, i *discordgo.InteractionCreate)
	listBannedWebsites(s discordClient, i *discordgo.InteractionCreate)
	configureAutomod(s discordClient, i *discordgo.InteractionCreate)
}

// maxEmbedDescriptionLength is Discord's hard limit on an embed description.
const maxEmbedDescriptionLength = 4096

func (d *Discord) addBannedWord(s discordClient, i *discordgo.InteractionCreate) {
	if !d.beginModCommand(s, i, "addBannedWord()") {
		return
	}

	lang := d.lang(i.GuildID)

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
		d.followup(s, i, "addBannedWord()", errorEmbed(lang, i18n.T(lang, "bannedword.title"), i18n.T(lang, "bannedword.empty_word")))
		return
	}

	if err := d.db.AddBannedWord(i.GuildID, wordPattern, isRegex); err != nil {
		d.logError(i.GuildID, "addBannedWord()", "Error adding banned word to database: %s", err)
		d.followup(s, i, "addBannedWord()", errorEmbed(lang, i18n.T(lang, "bannedword.title"), i18n.T(lang, "bannedword.add_error")))
		return
	}

	wordType := i18n.T(lang, "bannedword.type_literal")
	if isRegex {
		wordType = i18n.T(lang, "bannedword.type_regex")
	}

	d.followup(s, i, "addBannedWord()", successEmbed(lang, i18n.T(lang, "bannedword.added_title"),
		i18n.T(lang, "bannedword.added", wordPattern, wordType)))
}

// removeBannedWord opens a paginated dropdown of the guild's banned words so the
// moderator picks the entries to delete, instead of typing a raw id (which let
// users target another guild's ids and silently no-op). See the removal-selector
// framework in discord_remove_selector.go.
func (d *Discord) removeBannedWord(s discordClient, i *discordgo.InteractionCreate) {
	if !d.beginModCommand(s, i, "removeBannedWord()") {
		return
	}

	d.startRemoveSelection(s, i, d.bannedWordRemoveSelector(d.lang(i.GuildID)), "removeBannedWord()")
}

func (d *Discord) listBannedWords(s discordClient, i *discordgo.InteractionCreate) {
	if !d.beginModCommand(s, i, "listBannedWords()") {
		return
	}

	lang := d.lang(i.GuildID)

	bannedWords, err := d.db.GetBannedWordsByGuildId(i.GuildID)
	if err != nil {
		d.logError(i.GuildID, "listBannedWords()", "Error fetching banned words from database: %s", err)
		d.followup(s, i, "listBannedWords()", errorEmbed(lang, i18n.T(lang, "bannedword.list_error_title"), i18n.T(lang, "bannedword.list_error")))
		return
	}

	if len(bannedWords) == 0 {
		d.followup(s, i, "listBannedWords()", infoEmbed(lang, i18n.T(lang, "bannedword.list_empty_title"), i18n.T(lang, "bannedword.list_empty")))
		return
	}

	var wordsList strings.Builder
	for _, word := range bannedWords {
		wordType := i18n.T(lang, "bannedword.type_literal")
		if word.IsRegex {
			wordType = i18n.T(lang, "bannedword.type_regex")
		}
		wordsList.WriteString(i18n.T(lang, "bannedword.list_item", word.ID, word.WordPattern, wordType))
	}

	description := i18n.T(lang, "bannedword.list_total", len(bannedWords), wordsList.String())
	d.followup(s, i, "listBannedWords()", infoEmbed(lang, i18n.T(lang, "bannedword.list_title"), truncate(description, maxEmbedDescriptionLength)))
}

func (d *Discord) addBannedWebsite(s discordClient, i *discordgo.InteractionCreate) {
	if !d.beginModCommand(s, i, "addBannedWebsite()") {
		return
	}

	lang := d.lang(i.GuildID)

	websiteURL := ""
	for _, option := range i.ApplicationCommandData().Options {
		if option.Name == "url" {
			websiteURL = option.StringValue()
		}
	}

	if websiteURL == "" {
		d.followup(s, i, "addBannedWebsite()", errorEmbed(lang, i18n.T(lang, "bannedsite.title"), i18n.T(lang, "bannedsite.empty_url")))
		return
	}

	if err := d.db.AddBannedWebsite(i.GuildID, websiteURL); err != nil {
		d.logError(i.GuildID, "addBannedWebsite()", "Error adding banned website to database: %s", err)
		d.followup(s, i, "addBannedWebsite()", errorEmbed(lang, i18n.T(lang, "bannedsite.title"), i18n.T(lang, "bannedsite.add_error")))
		return
	}

	d.followup(s, i, "addBannedWebsite()", successEmbed(lang, i18n.T(lang, "bannedsite.added_title"),
		i18n.T(lang, "bannedsite.added", websiteURL)))
}

// removeBannedWebsite opens a paginated dropdown of the guild's banned websites
// so the moderator picks the entries to delete, mirroring removeBannedWord.
func (d *Discord) removeBannedWebsite(s discordClient, i *discordgo.InteractionCreate) {
	if !d.beginModCommand(s, i, "removeBannedWebsite()") {
		return
	}

	d.startRemoveSelection(s, i, d.bannedWebsiteRemoveSelector(d.lang(i.GuildID)), "removeBannedWebsite()")
}

func (d *Discord) listBannedWebsites(s discordClient, i *discordgo.InteractionCreate) {
	if !d.beginModCommand(s, i, "listBannedWebsites()") {
		return
	}

	lang := d.lang(i.GuildID)

	bannedWebsites, err := d.db.GetBannedWebsitesByGuildId(i.GuildID)
	if err != nil {
		d.logError(i.GuildID, "listBannedWebsites()", "Error fetching banned websites from database: %s", err)
		d.followup(s, i, "listBannedWebsites()", errorEmbed(lang, i18n.T(lang, "bannedsite.list_error_title"), i18n.T(lang, "bannedsite.list_error")))
		return
	}

	if len(bannedWebsites) == 0 {
		d.followup(s, i, "listBannedWebsites()", infoEmbed(lang, i18n.T(lang, "bannedsite.list_empty_title"), i18n.T(lang, "bannedsite.list_empty")))
		return
	}

	var websitesList strings.Builder
	for _, website := range bannedWebsites {
		websitesList.WriteString(i18n.T(lang, "bannedsite.list_item", website.ID, website.WebsiteURL))
	}

	description := i18n.T(lang, "bannedsite.list_total", len(bannedWebsites), websitesList.String())
	d.followup(s, i, "listBannedWebsites()", infoEmbed(lang, i18n.T(lang, "bannedsite.list_title"), truncate(description, maxEmbedDescriptionLength)))
}

func (d *Discord) configureAutomod(s discordClient, i *discordgo.InteractionCreate) {
	if !d.beginModCommand(s, i, "configureAutomod()") {
		return
	}

	lang := d.lang(i.GuildID)

	items, err := d.getAutomoderationFeaturesAsItems(s, i.GuildID)
	if err != nil {
		d.sendErrorMessage(s, i.Interaction, lang, i18n.T(lang, "automodcfg.error_title"), i18n.T(lang, "automodcfg.error"))
		return
	}

	config, dbOps := d.getAutomodSettingsConfig(lang)
	d.sendSelectPage(s, i.Interaction, items, 0, config, dbOps)
}
