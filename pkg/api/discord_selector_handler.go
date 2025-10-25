package api

import (
	"fmt"
	"sort"
	"strings"

	"github.com/bwmarrin/discordgo"
)

const channelsPerPage = 25
const rolesPerPage = 25

func (d *Discord) getLogChannelConfig() (SelectionConfig, DatabaseOperations) {
	config := SelectionConfig{
		Prefix:          "log_channel",
		ItemsPerPage:    channelsPerPage,
		Title:           "Sélection des salons",
		Description:     "Sélectionnez les salons de logs puis cliquez sur \"Terminer\".\nCe sont les salons dans lesquels les actions de modération vont être loggées.",
		Placeholder:     "Sélectionnez les salons de logs",
		DoneDescription: "⚠️ Aucun salon sélectionné",
		EmojiName:       "💬",
	}
	return config, &LogChannelDB{db: d.db}
}

func (d *Discord) handleLogChannelSelection(s *discordgo.Session, i *discordgo.InteractionCreate) {
	formatDoneMessage := func(items []SelectionItem) string {
		// Sort channels by position
		sort.Slice(items, func(i, j int) bool {
			chI := items[i].(ChannelItem)
			chJ := items[j].(ChannelItem)
			return chI.Position < chJ.Position
		})

		var channelNames []string
		for _, item := range items {
			channelNames = append(channelNames, fmt.Sprintf("- <#%s>", item.GetID()))
		}

		return "✅ Salons de logs sélectionnés :\n" + strings.Join(channelNames, "\n")
	}

	config, dbOps := d.getLogChannelConfig()
	d.handleSelection(s, i, config, dbOps, d.getTextChannelsAsItems, formatDoneMessage)
}

func (d *Discord) getModChannelConfig() (SelectionConfig, DatabaseOperations) {
	config := SelectionConfig{
		Prefix:          "mod_channel",
		ItemsPerPage:    channelsPerPage,
		Title:           "Sélection des salons",
		Description:     "Sélectionnez les salons de modération puis cliquez sur \"Terminer\".\nCe sont les salons dans lesquels les modérateurs vont être notifiés.",
		Placeholder:     "Sélectionnez les salons de modération",
		DoneDescription: "⚠️ Aucun salon sélectionné",
		EmojiName:       "💬",
	}
	return config, &ModChannelDB{db: d.db}
}

func (d *Discord) handleModChannelSelection(s *discordgo.Session, i *discordgo.InteractionCreate) {
	formatDoneMessage := func(items []SelectionItem) string {
		// Sort channels by position
		sort.Slice(items, func(i, j int) bool {
			chI := items[i].(ChannelItem)
			chJ := items[j].(ChannelItem)
			return chI.Position < chJ.Position
		})

		var channelNames []string
		for _, item := range items {
			channelNames = append(channelNames, fmt.Sprintf("- <#%s>", item.GetID()))
		}

		return "✅ Salons de modération sélectionnés :\n" + strings.Join(channelNames, "\n")
	}

	config, dbOps := d.getModChannelConfig()
	d.handleSelection(s, i, config, dbOps, d.getTextChannelsAsItems, formatDoneMessage)
}

func (d *Discord) getModRoleConfig() (SelectionConfig, DatabaseOperations) {
	config := SelectionConfig{
		Prefix:          "mod_role",
		ItemsPerPage:    rolesPerPage,
		Title:           "Sélection des roles",
		Description:     "Sélectionnez les roles de modération puis cliquez sur \"Terminer\".\nCe sont les roles qui sont administrateurs du serveur et qui seront notifiés.\n\n⚠️ Attention, si vous ne possédez pas au moins un de ces rôles, vous ne pourrez plus utiliser les commandes d'administration !",
		Placeholder:     "Sélectionnez les roles de modération",
		DoneDescription: "⚠️ Aucun role sélectionné.",
		EmojiName:       "",
	}
	return config, &ModRoleDB{db: d.db}
}

func (d *Discord) handleModRoleSelection(s *discordgo.Session, i *discordgo.InteractionCreate) {
	formatDoneMessage := func(items []SelectionItem) string {
		// Sort roles by position (descending, higher positions first)
		sort.Slice(items, func(i, j int) bool {
			roleI := items[i].(RoleItem)
			roleJ := items[j].(RoleItem)
			return roleI.Position > roleJ.Position
		})

		var roleNames []string
		for _, item := range items {
			roleNames = append(roleNames, fmt.Sprintf("- @%s", item.GetName()))
		}

		return fmt.Sprintf("✅ %d roles sélectionnés:\n%s", len(roleNames), strings.Join(roleNames, "\n"))
	}

	config, dbOps := d.getModRoleConfig()
	d.handleSelection(s, i, config, dbOps, d.getRolesAsItems, formatDoneMessage)
}
