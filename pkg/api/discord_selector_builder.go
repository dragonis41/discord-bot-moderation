package api

import (
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/bwmarrin/discordgo"
	"github.com/dragonis41/discord-bot-moderation/internal/database"
	"github.com/dragonis41/discord-bot-moderation/pkg/model"
)

// SelectionConfig defines the configuration for a selection menu
type SelectionConfig struct {
	Prefix          string // e.g., "log_channel", "mod_channel", "role"
	ItemsPerPage    int    // Max items per page before pagination (the is Discord limit of 25)
	Title           string // Embed title
	Description     string // Embed description
	Placeholder     string // Select menu placeholder
	DoneDescription string // Description to show when the selection is done
	EmojiName       string // Optional, this is the emoji before the option label
}

// SelectionItem represents a generic item that can be selected
type SelectionItem interface {
	GetID() string
	GetName() string
	GetDescription() string
}

// ChannelItem wraps a discordgo.Channel
type ChannelItem struct {
	*discordgo.Channel
}

func (c ChannelItem) GetID() string          { return c.ID }
func (c ChannelItem) GetName() string        { return c.Name }
func (c ChannelItem) GetDescription() string { return c.Topic }

// RoleItem wraps a discordgo.Role
type RoleItem struct {
	*discordgo.Role
}

func (r RoleItem) GetID() string          { return r.ID }
func (r RoleItem) GetName() string        { return r.Name }
func (r RoleItem) GetDescription() string { return "" }

// AutomodFeatureItem represents an automoderation feature
type AutomodFeatureItem struct {
	ID          string
	Name        string
	Description string
}

func (a AutomodFeatureItem) GetID() string          { return a.ID }
func (a AutomodFeatureItem) GetName() string        { return a.Name }
func (a AutomodFeatureItem) GetDescription() string { return a.Description }

// DatabaseOperations defines the database operations needed
type DatabaseOperations interface {
	GetSelected(guildID string) ([]string, error)
	RemoveByGuild(guildID string) error
	Add(guildID, itemID string) error
}

// sendSelectPage sends or updates the selection page
func (d *Discord) sendSelectPage(
	s discordClient,
	interaction *discordgo.Interaction,
	items []SelectionItem,
	page int,
	config SelectionConfig,
	dbOps DatabaseOperations,
) {
	embed, components := d.buildSelectMessage(interaction.GuildID, items, page, config, dbOps)

	if interaction.Message != nil {
		// Update existing message
		if _, err := s.FollowupMessageEdit(interaction, interaction.Message.ID, &discordgo.WebhookEdit{
			Embeds:     &[]*discordgo.MessageEmbed{embed},
			Components: &components,
		}); err != nil {
			d.logError(interaction.GuildID, "sendSelectPage()", "Error updating message: %s", err)
		}
	} else {
		// Create new message
		if _, err := s.FollowupMessageCreate(interaction, true, &discordgo.WebhookParams{
			Embeds:     []*discordgo.MessageEmbed{embed},
			Components: components,
		}); err != nil {
			d.logError(interaction.GuildID, "sendSelectPage()", "Error sending follow-up message: %s", err)
		}
	}
}

// buildSelectMessage builds the embed and components for the selection menu
func (d *Discord) buildSelectMessage(
	guildID string,
	items []SelectionItem,
	page int,
	config SelectionConfig,
	dbOps DatabaseOperations,
) (*discordgo.MessageEmbed, []discordgo.MessageComponent) {
	totalPages := (len(items) + config.ItemsPerPage - 1) / config.ItemsPerPage
	page = max(0, min(page, totalPages-1))

	start := page * config.ItemsPerPage
	end := min(start+config.ItemsPerPage, len(items))

	// Build select menu options
	previouslySelected, err := dbOps.GetSelected(guildID)
	if err != nil {
		d.logError(guildID, "buildSelectMessage()", "Error fetching selected items: %s", err)
	}
	options := d.buildSelectMenuOptions(items[start:end], previouslySelected, config.EmojiName)

	// Create components
	minVal := 0
	components := []discordgo.MessageComponent{
		discordgo.ActionsRow{
			Components: []discordgo.MessageComponent{
				discordgo.SelectMenu{
					CustomID:    fmt.Sprintf("%s_select_menu_%d", config.Prefix, page),
					Placeholder: config.Placeholder,
					MinValues:   &minVal,
					MaxValues:   len(options),
					Options:     options,
				},
			},
		},
	}

	// Add navigation buttons
	buttons := d.buildNavigationButtons(page, totalPages, config.Prefix)
	if len(buttons) > 0 {
		components = append(components, discordgo.ActionsRow{Components: buttons})
	}

	description := fmt.Sprintf("%s\n\n**%d**/**%d** item sélectionnés.",
		config.Description, len(previouslySelected), len(items))

	embed := &discordgo.MessageEmbed{
		Title:       config.Title,
		Description: description,
		Color:       model.Blue.Int(),
		Footer:      model.SelectionMenuFooter,
		Timestamp:   time.Now().UTC().Format(time.RFC3339),
	}

	return embed, components
}

