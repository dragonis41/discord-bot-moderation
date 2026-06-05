package api

import (
	"github.com/bwmarrin/discordgo"
	"github.com/dragonis41/discord-bot-moderation/pkg/i18n"
)

type DiscordSettingsFunctionInterface interface {
	selectLogChannels(s discordClient, i *discordgo.InteractionCreate)
	selectModeratorChannels(s discordClient, i *discordgo.InteractionCreate)
	selectModeratorRoles(s discordClient, i *discordgo.InteractionCreate)
}

func (d *Discord) selectLogChannels(s discordClient, i *discordgo.InteractionCreate) {
	if !d.beginModCommand(s, i, "selectLogChannels()") {
		return
	}

	lang := d.lang(i.GuildID)

	items, err := d.getTextChannelsAsItems(s, i.GuildID)
	if err != nil {
		d.sendErrorMessage(s, i.Interaction, lang, i18n.T(lang, "selector.error_channels_title"), i18n.T(lang, "selector.error_channels"))
		return
	}

	config, dbOps := d.getLogChannelConfig(lang)
	d.sendSelectPage(s, i.Interaction, items, 0, config, dbOps)
}

func (d *Discord) selectModeratorChannels(s discordClient, i *discordgo.InteractionCreate) {
	if !d.beginModCommand(s, i, "selectModeratorChannels()") {
		return
	}

	lang := d.lang(i.GuildID)

	items, err := d.getTextChannelsAsItems(s, i.GuildID)
	if err != nil {
		d.sendErrorMessage(s, i.Interaction, lang, i18n.T(lang, "selector.error_channels_title"), i18n.T(lang, "selector.error_channels"))
		return
	}

	config, dbOps := d.getModChannelConfig(lang)
	d.sendSelectPage(s, i.Interaction, items, 0, config, dbOps)
}

func (d *Discord) selectExcludedChannels(s discordClient, i *discordgo.InteractionCreate) {
	if !d.beginModCommand(s, i, "selectExcludedChannels()") {
		return
	}

	lang := d.lang(i.GuildID)

	items, err := d.getTextChannelsAsItems(s, i.GuildID)
	if err != nil {
		d.sendErrorMessage(s, i.Interaction, lang, i18n.T(lang, "selector.error_channels_title"), i18n.T(lang, "selector.error_channels"))
		return
	}

	config, dbOps := d.getExcludedChannelConfig(lang)
	d.sendSelectPage(s, i.Interaction, items, 0, config, dbOps)
}

func (d *Discord) selectModeratorRoles(s discordClient, i *discordgo.InteractionCreate) {
	if !d.beginModCommand(s, i, "selectModeratorRoles()") {
		return
	}

	lang := d.lang(i.GuildID)

	items, err := d.getRolesAsItems(s, i.GuildID)
	if err != nil {
		d.sendErrorMessage(s, i.Interaction, lang, i18n.T(lang, "selector.error_roles_title"), i18n.T(lang, "selector.error_roles"))
		return
	}

	config, dbOps := d.getModRoleConfig(lang)
	d.sendSelectPage(s, i.Interaction, items, 0, config, dbOps)
}
