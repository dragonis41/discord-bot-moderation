package api

import (
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/bwmarrin/discordgo"
	"github.com/dragonis41/discord-bot-moderation/pkg/logger"
	"github.com/dragonis41/discord-bot-moderation/pkg/model"
)

func (d *Discord) sendModChannelSelectPage(s *discordgo.Session, interaction *discordgo.Interaction, channels []*discordgo.Channel, page int) {
	embed, components := d.buildModChannelSelectMessage(interaction.GuildID, channels, page)

	if interaction.Message != nil {
		// Update existing message
		if _, err := s.FollowupMessageEdit(interaction, interaction.Message.ID, &discordgo.WebhookEdit{
			Embeds:     &[]*discordgo.MessageEmbed{embed},
			Components: &components,
		}); err != nil {
			d.log.LogError(logger.LogModel{Database: d.db, GuildID: interaction.GuildID, Function: "sendModChannelSelectPage()",
				Message: fmt.Sprintf("Error updating message: %s", err),
			})
		}
	} else {
		// Create new message
		if _, err := s.FollowupMessageCreate(interaction, true, &discordgo.WebhookParams{
			Embeds:     []*discordgo.MessageEmbed{embed},
			Components: components,
		}); err != nil {
			d.log.LogError(logger.LogModel{Database: d.db, GuildID: interaction.GuildID, Function: "sendModChannelSelectPage()",
				Message: fmt.Sprintf("Error sending follow-up message: %s", err),
			})
		}
	}
}

func (d *Discord) buildModChannelSelectMessage(guildID string, channels []*discordgo.Channel, page int) (*discordgo.MessageEmbed, []discordgo.MessageComponent) {
	totalPages := (len(channels) + channelsPerPage - 1) / channelsPerPage
	page = max(0, min(page, totalPages-1))

	start := page * channelsPerPage
	end := min(start+channelsPerPage, len(channels))

	// Build select menu options
	previouslySelected, err := d.db.GetModerationChannelsByGuildId(guildID)
	if err != nil {
		d.log.LogError(logger.LogModel{Database: d.db, GuildID: guildID, Function: "buildModChannelSelectMessage()",
			Message: fmt.Sprintf("Error fetching selected channels: %s", err),
		})
	}
	options := d.buildModChannelSelectMenuOptions(channels[start:end], previouslySelected)

	// Create components
	minVal := 0
	components := []discordgo.MessageComponent{
		discordgo.ActionsRow{
			Components: []discordgo.MessageComponent{
				discordgo.SelectMenu{
					CustomID:    fmt.Sprintf("mod_channel_select_menu_%d", page),
					Placeholder: "Sélectionnez les salons de modération",
					MinValues:   &minVal,
					MaxValues:   len(options),
					Options:     options,
				},
			},
		},
	}

	// Add navigation buttons
	buttons := d.buildModChannelNavigationButtons(page, totalPages)
	if len(buttons) > 0 {
		components = append(components, discordgo.ActionsRow{Components: buttons})
	}

	embed := &discordgo.MessageEmbed{
		Title: "Sélection des salons",
		Description: fmt.Sprintf("Sélectionnez les salons de modération puis cliquez sur \"Terminer\".\n"+
			"Ce sont les salons dans lesquels les modérateurs vont être notifiés.\n\n"+
			"**%d**/**%d** salons sélectionnés.", len(previouslySelected), len(channels)),
		Color:     model.Blue.Int(),
		Footer:    model.SelectionMenuFooter,
		Timestamp: time.Now().Format(time.RFC3339),
	}

	return embed, components
}

func (d *Discord) buildModChannelSelectMenuOptions(channels []*discordgo.Channel, selectedIDs []string) []discordgo.SelectMenuOption {
	selectedMap := make(map[string]bool)
	for _, id := range selectedIDs {
		selectedMap[id] = true
	}

	options := make([]discordgo.SelectMenuOption, len(channels))
	for i, channel := range channels {
		options[i] = discordgo.SelectMenuOption{
			Label:       channel.Name,
			Value:       channel.ID,
			Description: channel.Topic,
			Default:     selectedMap[channel.ID],
			Emoji:       &discordgo.ComponentEmoji{Name: "💬"},
		}
	}
	return options
}