// buildSelectMenuOptions builds the select menu options
func (d *Discord) buildSelectMenuOptions(
	items []SelectionItem,
	selectedIDs []string,
	emojiName string,
) []discordgo.SelectMenuOption {
	selectedMap := make(map[string]bool)
	for _, id := range selectedIDs {
		selectedMap[id] = true
	}

	options := make([]discordgo.SelectMenuOption, len(items))
	for i, item := range items {
		description := item.GetDescription()

		// Discord has a 100 character limit for option descriptions
		if len(description) > 100 {
			description = description[:97] + "..."
		}

		option := discordgo.SelectMenuOption{
			Label:       item.GetName(),
			Value:       item.GetID(),
			Description: description,
			Default:     selectedMap[item.GetID()],
		}

		if emojiName != "" {
			option.Emoji = &discordgo.ComponentEmoji{Name: emojiName}
		}

		options[i] = option
	}
	return options
}

// buildNavigationButtons builds the navigation buttons
func (d *Discord) buildNavigationButtons(page, totalPages int, prefix string) []discordgo.MessageComponent {
	var buttons []discordgo.MessageComponent

	if totalPages > 1 {
		buttons = append(buttons,
			discordgo.Button{
				Label:    "◀️ Précédent",
				Style:    discordgo.PrimaryButton,
				CustomID: fmt.Sprintf("%s_page_prev_%d", prefix, page),
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
				CustomID: fmt.Sprintf("%s_page_next_%d", prefix, page),
				Disabled: page == totalPages-1,
			},
		)
	}

	// Always add the done button
	buttons = append(buttons, discordgo.Button{
		Label:    "✅ Terminer",
		Style:    discordgo.SuccessButton,
		CustomID: fmt.Sprintf("%s_select_done", prefix),
	})

	return buttons
}

// handleSelection is the generic handler for all selection interactions
func (d *Discord) handleSelection(
	s discordClient,
	i *discordgo.InteractionCreate,
	config SelectionConfig,
	dbOps DatabaseOperations,
	itemsFetcher func(discordClient, string) ([]SelectionItem, error),
	formatDoneMessage func([]SelectionItem) string,
) {
	// Check if this is a message component interaction
	if i.Type != discordgo.InteractionMessageComponent {
		return
	}

	data := i.MessageComponentData()
	customID := data.CustomID

	if !strings.HasPrefix(customID, config.Prefix+"_select") &&
		!strings.HasPrefix(customID, config.Prefix+"_page") {
		return
	}

	// Respond immediately for all interactions
	if err := s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseUpdateMessage,
		Data: &discordgo.InteractionResponseData{Content: "Mise à jour..."},
	}); err != nil {
		d.logError(i.GuildID, "handleSelection()", "Error responding to interaction: %s", err)
		return
	}

	switch {
	case strings.HasPrefix(customID, config.Prefix+"_select_menu_"):
		d.handleSelectionUpdate(s, i, data.Values, config, dbOps, itemsFetcher)
	case strings.HasPrefix(customID, config.Prefix+"_page_"):
		d.handlePageNavigation(s, i, customID, config, itemsFetcher, dbOps)
	case customID == config.Prefix+"_select_done":
		d.handleSelectionDone(s, i, config, dbOps, itemsFetcher, formatDoneMessage)
	}
}

// handleSelectionUpdate updates the selection
func (d *Discord) handleSelectionUpdate(
	s discordClient,
	i *discordgo.InteractionCreate,
	selectedOnPage []string,
	config SelectionConfig,
	dbOps DatabaseOperations,
	itemsFetcher func(discordClient, string) ([]SelectionItem, error),
) {
	var page int
	_, _ = fmt.Sscanf(i.MessageComponentData().CustomID, config.Prefix+"_select_menu_%d", &page)

	items, err := itemsFetcher(s, i.GuildID)
	if err != nil {
		d.logError(i.GuildID, "handleSelectionUpdate()", "Error fetching items: %s", err)
		return
	}

	// Update selections
	start := page * config.ItemsPerPage
	end := min(start+config.ItemsPerPage, len(items))

	pageItemIDs := make(map[string]bool)
	for _, item := range items[start:end] {
		pageItemIDs[item.GetID()] = true
	}

	// Merge selections
	newSelections := make(map[string]bool)
	selectedItems, err := dbOps.GetSelected(i.GuildID)
	if err != nil {
		d.logError(i.GuildID, "handleSelectionUpdate()", "Error fetching selected items: %s", err)
	}
	for _, id := range selectedItems {
		if !pageItemIDs[id] {
			newSelections[id] = true
		}
	}
	for _, id := range selectedOnPage {
		newSelections[id] = true
	}

	// Clear the database and re-add selections
	err = dbOps.RemoveByGuild(i.GuildID)
	if err != nil {
		d.logError(i.GuildID, "handleSelectionUpdate()", "Error clearing selected items: %s", err)
	}
	for id := range newSelections {
		if err := dbOps.Add(i.GuildID, id); err != nil {
			d.logError(i.GuildID, "handleSelectionUpdate()", "Error adding selected item [%s]: %s", id, err)
		}
	}

	d.editSelectMessage(s, i, items, page, config, dbOps)
}

