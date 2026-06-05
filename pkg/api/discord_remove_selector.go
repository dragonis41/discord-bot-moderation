package api

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/bwmarrin/discordgo"
	"github.com/dragonis41/discord-bot-moderation/pkg/i18n"
	"github.com/dragonis41/discord-bot-moderation/pkg/model"
)

// removeItemsPerPage caps how many entries appear in a single removal dropdown.
// Discord allows at most 25 options per select menu.
const removeItemsPerPage = 25

// Discord's per-option limits for select menus.
const (
	maxSelectLabelLength       = 100
	maxSelectDescriptionLength = 100
)

// removeOption is one selectable entry in a removal dropdown.
type removeOption struct {
	ID    int    // database id, used as the select option value
	Label string // shown as the option label and echoed in the confirmation
	Desc  string // optional secondary text under the label
}

// removeSelector describes a paginated "pick an entry to delete" menu. The two
// automod removal commands (banned words, banned websites) differ only by these
// fields, so the build/handle logic below is shared. The list and remove
// closures wrap the matching database calls.
type removeSelector struct {
	prefix      string // component id namespace, e.g. "rmword" / "rmsite"
	title       string // embed title
	intro       string // embed description shown above the counter
	placeholder string // select menu placeholder
	emptyMsg    string // shown when the guild has nothing to remove
	errorTitle  string // title for error embeds
	errorMsg    string // body for the "fetch failed" error embed
	noun        string // pluralizable noun used in the confirmation ("word(s)", "website(s)")

	lang i18n.Lang // guild language, used for the dynamic counters/notices below

	list   func(guildID string) ([]removeOption, error)
	remove func(guildID string, id int) (bool, error)
}

func (d *Discord) bannedWordRemoveSelector(lang i18n.Lang) removeSelector {
	return removeSelector{
		prefix:      "rmword",
		title:       i18n.T(lang, "remove.word.title"),
		intro:       i18n.T(lang, "remove.word.intro"),
		placeholder: i18n.T(lang, "remove.word.placeholder"),
		emptyMsg:    i18n.T(lang, "remove.word.empty"),
		errorTitle:  i18n.T(lang, "remove.word.error_title"),
		errorMsg:    i18n.T(lang, "remove.word.error"),
		noun:        i18n.T(lang, "remove.word.noun"),
		lang:        lang,
		list: func(guildID string) ([]removeOption, error) {
			words, err := d.db.GetBannedWordsByGuildId(guildID)
			if err != nil {
				return nil, err
			}
			opts := make([]removeOption, len(words))
			for i, w := range words {
				wordType := i18n.T(lang, "bannedword.type_literal")
				if w.IsRegex {
					wordType = i18n.T(lang, "bannedword.type_regex")
				}
				opts[i] = removeOption{
					ID:    w.ID,
					Label: w.WordPattern,
					Desc:  i18n.T(lang, "remove.type", wordType),
				}
			}
			return opts, nil
		},
		remove: d.db.RemoveBannedWord,
	}
}

func (d *Discord) bannedWebsiteRemoveSelector(lang i18n.Lang) removeSelector {
	return removeSelector{
		prefix:      "rmsite",
		title:       i18n.T(lang, "remove.site.title"),
		intro:       i18n.T(lang, "remove.site.intro"),
		placeholder: i18n.T(lang, "remove.site.placeholder"),
		emptyMsg:    i18n.T(lang, "remove.site.empty"),
		errorTitle:  i18n.T(lang, "remove.site.error_title"),
		errorMsg:    i18n.T(lang, "remove.site.error"),
		noun:        i18n.T(lang, "remove.site.noun"),
		lang:        lang,
		list: func(guildID string) ([]removeOption, error) {
			sites, err := d.db.GetBannedWebsitesByGuildId(guildID)
			if err != nil {
				return nil, err
			}
			opts := make([]removeOption, len(sites))
			for i, s := range sites {
				opts[i] = removeOption{ID: s.ID, Label: s.WebsiteURL}
			}
			return opts, nil
		},
		remove: d.db.RemoveBannedWebsite,
	}
}

// startRemoveSelection fetches the guild's entries and posts the first page of
// the removal dropdown, or an informational message when there is nothing to
// remove. It is called from the slash command after the response is deferred.
func (d *Discord) startRemoveSelection(s discordClient, i *discordgo.InteractionCreate, sel removeSelector, function string) {
	opts, err := sel.list(i.GuildID)
	if err != nil {
		d.logError(i.GuildID, function, "Error fetching items: %s", err)
		d.followup(s, i, function, errorEmbed(sel.lang, sel.errorTitle, sel.errorMsg))
		return
	}

	if len(opts) == 0 {
		d.followup(s, i, function, infoEmbed(sel.lang, sel.title, sel.emptyMsg))
		return
	}

	embed, components := buildRemoveMessage(sel, opts, 0, "")
	if _, err := s.FollowupMessageCreate(i.Interaction, true, &discordgo.WebhookParams{
		Flags:      discordgo.MessageFlagsEphemeral,
		Embeds:     []*discordgo.MessageEmbed{embed},
		Components: components,
	}); err != nil {
		d.logError(i.GuildID, function, "Error sending removal menu: %s", err)
	}
}