func (d *Discord) buildModChannelNavigationButtons(page, totalPages int) []discordgo.MessageComponent {
	var buttons []discordgo.MessageComponent

	if totalPages > 1 {
		buttons = append(buttons,
			discordgo.Button{
				Label:    "◀️ Précédent",
				Style:    discordgo.PrimaryButton,
				CustomID: fmt.Sprintf("mod_channel_page_prev_%d", page),
				Disabled: page == 0,
			},
			discordgo.Button{
				Label:    fmt.Sprintf("Page %d/%d", page+1, totalPages),
				Style:    discordgo.SecondaryButton,
				CustomID: "page_indicator",
				Disabled: true,
			},
			discordgo.Button{
				Label:    "Suivant ▶️",
				Style:    discordgo.PrimaryButton,
				CustomID: fmt.Sprintf("mod_channel_page_next_%d", page),
				Disabled: page == totalPages-1,
			},
		)
	}

	// Always add the done button
	buttons = append(buttons, discordgo.Button{
		Label:    "✅ Terminer",
		Style:    discordgo.SuccessButton,
		CustomID: "mod_channel_select_done",
	})

	return buttons
}

func (d *Discord) handleModChannelSelection(s *discordgo.Session, i *discordgo.InteractionCreate) {
	// Check if this is a message component interaction
	if i.Type != discordgo.InteractionMessageComponent {
		return
	}

	data := i.MessageComponentData()
	customID := data.CustomID

	if !strings.HasPrefix(customID, "mod_channel_select") && !strings.HasPrefix(customID, "mod_channel_page") {
		return
	}

	// Respond immediately for all interactions
	if err := s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseUpdateMessage,
		Data: &discordgo.InteractionResponseData{Content: "Mise à jour..."},
	}); err != nil {
		d.log.LogError(logger.LogModel{Database: d.db, GuildID: i.GuildID, Function: "handleModChannelSelection()",
			Message: fmt.Sprintf("Error responding to interaction: %s", err),
		})
		return
	}

	switch {
	case strings.HasPrefix(customID, "mod_channel_select_menu_"):
		d.handleModChannelSelectionUpdate(s, i, data.Values)
	case strings.HasPrefix(customID, "mod_channel_page_"):
		d.handleModChannelPageNavigation(s, i, customID)
	case customID == "mod_channel_select_done":
		d.handleModChannelSelectionDone(s, i)
	}
}

func (d *Discord) handleModChannelSelectionUpdate(s *discordgo.Session, i *discordgo.InteractionCreate, selectedOnPage []string) {
	var page int
	_, _ = fmt.Sscanf(i.MessageComponentData().CustomID, "mod_channel_select_menu_%d", &page)

	textChannels, err := d.getTextChannels(s, i.GuildID)
	if err != nil {
		d.log.LogError(logger.LogModel{Database: d.db, GuildID: i.GuildID, Function: "handleModChannelSelectionUpdate()",
			Message: fmt.Sprintf("Error fetching channels: %s", err),
		})
		return
	}

	// Update selections
	start := page * channelsPerPage
	end := min(start+channelsPerPage, len(textChannels))

	pageChannelIDs := make(map[string]bool)
	for _, ch := range textChannels[start:end] {
		pageChannelIDs[ch.ID] = true
	}

	// Merge selections
	newSelections := make(map[string]bool)
	selectedChannels, err := d.db.GetModerationChannelsByGuildId(i.GuildID)
	if err != nil {
		d.log.LogError(logger.LogModel{Database: d.db, GuildID: i.GuildID, Function: "handleModChannelSelectionUpdate()",
			Message: fmt.Sprintf("Error fetching selected channels: %s", err),
		})
	}
	for _, id := range selectedChannels {
		if !pageChannelIDs[id] {
			newSelections[id] = true
		}
	}
	for _, id := range selectedOnPage {
		newSelections[id] = true
	}

	// Clear the database and re-add selections
	err = d.db.RemoveModerationChannelsByGuild(i.GuildID)
	if err != nil {
		d.log.LogError(logger.LogModel{Database: d.db, GuildID: i.GuildID, Function: "handleModChannelSelectionUpdate()",
			Message: fmt.Sprintf("Error clearing selected channels: %s", err),
		})
	}
	for id := range newSelections {
		err := d.db.AddModerationChannel(i.GuildID, id)
		if err != nil {
			d.log.LogError(logger.LogModel{Database: d.db, GuildID: i.GuildID, Function: "handleModChannelSelectionUpdate()",
				Message: fmt.Sprintf("Error adding selected channel [%s]: %s", id, err),
			})
		}
	}

	d.editModChannelSelectMessage(s, i, textChannels, page)
}