// handlePageNavigation handles page navigation
func (d *Discord) handlePageNavigation(
	s discordClient,
	i *discordgo.InteractionCreate,
	customID string,
	config SelectionConfig,
	itemsFetcher func(discordClient, string) ([]SelectionItem, error),
	dbOps DatabaseOperations,
) {
	var currentPage int

	if strings.Contains(customID, "prev") {
		_, _ = fmt.Sscanf(customID, config.Prefix+"_page_prev_%d", &currentPage)
		currentPage--
	} else {
		_, _ = fmt.Sscanf(customID, config.Prefix+"_page_next_%d", &currentPage)
		currentPage++
	}

	items, _ := itemsFetcher(s, i.GuildID)
	d.editSelectMessage(s, i, items, currentPage, config, dbOps)
}

// handleSelectionDone handles the completion of selection
func (d *Discord) handleSelectionDone(
	s discordClient,
	i *discordgo.InteractionCreate,
	config SelectionConfig,
	dbOps DatabaseOperations,
	itemsFetcher func(discordClient, string) ([]SelectionItem, error),
	formatDoneMessage func([]SelectionItem) string,
) {
	selectedIDs, err := dbOps.GetSelected(i.GuildID)
	if err != nil {
		d.logError(i.GuildID, "handleSelectionDone()", "Error fetching selected items: %s", err)
	}

	description := config.DoneDescription
	if len(selectedIDs) > 0 && formatDoneMessage != nil {
		// Fetch all items to get their details
		allItems, err := itemsFetcher(s, i.GuildID)
		if err != nil {
			d.logError(i.GuildID, "handleSelectionDone()", "Error fetching items for done message: %s", err)
		} else {
			// Filter only selected items
			selectedItemsMap := make(map[string]bool)
			for _, id := range selectedIDs {
				selectedItemsMap[id] = true
			}

			var selectedItems []SelectionItem
			for _, item := range allItems {
				if selectedItemsMap[item.GetID()] {
					selectedItems = append(selectedItems, item)
				}
			}

			if len(selectedItems) > 0 {
				description = formatDoneMessage(selectedItems)
			}
		}
	}

	_, _ = s.InteractionResponseEdit(i.Interaction, &discordgo.WebhookEdit{
		Embeds: &[]*discordgo.MessageEmbed{{
			Title:       "Configuration terminée",
			Description: description,
			Color:       model.Green.Int(),
			Timestamp:   time.Now().UTC().Format(time.RFC3339),
		}},
		Components: &[]discordgo.MessageComponent{},
	})

	d.logInfo(i.GuildID, "handleSelectionDone()", "User [%s] finished selecting %d items for guild [%s]", i.Member.User.Username, len(selectedIDs), i.GuildID)
}

// editSelectMessage edits the selection message
func (d *Discord) editSelectMessage(
	s discordClient,
	i *discordgo.InteractionCreate,
	items []SelectionItem,
	page int,
	config SelectionConfig,
	dbOps DatabaseOperations,
) {
	embed, components := d.buildSelectMessage(i.GuildID, items, page, config, dbOps)

	if _, err := s.InteractionResponseEdit(i.Interaction, &discordgo.WebhookEdit{
		Embeds:     &[]*discordgo.MessageEmbed{embed},
		Components: &components,
		Content:    nil,
	}); err != nil {
		d.logError(i.GuildID, "editSelectMessage()", "Error editing message: %s", err)
	}
}

// Database operation wrappers for each type
type LogChannelDB struct {
	db store
}

func (l *LogChannelDB) GetSelected(guildID string) ([]string, error) {
	return l.db.GetLogChannelsByGuildId(guildID)
}

func (l *LogChannelDB) RemoveByGuild(guildID string) error {
	return l.db.RemoveLogChannelsByGuild(guildID)
}