// buildRemoveMessage renders one page of a removal dropdown: an embed with a
// running counter (plus an optional notice such as a deletion confirmation), a
// multi-select menu of the page's entries, and prev/next buttons when needed.
func buildRemoveMessage(sel removeSelector, opts []removeOption, page int, notice string) (*discordgo.MessageEmbed, []discordgo.MessageComponent) {
	totalPages := max((len(opts)+removeItemsPerPage-1)/removeItemsPerPage, 1)
	page = max(0, min(page, totalPages-1))

	start := page * removeItemsPerPage
	end := min(start+removeItemsPerPage, len(opts))

	menuOptions := make([]discordgo.SelectMenuOption, 0, end-start)
	for _, o := range opts[start:end] {
		menuOptions = append(menuOptions, discordgo.SelectMenuOption{
			Label:       truncate(o.Label, maxSelectLabelLength),
			Value:       strconv.Itoa(o.ID),
			Description: truncate(o.Desc, maxSelectDescriptionLength),
		})
	}

	minValues := 1
	components := []discordgo.MessageComponent{
		discordgo.ActionsRow{Components: []discordgo.MessageComponent{
			discordgo.SelectMenu{
				CustomID:    fmt.Sprintf("%s_select_%d", sel.prefix, page),
				Placeholder: sel.placeholder,
				MinValues:   &minValues,
				MaxValues:   len(menuOptions),
				Options:     menuOptions,
			},
		}},
	}

	if buttons := removeNavigationButtons(sel.lang, sel.prefix, page, totalPages); len(buttons) > 0 {
		components = append(components, discordgo.ActionsRow{Components: buttons})
	}

	description := sel.intro + "\n\n" + i18n.T(sel.lang, "remove.total", len(opts), page+1, totalPages)
	if notice != "" {
		description = notice + "\n\n" + description
	}

	return &discordgo.MessageEmbed{
		Title:       sel.title,
		Description: truncate(description, maxEmbedDescriptionLength),
		Color:       model.Blue.Int(),
		Footer:      hintFooter(sel.lang),
		Timestamp:   time.Now().UTC().Format(time.RFC3339),
	}, components
}

// removeNavigationButtons returns prev/next buttons (with a disabled page
// indicator) when the list spans more than one page, and nothing otherwise. The
// indicator's custom id deliberately avoids the "_select"/"_page_" namespaces so
// the handler never tries to route it (it is disabled and never fires anyway).
func removeNavigationButtons(lang i18n.Lang, prefix string, page, totalPages int) []discordgo.MessageComponent {
	if totalPages <= 1 {
		return nil
	}
	return []discordgo.MessageComponent{
		discordgo.Button{
			Label:    i18n.T(lang, "button.prev"),
			Style:    discordgo.PrimaryButton,
			CustomID: fmt.Sprintf("%s_page_prev_%d", prefix, page),
			Disabled: page == 0,
		},
		discordgo.Button{
			Label:    i18n.T(lang, "button.page", page+1, totalPages),
			Style:    discordgo.SecondaryButton,
			CustomID: prefix + "_indicator",
			Disabled: true,
		},
		discordgo.Button{
			Label:    i18n.T(lang, "button.next"),
			Style:    discordgo.PrimaryButton,
			CustomID: fmt.Sprintf("%s_page_next_%d", prefix, page),
			Disabled: page == totalPages-1,
		},
	}
}

// handleRemoveBannedWordSelection and handleRemoveBannedWebsiteSelection are the
// discordgo component callbacks (registered in RunDiscordBot). They delegate to
// the shared handler with the matching selector.
func (d *Discord) handleRemoveBannedWordSelection(s *discordgo.Session, i *discordgo.InteractionCreate) {
	d.handleRemoveSelection(s, i, d.bannedWordRemoveSelector(d.lang(i.GuildID)), "handleRemoveBannedWordSelection()")
}

func (d *Discord) handleRemoveBannedWebsiteSelection(s *discordgo.Session, i *discordgo.InteractionCreate) {
	d.handleRemoveSelection(s, i, d.bannedWebsiteRemoveSelector(d.lang(i.GuildID)), "handleRemoveBannedWebsiteSelection()")
}

// handleRemoveSelection routes a component interaction belonging to this
// selector to either a deletion (select menu) or a page change (nav buttons).
// Interactions for other components are ignored so the other handlers can claim
// them.
func (d *Discord) handleRemoveSelection(s discordClient, i *discordgo.InteractionCreate, sel removeSelector, function string) {
	if i.Type != discordgo.InteractionMessageComponent {
		return
	}
	customID := i.MessageComponentData().CustomID
	if !strings.HasPrefix(customID, sel.prefix+"_") {
		return
	}

	switch {
	case strings.HasPrefix(customID, sel.prefix+"_select_"):
		d.handleRemoveDelete(s, i, sel, customID, function)
	case strings.HasPrefix(customID, sel.prefix+"_page_prev_"), strings.HasPrefix(customID, sel.prefix+"_page_next_"):
		d.handleRemovePage(s, i, sel, customID, function)
	}
}

