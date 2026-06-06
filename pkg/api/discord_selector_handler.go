package api

import (
	"fmt"
	"sort"
	"strings"

	"github.com/bwmarrin/discordgo"
	"github.com/dragonis41/discord-bot-moderation/pkg/i18n"
)

const channelsPerPage = 25
const rolesPerPage = 25

// channelSelectionDoneMessage returns a "done" formatter that lists the selected
// channels (sorted by position) under the given header. The three channel
// selectors (log, moderation, excluded) differ only by that header.
func channelSelectionDoneMessage(header string) func([]SelectionItem) string {
	return func(items []SelectionItem) string {
		sort.Slice(items, func(i, j int) bool {
			return items[i].(ChannelItem).Position < items[j].(ChannelItem).Position
		})

		var channelNames []string
		for _, item := range items {
			channelNames = append(channelNames, fmt.Sprintf("- <#%s>", item.GetID()))
		}

		return header + "\n" + strings.Join(channelNames, "\n")
	}
}

func (d *Discord) getLogChannelConfig(lang i18n.Lang) (SelectionConfig, DatabaseOperations) {
	config := SelectionConfig{
		Prefix:          "log_channel",
		ItemsPerPage:    channelsPerPage,
		Title:           i18n.T(lang, "selector.log.title"),
		Description:     i18n.T(lang, "selector.log.description"),
		Placeholder:     i18n.T(lang, "selector.log.placeholder"),
		DoneDescription: i18n.T(lang, "selector.no_channel"),
		EmojiName:       "💬",
	}
	return config, &LogChannelDB{db: d.db}
}

func (d *Discord) handleLogChannelSelection(s *discordgo.Session, i *discordgo.InteractionCreate) {
	lang := d.lang(i.GuildID)
	config, dbOps := d.getLogChannelConfig(lang)
	d.handleSelection(s, i, config, dbOps, d.getTextChannelsAsItems, channelSelectionDoneMessage(i18n.T(lang, "selector.log.done_header")))
}

func (d *Discord) getModChannelConfig(lang i18n.Lang) (SelectionConfig, DatabaseOperations) {
	config := SelectionConfig{
		Prefix:          "mod_channel",
		ItemsPerPage:    channelsPerPage,
		Title:           i18n.T(lang, "selector.mod.title"),
		Description:     i18n.T(lang, "selector.mod.description"),
		Placeholder:     i18n.T(lang, "selector.mod.placeholder"),
		DoneDescription: i18n.T(lang, "selector.no_channel"),
		EmojiName:       "💬",
	}
	return config, &ModChannelDB{db: d.db}
}

func (d *Discord) handleModChannelSelection(s *discordgo.Session, i *discordgo.InteractionCreate) {
	lang := d.lang(i.GuildID)
	config, dbOps := d.getModChannelConfig(lang)
	d.handleSelection(s, i, config, dbOps, d.getTextChannelsAsItems, channelSelectionDoneMessage(i18n.T(lang, "selector.mod.done_header")))
}

func (d *Discord) getExcludedChannelConfig(lang i18n.Lang) (SelectionConfig, DatabaseOperations) {
	config := SelectionConfig{
		Prefix:          "excluded_channel",
		ItemsPerPage:    channelsPerPage,
		Title:           i18n.T(lang, "selector.excluded.title"),
		Description:     i18n.T(lang, "selector.excluded.description"),
		Placeholder:     i18n.T(lang, "selector.excluded.placeholder"),
		DoneDescription: i18n.T(lang, "selector.no_channel"),
		EmojiName:       "💬",
	}
	return config, &ExcludedChannelDB{db: d.db}
}

func (d *Discord) handleExcludedChannelSelection(s *discordgo.Session, i *discordgo.InteractionCreate) {
	lang := d.lang(i.GuildID)
	config, dbOps := d.getExcludedChannelConfig(lang)
	d.handleSelection(s, i, config, dbOps, d.getTextChannelsAsItems, channelSelectionDoneMessage(i18n.T(lang, "selector.excluded.done_header")))
}