func (l *LogChannelDB) Add(guildID, itemID string) error {
	return l.db.AddLogChannel(guildID, itemID)
}

type ModChannelDB struct {
	db store
}

func (m *ModChannelDB) GetSelected(guildID string) ([]string, error) {
	return m.db.GetModerationChannelsByGuildId(guildID)
}

func (m *ModChannelDB) RemoveByGuild(guildID string) error {
	return m.db.RemoveModerationChannelsByGuild(guildID)
}

func (m *ModChannelDB) Add(guildID, itemID string) error {
	return m.db.AddModerationChannel(guildID, itemID)
}

type ExcludedChannelDB struct {
	db store
}

func (m *ExcludedChannelDB) GetSelected(guildID string) ([]string, error) {
	return m.db.GetExcludedChannelsByGuildId(guildID)
}

func (m *ExcludedChannelDB) RemoveByGuild(guildID string) error {
	return m.db.RemoveExcludedChannelsByGuild(guildID)
}

func (m *ExcludedChannelDB) Add(guildID, itemID string) error {
	return m.db.AddExcludedChannel(guildID, itemID)
}

type ModRoleDB struct {
	db store
}

func (r *ModRoleDB) GetSelected(guildID string) ([]string, error) {
	return r.db.GetModerationRolesByGuildId(guildID)
}

func (r *ModRoleDB) RemoveByGuild(guildID string) error {
	return r.db.RemoveModerationRolesByGuild(guildID)
}

func (r *ModRoleDB) Add(guildID, itemID string) error {
	return r.db.AddModerationRole(guildID, itemID)
}

type AutomodSettingsDB struct {
	db store
}

func (a *AutomodSettingsDB) GetSelected(guildID string) ([]string, error) {
	settings, err := a.db.GetAutomoderationSettings(guildID)
	if err != nil {
		return nil, err
	}

	var enabled []string
	if settings.BannedWordsEnabled {
		enabled = append(enabled, "banned_words")
	}
	if settings.BannedWebsitesEnabled {
		enabled = append(enabled, "banned_websites")
	}
	if settings.SpamDetectionEnabled {
		enabled = append(enabled, "spam_detection")
	}
	return enabled, nil
}

func (a *AutomodSettingsDB) RemoveByGuild(guildID string) error {
	// Set all to disabled
	return a.db.SetAutomoderationSettings(guildID, &database.AutomoderationSettings{
		GuildID:               guildID,
		BannedWordsEnabled:    false,
		BannedWebsitesEnabled: false,
		SpamDetectionEnabled:  false,
	})
}

func (a *AutomodSettingsDB) Add(guildID, itemID string) error {
	// Get current settings
	settings, err := a.db.GetAutomoderationSettings(guildID)
	if err != nil {
		return err
	}

	// Enable the specified feature
	switch itemID {
	case "banned_words":
		settings.BannedWordsEnabled = true
	case "banned_websites":
		settings.BannedWebsitesEnabled = true
	case "spam_detection":
		settings.SpamDetectionEnabled = true
	}

	return a.db.SetAutomoderationSettings(guildID, settings)
}

func (d *Discord) sendErrorMessage(s discordClient, i *discordgo.Interaction, title, description string) {
	_, _ = s.FollowupMessageCreate(i, true, &discordgo.WebhookParams{
		Embeds: []*discordgo.MessageEmbed{{
			Title:       title,
			Description: description,
			Color:       model.Red.Int(),
			Timestamp:   time.Now().UTC().Format(time.RFC3339),
		}},
	})
}

func (d *Discord) getTextChannels(s discordClient, guildID string) ([]*discordgo.Channel, error) {
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

func (d *Discord) getRoles(s discordClient, guildID string) ([]*discordgo.Role, error) {
	roles, err := s.GuildRoles(guildID)
	if err != nil {
		return nil, err
	}

	// Sort roles by position
	sort.Slice(roles, func(i, j int) bool {
		return roles[i].Position > roles[j].Position
	})

	return roles, nil
}

// Helper functions to convert channels and roles to SelectionItems
func (d *Discord) getTextChannelsAsItems(s discordClient, guildID string) ([]SelectionItem, error) {
	channels, err := d.getTextChannels(s, guildID)
	if err != nil {
		return nil, err
	}

	items := make([]SelectionItem, len(channels))
	for i, ch := range channels {
		items[i] = ChannelItem{ch}
	}
	return items, nil
}

func (d *Discord) getRolesAsItems(s discordClient, guildID string) ([]SelectionItem, error) {
	roles, err := d.getRoles(s, guildID)
	if err != nil {
		return nil, err
	}

	items := make([]SelectionItem, len(roles))
	for i, r := range roles {
		items[i] = RoleItem{r}
	}
	return items, nil
}
