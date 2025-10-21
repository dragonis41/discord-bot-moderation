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

func (d *Discord) sendRoleSelectPage(s *discordgo.Session, interaction *discordgo.Interaction, roles []*discordgo.Role, page int) {
	embed, components := d.buildRoleSelectMessage(interaction.GuildID, roles, page)

	if interaction.Message != nil {
		// Update existing message
		if _, err := s.FollowupMessageEdit(interaction, interaction.Message.ID, &discordgo.WebhookEdit{
			Embeds:     &[]*discordgo.MessageEmbed{embed},
			Components: &components,
		}); err != nil {
			d.log.LogError(logger.LogModel{Database: d.db, GuildID: interaction.GuildID, Function: "sendRoleSelectPage()",
				Message: fmt.Sprintf("Error updating follow-up message: %s", err),
			})
		}
	} else {
		// Create new message
		if _, err := s.FollowupMessageCreate(interaction, true, &discordgo.WebhookParams{
			Embeds:     []*discordgo.MessageEmbed{embed},
			Components: components,
		}); err != nil {
			d.log.LogError(logger.LogModel{Database: d.db, GuildID: interaction.GuildID, Function: "sendRoleSelectPage()",
				Message: fmt.Sprintf("Error sending follow-up message: %s", err),
			})
		}
	}
}

func (d *Discord) buildRoleSelectMessage(guildID string, roles []*discordgo.Role, page int) (*discordgo.MessageEmbed, []discordgo.MessageComponent) {
	totalPages := (len(roles) + rolesPerPage - 1) / rolesPerPage
	page = max(0, min(page, totalPages-1))

	start := page * rolesPerPage
	end := min(start+rolesPerPage, len(roles))

	// Build select menu options
	previouslySelected, err := d.db.GetModerationRolesByGuildId(guildID)
	if err != nil {
		d.log.LogError(logger.LogModel{Database: d.db, GuildID: guildID, Function: "buildRoleSelectMessage()",
			Message: fmt.Sprintf("Error fetching selected roles: %s", err),
		})
	}
	options := d.buildSelectRoleMenuOptions(roles[start:end], previouslySelected)

	// Create components
	minVal := 0
	components := []discordgo.MessageComponent{
		discordgo.ActionsRow{
			Components: []discordgo.MessageComponent{
				discordgo.SelectMenu{
					CustomID:    fmt.Sprintf("role_select_menu_%d", page),
					Placeholder: "Sélectionnez les roles de modération",
					MinValues:   &minVal,
					MaxValues:   len(options),
					Options:     options,
				},
			},
		},
	}

	// Add navigation buttons
	buttons := d.buildRoleNavigationButtons(page, totalPages)
	if len(buttons) > 0 {
		components = append(components, discordgo.ActionsRow{Components: buttons})
	}

	embed := &discordgo.MessageEmbed{
		Title: "Sélection des roles",
		Description: fmt.Sprintf("Sélectionnez les roles de modération puis cliquez sur \"Terminer\".\n"+
			"Ce sont les roles qui sont administrateurs du serveur et qui seront notifiés.\n\n"+
			"⚠️ Attention, si vous ne possédez pas au moins un de ces rôles, vous ne pourrez plus utiliser les commandes d'administration !\n\n"+
			"**%d**/**%d** Roles sélectionnés.", len(previouslySelected), len(roles)),
		Color:     model.Blue.Int(),
		Footer:    model.SelectionMenuFooter,
		Timestamp: time.Now().Format(time.RFC3339),
	}

	return embed, components
}

func (d *Discord) buildSelectRoleMenuOptions(roles []*discordgo.Role, selectedIDs []string) []discordgo.SelectMenuOption {
	selectedMap := make(map[string]bool)
	for _, id := range selectedIDs {
		selectedMap[id] = true
	}

	options := make([]discordgo.SelectMenuOption, len(roles))
	for i, role := range roles {
		options[i] = discordgo.SelectMenuOption{
			Label:   role.Name,
			Value:   role.ID,
			Default: selectedMap[role.ID],
		}
	}
	return options
}

func (d *Discord) buildRoleNavigationButtons(page, totalPages int) []discordgo.MessageComponent {
	var buttons []discordgo.MessageComponent

	if totalPages > 1 {
		buttons = append(buttons,
			discordgo.Button{
				Label:    "◀️ Précédent",
				Style:    discordgo.PrimaryButton,
				CustomID: fmt.Sprintf("role_page_prev_%d", page),
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
				CustomID: fmt.Sprintf("role_page_next_%d", page),
				Disabled: page == totalPages-1,
			},
		)
	}

	// Always add the done button
	buttons = append(buttons, discordgo.Button{
		Label:    "✅ Terminer",
		Style:    discordgo.SuccessButton,
		CustomID: "role_select_done",
	})

	return buttons
}

func (d *Discord) handleRoleSelection(s *discordgo.Session, i *discordgo.InteractionCreate) {
	// Check if this is a message component interaction
	if i.Type != discordgo.InteractionMessageComponent {
		return
	}

	data := i.MessageComponentData()
	customID := data.CustomID

	if !strings.HasPrefix(customID, "role_select") && !strings.HasPrefix(customID, "role_page") {
		return
	}

	// Respond immediately for all interactions
	if err := s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseUpdateMessage,
		Data: &discordgo.InteractionResponseData{Content: "Mise à jour..."},
	}); err != nil {
		d.log.LogError(logger.LogModel{Database: d.db, GuildID: i.GuildID, Function: "handleRoleSelection()",
			Message: fmt.Sprintf("Error responding to interaction: %s", err),
		})
		return
	}

	switch {
	case strings.HasPrefix(customID, "role_select_menu_"):
		d.handleRoleSelectionUpdate(s, i, data.Values)
	case strings.HasPrefix(customID, "role_page_"):
		d.handleRolePageNavigation(s, i, customID)
	case customID == "role_select_done":
		d.handleRoleSelectionDone(s, i)
	}
}