func (d *Discord) handleModChannelPageNavigation(s *discordgo.Session, i *discordgo.InteractionCreate, customID string) {
	var currentPage int

	if strings.Contains(customID, "prev") {
		_, _ = fmt.Sscanf(customID, "mod_channel_page_prev_%d", &currentPage)
		currentPage--
	} else {
		_, _ = fmt.Sscanf(customID, "mod_channel_page_next_%d", &currentPage)
		currentPage++
	}

	textChannels, _ := d.getTextChannels(s, i.GuildID)
	d.editModChannelSelectMessage(s, i, textChannels, currentPage)
}

func (d *Discord) handleModChannelSelectionDone(s *discordgo.Session, i *discordgo.InteractionCreate) {
	selectedIDs, err := d.db.GetModerationChannelsByGuildId(i.GuildID)
	if err != nil {
		d.log.LogError(logger.LogModel{Database: d.db, GuildID: i.GuildID, Function: "handleModChannelSelectionDone()",
			Message: fmt.Sprintf("Error fetching selected channels: %s", err),
		})
	}
	var channelNames []string

	// Get all channels for the guild to ensure we have the latest data
	allChannels, err := s.GuildChannels(i.GuildID)
	if err != nil {
		d.log.LogError(logger.LogModel{Database: d.db, GuildID: i.GuildID, Function: "handleModChannelSelectionDone()",
			Message: fmt.Sprintf("Error fetching guild channels: %s", err),
		})
	} else {
		// Create a map for quick lookup
		channelMap := make(map[string]*discordgo.Channel)
		for _, ch := range allChannels {
			channelMap[ch.ID] = ch
		}

		// Build the list of selected channel names
		for _, channelID := range selectedIDs {
			if channel, exists := channelMap[channelID]; exists && channel.Type == discordgo.ChannelTypeGuildText {
				channelNames = append(channelNames, fmt.Sprintf("- #%s", channel.Name))
			}
		}
	}

	description := "⚠️ Aucun salon sélectionné"
	if len(channelNames) > 0 {
		// Sort channel names for consistent display
		sort.Strings(channelNames)
		description = fmt.Sprintf("✅ %d salons sélectionnés:\n%s", len(channelNames), strings.Join(channelNames, "\n"))
	}

	_, _ = s.InteractionResponseEdit(i.Interaction, &discordgo.WebhookEdit{
		Embeds: &[]*discordgo.MessageEmbed{{
			Title:       "Configuration terminée",
			Description: description,
			Color:       model.Green.Int(),
			Footer:      model.DefaultFooter,
			Timestamp:   time.Now().Format(time.RFC3339),
		}},
		Components: &[]discordgo.MessageComponent{},
	})

	d.log.LogInfo(logger.LogModel{Database: d.db, GuildID: i.GuildID, Function: "handleModChannelSelectionDone()",
		Message: fmt.Sprintf("User [%s] finished selecting %d mod channels for guild [%s]", i.Member.User.Username, len(selectedIDs), i.GuildID),
	})
}

func (d *Discord) editModChannelSelectMessage(s *discordgo.Session, i *discordgo.InteractionCreate, channels []*discordgo.Channel, page int) {
	embed, components := d.buildModChannelSelectMessage(i.GuildID, channels, page)

	if _, err := s.InteractionResponseEdit(i.Interaction, &discordgo.WebhookEdit{
		Embeds:     &[]*discordgo.MessageEmbed{embed},
		Components: &components,
		Content:    nil,
	}); err != nil {
		d.log.LogError(logger.LogModel{Database: d.db, GuildID: i.GuildID, Function: "editModChannelSelectMessage()",
			Message: fmt.Sprintf("Error editing message: %s", err),
		})
	}
}
