package database

import (
	"github.com/bwmarrin/discordgo"
	"github.com/dragonis41/discord-bot-moderation/pkg/model"
)

type HelperInterface interface {
	CheckAdminPermissionOnInteraction(s *discordgo.Session, i *discordgo.InteractionCreate) bool
	UserHasModerationRoleOnMessageCreate(s *discordgo.Session, m *discordgo.MessageCreate) bool
	UserHasModerationRoleOnMessageUpdate(s *discordgo.Session, m *discordgo.MessageUpdate) bool
	UserHasModerationRoleOnInteraction(s *discordgo.Session, i *discordgo.InteractionCreate) bool
	UserHasRoleByIdsOnInteraction(i *discordgo.InteractionCreate, roleIds []string) bool
	UserHasRoleByNameOnInteraction(s *discordgo.Session, i *discordgo.InteractionCreate, roleNames []string) bool
}

// CheckAdminPermissionOnInteraction checks if the user has admin permissions based on roles set in the database
func (d *Database) CheckAdminPermissionOnInteraction(s *discordgo.Session, i *discordgo.InteractionCreate) bool {
	if !d.UserHasModerationRole(i.GuildID, i.Member) {
		_, _ = s.FollowupMessageCreate(i.Interaction, true, &discordgo.WebhookParams{
			Embeds: []*discordgo.MessageEmbed{{
				Description: "❌ Vous n'avez pas la permission d'utiliser cette commande.",
				Color:       model.Red.Int(),
			}},
		})
		return false
	}
	return true
}

// UserHasModerationRole checks if the user has any moderation roles
func (d *Database) UserHasModerationRole(guildID string, member *discordgo.Member) bool {
	modRoles, err := d.GetModerationRolesByGuildId(guildID)
	if err != nil {
		return false
	}
	if len(modRoles) == 0 {
		return false
	}

	// Convert modRoles slice to map for O(1) lookup
	modRolesMap := make(map[string]bool, len(modRoles))
	for _, roleID := range modRoles {
		modRolesMap[roleID] = true
	}

	// Check if any of the member's roles exist in the modRolesMap
	for _, memberRoleID := range member.Roles {
		if modRolesMap[memberRoleID] {
			return true
		}
	}

	return false
}

// UserHasRoleByIds checks if a member has any of the specified role IDs
func (d *Database) UserHasRoleByIds(m *discordgo.Member, roleIds []string) bool {
	// Create a map for O(1) lookup
	roleMap := make(map[string]bool)
	for _, role := range roleIds {
		roleMap[role] = true
	}

	// Check if any of the member's roles exist in the roleMap
	for _, memberRoleID := range m.Roles {
		if roleMap[memberRoleID] {
			return true
		}
	}

	return false
}

// UserHasRoleByName checks if a member has any of the specified roles by name
// This function requires the guild to be passed to resolve role names to IDs
func (d *Database) UserHasRoleByName(s *discordgo.Session, m *discordgo.Member, guildID string, roleNames []string) bool {
	// Get all roles for the guild
	roles, err := s.GuildRoles(guildID)
	if err != nil {
		return false
	}

	// Create a map of role names to role IDs
	roleNameToID := make(map[string]string)
	for _, role := range roles {
		roleNameToID[role.Name] = role.ID
	}

	// Convert role names to IDs
	roleIDs := make([]string, 0, len(roleNames))
	for _, roleName := range roleNames {
		if roleID, exists := roleNameToID[roleName]; exists {
			roleIDs = append(roleIDs, roleID)
		}
	}

	// Use the original function with role IDs
	return d.UserHasRoleByIds(m, roleIDs)
}
