package api

import (
	"time"

	"github.com/bwmarrin/discordgo"
	"github.com/dragonis41/discord-bot-moderation/pkg/i18n"
	"github.com/dragonis41/discord-bot-moderation/pkg/model"
)

// langSelectCustomID namespaces the language select menu component.
const langSelectCustomID = "set_lang"

// selectLanguage handles the /lang command: it shows a select menu of the
// supported languages so a moderator can set the server's language. The menu is
// rendered in the server's current language and marks that language as selected.
func (d *Discord) selectLanguage(s discordClient, i *discordgo.InteractionCreate) {
	if !d.beginModCommand(s, i, "selectLanguage()") {
		return
	}

	lang := d.lang(i.GuildID)

	if _, err := s.FollowupMessageCreate(i.Interaction, true, &discordgo.WebhookParams{
		Flags:      discordgo.MessageFlagsEphemeral,
		Embeds:     []*discordgo.MessageEmbed{langMenuEmbed(lang)},
		Components: []discordgo.MessageComponent{langSelectMenu(lang)},
	}); err != nil {
		d.logError(i.GuildID, "selectLanguage()", "Error sending language menu: %s", err)
	}
}

// handleLanguageSelection is the discordgo component callback for the /lang
// select menu (registered in RunDiscordBot). It keeps the concrete
// *discordgo.Session signature discordgo's reflection requires and delegates to
// applyLanguageSelection, which takes the testable discordClient interface.
func (d *Discord) handleLanguageSelection(s *discordgo.Session, i *discordgo.InteractionCreate) {
	d.applyLanguageSelection(s, i)
}

// applyLanguageSelection persists the chosen language and replaces the menu with
// a confirmation written in the newly selected language.
func (d *Discord) applyLanguageSelection(s discordClient, i *discordgo.InteractionCreate) {
	if i.Type != discordgo.InteractionMessageComponent {
		return
	}
	data := i.MessageComponentData()
	if data.CustomID != langSelectCustomID {
		return
	}

	if len(data.Values) == 0 {
		return
	}
	newLang := i18n.Parse(data.Values[0])

	if err := d.db.SetGuildLanguage(i.GuildID, string(newLang)); err != nil {
		d.logError(i.GuildID, "handleLanguageSelection()", "Error saving guild language: %s", err)
		if err := s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseUpdateMessage,
			Data: &discordgo.InteractionResponseData{
				Embeds:     []*discordgo.MessageEmbed{errorEmbed(d.lang(i.GuildID), i18n.T(newLang, "lang.menu_title"), i18n.T(newLang, "lang.error"))},
				Components: []discordgo.MessageComponent{},
			},
		}); err != nil {
			d.logError(i.GuildID, "handleLanguageSelection()", "Error responding to interaction: %s", err)
		}
		return
	}

	d.logSuccess(i.GuildID, "handleLanguageSelection()", "User [%s] set guild [%s] language to [%s]", i.Member.User.Username, i.GuildID, newLang)

	if err := s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseUpdateMessage,
		Data: &discordgo.InteractionResponseData{
			Embeds: []*discordgo.MessageEmbed{{
				Title:       i18n.T(newLang, "lang.menu_title"),
				Description: i18n.T(newLang, "lang.updated", i18n.Name(newLang)),
				Color:       model.Green.Int(),
				Footer:      hintFooter(newLang),
				Timestamp:   time.Now().UTC().Format(time.RFC3339),
			}},
			Components: []discordgo.MessageComponent{},
		},
	}); err != nil {
		d.logError(i.GuildID, "handleLanguageSelection()", "Error responding to interaction: %s", err)
	}
}

// langMenuEmbed builds the prompt embed shown by /lang, in the given language.
func langMenuEmbed(lang i18n.Lang) *discordgo.MessageEmbed {
	return &discordgo.MessageEmbed{
		Title:       i18n.T(lang, "lang.menu_title"),
		Description: i18n.T(lang, "lang.menu_description"),
		Color:       model.Blue.Int(),
		Footer:      hintFooter(lang),
		Timestamp:   time.Now().UTC().Format(time.RFC3339),
	}
}

// langSelectMenu builds the language select menu, marking current as selected.
func langSelectMenu(current i18n.Lang) discordgo.ActionsRow {
	options := make([]discordgo.SelectMenuOption, 0, len(i18n.Supported))
	for _, l := range i18n.Supported {
		options = append(options, discordgo.SelectMenuOption{
			Label:   i18n.Name(l),
			Value:   string(l),
			Emoji:   &discordgo.ComponentEmoji{Name: i18n.Flag(l)},
			Default: l == current,
		})
	}

	return discordgo.ActionsRow{
		Components: []discordgo.MessageComponent{
			discordgo.SelectMenu{
				CustomID:    langSelectCustomID,
				Placeholder: i18n.T(current, "lang.placeholder"),
				Options:     options,
			},
		},
	}
}
