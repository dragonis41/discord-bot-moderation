package api

import (
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/bwmarrin/discordgo"
	"github.com/dragonis41/discord-bot-moderation/pkg/utils"
)

// TODO : Register this in a database
var selectedChannels = make(map[string][]string) // guildID -> []channelID

const channelsPerPage = 25

func (d *Discord) getTextChannels(s *discordgo.Session, guildID string) ([]*discordgo.Channel, error) {
	channels, err := s.GuildChannels(guildID)
	if err != nil {
		return nil, err
	}

	var textChannels []*discordgo.Channel
	for _, channel := range channels {
		if channel.Type == discordgo.ChannelTypeGuildText {
			textChannels = append(textChannels, channel)
		}
	}

	// Sort channels by position
	sort.Slice(textChannels, func(i, j int) bool {
		return textChannels[i].Position < textChannels[j].Position
	})

	return textChannels, nil
}

func (d *Discord) sendErrorMessage(s *discordgo.Session, i *discordgo.Interaction, title, description string) {
	_, _ = s.FollowupMessageCreate(i, true, &discordgo.WebhookParams{
		Embeds: []*discordgo.MessageEmbed{{
			Title:       title,
			Description: description,
			Color:       red,
			Timestamp:   time.Now().Format(time.RFC3339),
		}},
	})
}

func (d *Discord) sendChannelSelectPage(s *discordgo.Session, interaction *discordgo.Interaction, channels []*discordgo.Channel, page int) {
	embed, components := d.buildChannelSelectMessage(interaction.GuildID, channels, page)

	if interaction.Message != nil {
		// Update existing message
		if _, err := s.FollowupMessageEdit(interaction, interaction.Message.ID, &discordgo.WebhookEdit{
			Embeds:     &[]*discordgo.MessageEmbed{embed},
			Components: &components,
		}); err != nil {
			utils.LogError(fmt.Sprintf("sendChannelSelectPage: Error updating message: %s", err))
		}
	} else {
		// Create new message
		if _, err := s.FollowupMessageCreate(interaction, true, &discordgo.WebhookParams{
			Embeds:     []*discordgo.MessageEmbed{embed},
			Components: components,
		}); err != nil {
			utils.LogError(fmt.Sprintf("sendChannelSelectPage: Error sending follow-up message: %s", err))
		}
	}
}

func (d *Discord) buildChannelSelectMessage(guildID string, channels []*discordgo.Channel, page int) (*discordgo.MessageEmbed, []discordgo.MessageComponent) {
	totalPages := (len(channels) + channelsPerPage - 1) / channelsPerPage
	page = max(0, min(page, totalPages-1))

	start := page * channelsPerPage
	end := min(start+channelsPerPage, len(channels))

	// Build select menu options
	previouslySelected := selectedChannels[guildID]
	options := d.buildSelectMenuOptions(channels[start:end], previouslySelected)

	// Create components
	minVal := 0
	components := []discordgo.MessageComponent{
		discordgo.ActionsRow{
			Components: []discordgo.MessageComponent{
				discordgo.SelectMenu{
					CustomID:    fmt.Sprintf("channel_select_menu_%d", page),
					Placeholder: "Sélectionnez les salons de modération",
					MinValues:   &minVal,
					MaxValues:   len(options),
					Options:     options,
				},
			},
		},
	}

	// Add navigation buttons
	buttons := d.buildNavigationButtons(page, totalPages)
	if len(buttons) > 0 {
		components = append(components, discordgo.ActionsRow{Components: buttons})
	}

	embed := &discordgo.MessageEmbed{
		Title:       "Sélection des salons",
		Description: fmt.Sprintf("Sélectionnez les salons de modération puis cliquez sur \"Terminer\".\nCe sont les salons dans lesquels les modérateurs vont être notifiés.\n\n**%d**/**%d** salons sélectionnés.", len(previouslySelected), len(channels)),
		Color:       blue,
		Footer:      &discordgo.MessageEmbedFooter{Text: "Les sélections sont sauvegardées à chaque modification"},
		Timestamp:   time.Now().Format(time.RFC3339),
	}

	return embed, components
}