func (d *Discord) getModRoleConfig(lang i18n.Lang) (SelectionConfig, DatabaseOperations) {
	config := SelectionConfig{
		Prefix:          "mod_role",
		ItemsPerPage:    rolesPerPage,
		Title:           i18n.T(lang, "selector.role.title"),
		Description:     i18n.T(lang, "selector.role.description"),
		Placeholder:     i18n.T(lang, "selector.role.placeholder"),
		DoneDescription: i18n.T(lang, "selector.no_role"),
		EmojiName:       "",
	}
	return config, &ModRoleDB{db: d.db}
}

func (d *Discord) handleModRoleSelection(s *discordgo.Session, i *discordgo.InteractionCreate) {
	lang := d.lang(i.GuildID)
	formatDoneMessage := func(items []SelectionItem) string {
		// Sort roles by position
		sort.Slice(items, func(i, j int) bool {
			return items[i].(RoleItem).Position > items[j].(RoleItem).Position
		})

		var roleNames []string
		for _, item := range items {
			roleNames = append(roleNames, fmt.Sprintf("- @%s", item.GetName()))
		}

		return i18n.T(lang, "selector.role.done", len(roleNames), strings.Join(roleNames, "\n"))
	}

	config, dbOps := d.getModRoleConfig(lang)
	d.handleSelection(s, i, config, dbOps, d.getRolesAsItems, formatDoneMessage)
}

func (d *Discord) getAutomodSettingsConfig(lang i18n.Lang) (SelectionConfig, DatabaseOperations) {
	config := SelectionConfig{
		Prefix:          "automod",
		ItemsPerPage:    25,
		Title:           i18n.T(lang, "selector.automod.title"),
		Description:     i18n.T(lang, "selector.automod.description"),
		Placeholder:     i18n.T(lang, "selector.automod.placeholder"),
		DoneDescription: i18n.T(lang, "selector.automod.none"),
		EmojiName:       "",
	}
	return config, &AutomodSettingsDB{db: d.db}
}

func (d *Discord) handleAutomoderationSettings(s *discordgo.Session, i *discordgo.InteractionCreate) {
	lang := d.lang(i.GuildID)
	formatDoneMessage := func(items []SelectionItem) string {
		var featureNames []string
		for _, item := range items {
			featureNames = append(featureNames, fmt.Sprintf("• %s", item.GetName()))
		}

		return i18n.T(lang, "selector.automod.done", len(featureNames), strings.Join(featureNames, "\n"))
	}

	config, dbOps := d.getAutomodSettingsConfig(lang)
	d.handleSelection(s, i, config, dbOps, d.getAutomoderationFeaturesAsItems, formatDoneMessage)
}

// getAutomoderationFeaturesAsItems returns the toggleable automod features,
// labelled in the guild's language. It matches the itemsFetcher signature
// (discordClient, guildID) so it can be passed to handleSelection; the session
// argument is unused.
func (d *Discord) getAutomoderationFeaturesAsItems(_ discordClient, guildID string) ([]SelectionItem, error) {
	lang := d.lang(guildID)
	features := []SelectionItem{
		AutomodFeatureItem{
			ID:          "banned_words",
			Name:        i18n.T(lang, "automod_feature.banned_words.name"),
			Description: i18n.T(lang, "automod_feature.banned_words.desc"),
		},
		AutomodFeatureItem{
			ID:          "banned_websites",
			Name:        i18n.T(lang, "automod_feature.banned_websites.name"),
			Description: i18n.T(lang, "automod_feature.banned_websites.desc"),
		},
		AutomodFeatureItem{
			ID:          "spam_detection",
			Name:        i18n.T(lang, "automod_feature.spam.name"),
			Description: i18n.T(lang, "automod_feature.spam.desc"),
		},
	}
	return features, nil
}