func (d *Discord) handleRoleSelectionUpdate(s *discordgo.Session, i *discordgo.InteractionCreate, selectedOnPage []string) {
	var page int
	_, _ = fmt.Sscanf(i.MessageComponentData().CustomID, "role_select_menu_%d", &page)

	roles, err := d.getRoles(s, i.GuildID)
	if err != nil {
		d.log.LogError(logger.LogModel{Database: d.db, GuildID: i.GuildID, Function: "handleRoleSelectionUpdate()",
			Message: fmt.Sprintf("Error fetching roles: %s", err),
		})
		return
	}

	// Update selections
	start := page * rolesPerPage
	end := min(start+rolesPerPage, len(roles))

	pageRoleIDs := make(map[string]bool)
	for _, ch := range roles[start:end] {
		pageRoleIDs[ch.ID] = true
	}

	// Merge selections
	newSelections := make(map[string]bool)
	selectedRoles, err := d.db.GetModerationRolesByGuildId(i.GuildID)
	if err != nil {
		d.log.LogError(logger.LogModel{Database: d.db, GuildID: i.GuildID, Function: "handleRoleSelectionUpdate()",
			Message: fmt.Sprintf("Error fetching selected roles: %s", err),
		})
	}
	for _, id := range selectedRoles {
		if !pageRoleIDs[id] {
			newSelections[id] = true
		}
	}
	for _, id := range selectedOnPage {
		newSelections[id] = true
	}

	// Clear the database and re-add selections
	err = d.db.RemoveModerationRolesByGuild(i.GuildID)
	if err != nil {
		d.log.LogError(logger.LogModel{Database: d.db, GuildID: i.GuildID, Function: "handleRoleSelectionUpdate()",
			Message: fmt.Sprintf("Error clearing selected roles: %s", err),
		})
	}
	for id := range newSelections {
		err := d.db.AddModerationRole(i.GuildID, id)
		if err != nil {
			d.log.LogError(logger.LogModel{Database: d.db, GuildID: i.GuildID, Function: "handleRoleSelectionUpdate()",
				Message: fmt.Sprintf("Error adding selected role [%s]: %s", id, err),
			})
		}
	}

	d.editRoleSelectMessage(s, i, roles, page)
}

func (d *Discord) handleRolePageNavigation(s *discordgo.Session, i *discordgo.InteractionCreate, customID string) {
	var currentPage int

	if strings.Contains(customID, "prev") {
		_, _ = fmt.Sscanf(customID, "role_page_prev_%d", &currentPage)
		currentPage--
	} else {
		_, _ = fmt.Sscanf(customID, "role_page_next_%d", &currentPage)
		currentPage++
	}

	roles, _ := d.getRoles(s, i.GuildID)
	d.editRoleSelectMessage(s, i, roles, currentPage)
}

func (d *Discord) handleRoleSelectionDone(s *discordgo.Session, i *discordgo.InteractionCreate) {
	selectedIDs, err := d.db.GetModerationRolesByGuildId(i.GuildID)
	if err != nil {
		d.log.LogError(logger.LogModel{Database: d.db, GuildID: i.GuildID, Function: "handleRoleSelectionDone()",
			Message: fmt.Sprintf("Error fetching selected roles: %s", err),
		})
	}
	var roleNames []string

	// Get all roles for the guild to ensure we have the latest data
	allRoles, err := s.GuildRoles(i.GuildID)
	if err != nil {
		d.log.LogError(logger.LogModel{Database: d.db, GuildID: i.GuildID, Function: "handleRoleSelectionDone()",
			Message: fmt.Sprintf("Error fetching guild roles: %s", err),
		})
	} else {
		// Create a map for quick lookup
		roleMap := make(map[string]*discordgo.Role)
		for _, ch := range allRoles {
			roleMap[ch.ID] = ch
		}

		// Build the list of selected role names
		for _, roleID := range selectedIDs {
			if role, exists := roleMap[roleID]; exists {
				roleNames = append(roleNames, fmt.Sprintf("- @%s", role.Name))
			}
		}
	}

	description := "⚠️ Aucun role sélectionné."
	if len(roleNames) > 0 {
		// Sort role names for consistent display
		sort.Strings(roleNames)
		description = fmt.Sprintf("✅ %d roles sélectionnés:\n%s", len(roleNames), strings.Join(roleNames, "\n"))
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

	d.log.LogInfo(logger.LogModel{Database: d.db, GuildID: i.GuildID, Function: "handleRoleSelectionDone()",
		Message: fmt.Sprintf("User [%s] finished selecting %d moderator roles for guild [%s]", i.Member.User.Username, len(selectedIDs), i.GuildID),
	})
}

func (d *Discord) editRoleSelectMessage(s *discordgo.Session, i *discordgo.InteractionCreate, roles []*discordgo.Role, page int) {
	embed, components := d.buildRoleSelectMessage(i.GuildID, roles, page)

	if _, err := s.InteractionResponseEdit(i.Interaction, &discordgo.WebhookEdit{
		Embeds:     &[]*discordgo.MessageEmbed{embed},
		Components: &components,
		Content:    nil,
	}); err != nil {
		d.log.LogError(logger.LogModel{Database: d.db, GuildID: i.GuildID, Function: "editRoleSelectMessage()",
			Message: fmt.Sprintf("Error editing message: %s", err),
		})
	}
}