func (d *Discord) buildSelectMenuOptions(channels []*discordgo.Channel, selectedIDs []string) []discordgo.SelectMenuOption {
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

func (d *Discord) buildNavigationButtons(page, totalPages int) []discordgo.MessageComponent {
	var buttons []discordgo.MessageComponent

	if totalPages > 1 {
		buttons = append(buttons,
			discordgo.Button{
				Label:    "◀️ Précédent",
				Style:    discordgo.PrimaryButton,
				CustomID: fmt.Sprintf("channel_page_prev_%d", page),
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
				CustomID: fmt.Sprintf("channel_page_next_%d", page),
				Disabled: page == totalPages-1,
			},
		)
	}

	// Always add the done button
	buttons = append(buttons, discordgo.Button{
		Label:    "✅ Terminer",
		Style:    discordgo.SuccessButton,
		CustomID: "channel_select_done",
	})

	return buttons
}

func (d *Discord) handleChannelSelection(s *discordgo.Session, i *discordgo.InteractionCreate) {
	// Check if this is a message component interaction
	if i.Type != discordgo.InteractionMessageComponent {
		return
	}

	data := i.MessageComponentData()
	customID := data.CustomID

	// Respond immediately for all interactions
	if err := s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseUpdateMessage,
		Data: &discordgo.InteractionResponseData{Content: "Mise à jour..."},
	}); err != nil {
		utils.LogError(fmt.Sprintf("handleChannelSelection: Error responding: %s", err))
		return
	}

	switch {
	case strings.HasPrefix(customID, "channel_select_menu_"):
		d.handleChannelSelectionUpdate(s, i, data.Values)
	case strings.HasPrefix(customID, "channel_page_"):
		d.handlePageNavigation(s, i, customID)
	case customID == "channel_select_done":
		d.handleSelectionDone(s, i)
	}
}

func (d *Discord) handleChannelSelectionUpdate(s *discordgo.Session, i *discordgo.InteractionCreate, selectedOnPage []string) {
	var page int
	_, _ = fmt.Sscanf(i.MessageComponentData().CustomID, "channel_select_menu_%d", &page)

	textChannels, err := d.getTextChannels(s, i.GuildID)
	if err != nil {
		utils.LogError(fmt.Sprintf("handleChannelSelectionUpdate: Error fetching channels: %s", err))
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
	for _, id := range selectedChannels[i.GuildID] {
		if !pageChannelIDs[id] {
			newSelections[id] = true
		}
	}
	for _, id := range selectedOnPage {
		newSelections[id] = true
	}

	// Convert back to slice
	selectedChannels[i.GuildID] = nil
	for id := range newSelections {
		selectedChannels[i.GuildID] = append(selectedChannels[i.GuildID], id)
	}

	d.editChannelSelectMessage(s, i, textChannels, page)
}

func (d *Discord) handlePageNavigation(s *discordgo.Session, i *discordgo.InteractionCreate, customID string) {
	var currentPage int

	if strings.Contains(customID, "prev") {
		_, _ = fmt.Sscanf(customID, "channel_page_prev_%d", &currentPage)
		currentPage--
	} else {
		_, _ = fmt.Sscanf(customID, "channel_page_next_%d", &currentPage)
		currentPage++
	}

	textChannels, _ := d.getTextChannels(s, i.GuildID)
	d.editChannelSelectMessage(s, i, textChannels, currentPage)
}

func (d *Discord) handleSelectionDone(s *discordgo.Session, i *discordgo.InteractionCreate) {
	selectedIDs := selectedChannels[i.GuildID]
	var channelNames []string

	// Get all channels for the guild to ensure we have the latest data
	allChannels, err := s.GuildChannels(i.GuildID)
	if err != nil {
		utils.LogError(fmt.Sprintf("handleSelectionDone: Error fetching guild channels: %s", err))
	} else {
		// Create a map for quick lookup
		channelMap := make(map[string]*discordgo.Channel)
		for _, ch := range allChannels {
			channelMap[ch.ID] = ch
		}

		// Build the list of selected channel names
		for _, channelID := range selectedIDs {
			if channel, exists := channelMap[channelID]; exists && channel.Type == discordgo.ChannelTypeGuildText {
				channelNames = append(channelNames, fmt.Sprintf("#%s", channel.Name))
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
			Color:       green,
			Timestamp:   time.Now().Format(time.RFC3339),
		}},
		Components: &[]discordgo.MessageComponent{},
	})

	utils.LogInfo(fmt.Sprintf("User [%s] finished selecting %d channels for guild [%s]", i.Member.User.Username, len(selectedIDs), i.GuildID))
}

func (d *Discord) editChannelSelectMessage(s *discordgo.Session, i *discordgo.InteractionCreate, channels []*discordgo.Channel, page int) {
	embed, components := d.buildChannelSelectMessage(i.GuildID, channels, page)

	if _, err := s.InteractionResponseEdit(i.Interaction, &discordgo.WebhookEdit{
		Embeds:     &[]*discordgo.MessageEmbed{embed},
		Components: &components,
		Content:    nil,
	}); err != nil {
		utils.LogError(fmt.Sprintf("editChannelSelectMessage: Error editing message: %s", err))
	}
}
