package api

import (
	"github.com/bwmarrin/discordgo"
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

	items, err := d.getTextChannelsAsItems(s, i.GuildID)
	if err != nil {
		d.sendErrorMessage(s, i.Interaction, "Sélection des salons", "Une erreur est survenue lors de la récupération des salons.")
		return
	}

	config, dbOps := d.getLogChannelConfig()
	d.sendSelectPage(s, i.Interaction, items, 0, config, dbOps)
}

func (d *Discord) selectModeratorChannels(s discordClient, i *discordgo.InteractionCreate) {
	if !d.beginModCommand(s, i, "selectModeratorChannels()") {
		return
	}

	items, err := d.getTextChannelsAsItems(s, i.GuildID)
	if err != nil {
		d.sendErrorMessage(s, i.Interaction, "Sélection des salons", "Une erreur est survenue lors de la récupération des salons.")
		return
	}

	config, dbOps := d.getModChannelConfig()
	d.sendSelectPage(s, i.Interaction, items, 0, config, dbOps)
}

func (d *Discord) selectExcludedChannels(s discordClient, i *discordgo.InteractionCreate) {
	if !d.beginModCommand(s, i, "selectExcludedChannels()") {
		return
	}

	items, err := d.getTextChannelsAsItems(s, i.GuildID)
	if err != nil {
		d.sendErrorMessage(s, i.Interaction, "Sélection des salons", "Une erreur est survenue lors de la récupération des salons.")
		return
	}

	config, dbOps := d.getExcludedChannelConfig()
	d.sendSelectPage(s, i.Interaction, items, 0, config, dbOps)
}

func (d *Discord) selectModeratorRoles(s discordClient, i *discordgo.InteractionCreate) {
	if !d.beginModCommand(s, i, "selectModeratorRoles()") {
		return
	}

	items, err := d.getRolesAsItems(s, i.GuildID)
	if err != nil {
		d.sendErrorMessage(s, i.Interaction, "Sélection des roles", "Une erreur est survenue lors de la récupération des roles.")
		return
	}

	config, dbOps := d.getModRoleConfig()
	d.sendSelectPage(s, i.Interaction, items, 0, config, dbOps)
}