// handleRemoveDelete deletes the selected entries, then re-renders the menu with
// a confirmation notice (or the empty-list message once nothing remains).
func (d *Discord) handleRemoveDelete(s discordClient, i *discordgo.InteractionCreate, sel removeSelector, customID, function string) {
	var page int
	_, _ = fmt.Sscanf(customID, sel.prefix+"_select_%d", &page)

	// Snapshot id -> label before deleting so the confirmation can name what was
	// removed (the entries are gone by the time we re-list).
	opts, err := sel.list(i.GuildID)
	if err != nil {
		d.logError(i.GuildID, function, "Error fetching items: %s", err)
		d.respondRemoveError(s, i, sel)
		return
	}
	labels := make(map[int]string, len(opts))
	for _, o := range opts {
		labels[o.ID] = o.Label
	}

	var removed []string
	for _, v := range i.MessageComponentData().Values {
		id, convErr := strconv.Atoi(v)
		if convErr != nil {
			continue
		}
		ok, remErr := sel.remove(i.GuildID, id)
		if remErr != nil {
			d.logError(i.GuildID, function, "Error removing item [%d]: %s", id, remErr)
			continue
		}
		if ok {
			removed = append(removed, labels[id])
		}
	}

	// Re-fetch so the refreshed menu reflects the deletions.
	opts, err = sel.list(i.GuildID)
	if err != nil {
		d.logError(i.GuildID, function, "Error refreshing items: %s", err)
		opts = nil
	}

	d.respondRemoveUpdate(s, i, sel, opts, page, removeNotice(sel, removed), function)

	if len(removed) > 0 {
		d.logSuccess(i.GuildID, function, "User [%s] removed %d %s entry/entries for guild [%s]", i.Member.User.Username, len(removed), sel.noun, i.GuildID)
	}
}

// removeNotice builds the confirmation line shown after a deletion attempt.
func removeNotice(sel removeSelector, removed []string) string {
	if len(removed) == 0 {
		return i18n.T(sel.lang, "remove.notice_none")
	}
	quoted := make([]string, len(removed))
	for i, r := range removed {
		quoted[i] = fmt.Sprintf("`%s`", sanitizeInlineCode(r))
	}
	return i18n.T(sel.lang, "remove.notice", len(removed), sel.noun, strings.Join(quoted, ", "))
}

// handleRemovePage re-renders the menu at the previous or next page.
func (d *Discord) handleRemovePage(s discordClient, i *discordgo.InteractionCreate, sel removeSelector, customID, function string) {
	var page int
	if strings.HasPrefix(customID, sel.prefix+"_page_prev_") {
		_, _ = fmt.Sscanf(customID, sel.prefix+"_page_prev_%d", &page)
		page--
	} else {
		_, _ = fmt.Sscanf(customID, sel.prefix+"_page_next_%d", &page)
		page++
	}

	opts, err := sel.list(i.GuildID)
	if err != nil {
		d.logError(i.GuildID, function, "Error fetching items: %s", err)
		d.respondRemoveError(s, i, sel)
		return
	}
	d.respondRemoveUpdate(s, i, sel, opts, page, "", function)
}

// respondRemoveUpdate replaces the menu message in place with the given page (or
// the empty-list message when nothing remains).
func (d *Discord) respondRemoveUpdate(s discordClient, i *discordgo.InteractionCreate, sel removeSelector, opts []removeOption, page int, notice, function string) {
	var embed *discordgo.MessageEmbed
	components := []discordgo.MessageComponent{}

	if len(opts) == 0 {
		desc := sel.emptyMsg
		if notice != "" {
			desc = notice + "\n\n" + desc
		}
		embed = &discordgo.MessageEmbed{
			Title:       sel.title,
			Description: desc,
			Color:       model.Blue.Int(),
			Footer:      hintFooter(sel.lang),
			Timestamp:   time.Now().UTC().Format(time.RFC3339),
		}
	} else {
		embed, components = buildRemoveMessage(sel, opts, page, notice)
	}

	if err := s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseUpdateMessage,
		Data: &discordgo.InteractionResponseData{
			Embeds:     []*discordgo.MessageEmbed{embed},
			Components: components,
		},
	}); err != nil {
		d.logError(i.GuildID, function, "Error updating removal menu: %s", err)
	}
}

// respondRemoveError acknowledges the interaction with an ephemeral error embed,
// leaving the existing menu untouched.
func (d *Discord) respondRemoveError(s discordClient, i *discordgo.InteractionCreate, sel removeSelector) {
	if err := s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Flags:  discordgo.MessageFlagsEphemeral,
			Embeds: []*discordgo.MessageEmbed{errorEmbed(sel.lang, sel.errorTitle, sel.errorMsg)},
		},
	}); err != nil {
		d.logError(i.GuildID, "respondRemoveError()", "Error responding to interaction: %s", err)
	}
}
