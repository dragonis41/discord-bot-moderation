package api

import (
	"fmt"
	"sort"
	"strings"

	"github.com/bwmarrin/discordgo"
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

func (d *Discord) getLogChannelConfig() (SelectionConfig, DatabaseOperations) {
	config := SelectionConfig{
		Prefix:          "log_channel",
		ItemsPerPage:    channelsPerPage,
		Title:           "Sélection des salons de logs",
		Description:     "Sélectionnez les salons de logs puis cliquez sur \"Terminer\".\nCe sont les salons dans lesquels les actions de modération vont être loggées.",
		Placeholder:     "Sélectionnez les salons de logs",
		DoneDescription: "⚠️ Aucun salon sélectionné",
		EmojiName:       "💬",
	}
	return config, &LogChannelDB{db: d.db}
}

func (d *Discord) handleLogChannelSelection(s *discordgo.Session, i *discordgo.InteractionCreate) {
	config, dbOps := d.getLogChannelConfig()
	d.handleSelection(s, i, config, dbOps, d.getTextChannelsAsItems, channelSelectionDoneMessage("✅ Salons de logs sélectionnés :"))
}

func (d *Discord) getModChannelConfig() (SelectionConfig, DatabaseOperations) {
	config := SelectionConfig{
		Prefix:          "mod_channel",
		ItemsPerPage:    channelsPerPage,
		Title:           "Sélection des salons de modération",
		Description:     "Sélectionnez les salons de modération puis cliquez sur \"Terminer\".\nCe sont les salons dans lesquels les modérateurs vont être notifiés.",
		Placeholder:     "Sélectionnez les salons de modération",
		DoneDescription: "⚠️ Aucun salon sélectionné",
		EmojiName:       "💬",
	}
	return config, &ModChannelDB{db: d.db}
}

func (d *Discord) handleModChannelSelection(s *discordgo.Session, i *discordgo.InteractionCreate) {
	config, dbOps := d.getModChannelConfig()
	d.handleSelection(s, i, config, dbOps, d.getTextChannelsAsItems, channelSelectionDoneMessage("✅ Salons de modération sélectionnés :"))
}

func (d *Discord) getExcludedChannelConfig() (SelectionConfig, DatabaseOperations) {
	config := SelectionConfig{
		Prefix:          "excluded_channel",
		ItemsPerPage:    channelsPerPage,
		Title:           "Sélection des salons exclus",
		Description:     "Sélectionnez les salons exclus puis cliquez sur \"Terminer\".\nCe sont les salons qui ne seront pas pris en compte dans l'automodération.",
		Placeholder:     "Sélectionnez les salons exclus",
		DoneDescription: "⚠️ Aucun salon sélectionné",
		EmojiName:       "💬",
	}
	return config, &ExcludedChannelDB{db: d.db}
}

func (d *Discord) handleExcludedChannelSelection(s *discordgo.Session, i *discordgo.InteractionCreate) {
	config, dbOps := d.getExcludedChannelConfig()
	d.handleSelection(s, i, config, dbOps, d.getTextChannelsAsItems, channelSelectionDoneMessage("✅ Salons exclus sélectionnés :"))
}

func (d *Discord) getModRoleConfig() (SelectionConfig, DatabaseOperations) {
	config := SelectionConfig{
		Prefix:          "mod_role",
		ItemsPerPage:    rolesPerPage,
		Title:           "Sélection des roles de modération",
		Description:     "Sélectionnez les roles de modération puis cliquez sur \"Terminer\".\nCe sont les roles qui sont administrateurs du serveur et qui seront notifiés.\n\n⚠️ Attention, si vous ne possédez pas au moins un de ces rôles, vous ne pourrez plus utiliser les commandes d'administration !",
		Placeholder:     "Sélectionnez les roles de modération",
		DoneDescription: "⚠️ Aucun role sélectionné.",
		EmojiName:       "",
	}
	return config, &ModRoleDB{db: d.db}
}

func (d *Discord) handleModRoleSelection(s *discordgo.Session, i *discordgo.InteractionCreate) {
	formatDoneMessage := func(items []SelectionItem) string {
		// Sort roles by position
		sort.Slice(items, func(i, j int) bool {
			return items[i].(RoleItem).Position > items[j].(RoleItem).Position
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

func (d *Discord) getAutomodSettingsConfig() (SelectionConfig, DatabaseOperations) {
	config := SelectionConfig{
		Prefix:          "automod",
		ItemsPerPage:    25,
		Title:           "⚙️ Configuration Automodération",
		Description:     "Sélectionnez les fonctionnalités d'automodération à activer puis cliquez sur \"Terminer\".",
		Placeholder:     "Sélectionnez les fonctionnalités à activer",
		DoneDescription: "⚠️ Toutes les fonctionnalités d'automodération sont désactivées",
		EmojiName:       "",
	}
	return config, &AutomodSettingsDB{db: d.db}
}

func (d *Discord) handleAutomoderationSettings(s *discordgo.Session, i *discordgo.InteractionCreate) {
	formatDoneMessage := func(items []SelectionItem) string {
		var featureNames []string
		for _, item := range items {
			featureNames = append(featureNames, fmt.Sprintf("• %s", item.GetName()))
		}

		return fmt.Sprintf("✅ Configuration enregistrée avec succès!\n\n**Fonctionnalités activées** (%d):\n%s", len(featureNames), strings.Join(featureNames, "\n"))
	}

	config, dbOps := d.getAutomodSettingsConfig()
	d.handleSelection(s, i, config, dbOps, d.getAutomoderationFeaturesAsItems, formatDoneMessage)
}

func (d *Discord) getAutomoderationFeaturesAsItems(s *discordgo.Session, guildID string) ([]SelectionItem, error) {
	features := []SelectionItem{
		AutomodFeatureItem{
			ID:          "banned_words",
			Name:        "📝 Mots interdits",
			Description: "Détecte et supprime les messages contenant des mots interdits",
		},
		AutomodFeatureItem{
			ID:          "banned_websites",
			Name:        "🌐 Sites web interdits",
			Description: "Bloque les liens vers des sites web interdits",
		},
		AutomodFeatureItem{
			ID:          "spam_detection",
			Name:        "🚨 Détection de spam",
			Description: "Bannit automatiquement les utilisateurs qui spamment des messages",
		},
	}
	return features, nil
}
