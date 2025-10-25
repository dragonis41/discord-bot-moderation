package database

import (
	"github.com/bwmarrin/discordgo"
)

type HelperInterface interface {
	CheckAdminPermissionOnInteraction(s *discordgo.Session, i *discordgo.InteractionCreate) bool
	UserHasModerationRoleOnMessageCreate(s *discordgo.Session, m *discordgo.MessageCreate) bool
	UserHasModerationRoleOnMessageUpdate(s *discordgo.Session, m *discordgo.MessageUpdate) bool
	UserHasRoleByIdsOnInteraction(i *discordgo.InteractionCreate, roleIds []string) bool
	UserHasRoleByNameOnInteraction(s *discordgo.Session, i *discordgo.InteractionCreate, roleNames []string) bool
}

// CheckAdminPermissionOnInteraction checks if the user has admin permissions based on roles set in the database
func (d *Database) CheckAdminPermissionOnInteraction(s *discordgo.Session, i *discordgo.InteractionCreate) bool {
	adminRoleIds, err := d.GetModerationRolesByGuildId(i.GuildID)
	if err != nil {
		_, _ = s.FollowupMessageCreate(i.Interaction, true, &discordgo.WebhookParams{
			Embeds: []*discordgo.MessageEmbed{{
				Description: "❌ Une erreur est survenue lors de la vérification des permissions. Contactez un modérateur.",
				Color:       0xff0000,
			}},
		})
		return false
	}
	// If no admin roles are set, allow by default avoiding locking out all commands
	if len(adminRoleIds) == 0 {
		return true
	}
	if !d.UserHasRoleByIdsOnInteraction(i, adminRoleIds) {
		_, _ = s.FollowupMessageCreate(i.Interaction, true, &discordgo.WebhookParams{
			Embeds: []*discordgo.MessageEmbed{{
				Description: "❌ Vous n'avez pas la permission d'utiliser cette commande.",
				Color:       0xff0000,
			}},
		})
		return false
	}
	return true
}

// UserHasModerationRoleOnMessageCreate checks if the author of the message has any moderation roles
func (d *Database) UserHasModerationRoleOnMessageCreate(s *discordgo.Session, m *discordgo.MessageCreate) bool {
	modRoles, err := d.GetModerationRolesByGuildId(m.GuildID)
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

	member, err := s.GuildMember(m.GuildID, m.Author.ID)
	if err != nil {
		return false
	}

	// O(1) lookup for each member role
	for _, memberRoleID := range member.Roles {
		if modRolesMap[memberRoleID] {
			return true
		}
	}

	return false
}

// UserHasModerationRoleOnMessageUpdate checks if the author of the message has any moderation roles
func (d *Database) UserHasModerationRoleOnMessageUpdate(s *discordgo.Session, m *discordgo.MessageUpdate) bool {
	modRoles, err := d.GetModerationRolesByGuildId(m.GuildID)
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

	member, err := s.GuildMember(m.GuildID, m.Author.ID)
	if err != nil {
		return false
	}

	// O(1) lookup for each member role
	for _, memberRoleID := range member.Roles {
		if modRolesMap[memberRoleID] {
			return true
		}
	}

	return false
}

// UserHasRoleByIdsOnInteraction checks if a member has any of the specified role IDs
func (d *Database) UserHasRoleByIdsOnInteraction(i *discordgo.InteractionCreate, roleIds []string) bool {
	// Create a map for O(1) lookup
	roleMap := make(map[string]bool)
	for _, role := range roleIds {
		roleMap[role] = true
	}

	// Check if any of the member's roles exist in the roleMap
	for _, memberRoleID := range i.Member.Roles {
		if roleMap[memberRoleID] {
			return true
		}
	}

	return false
}

// UserHasRoleByNameOnInteraction checks if a member has any of the specified roles by name
// This function requires the guild to be passed to resolve role names to IDs
func (d *Database) UserHasRoleByNameOnInteraction(s *discordgo.Session, i *discordgo.InteractionCreate, roleNames []string) bool {
	// Get all roles for the guild
	roles, err := s.GuildRoles(i.GuildID)
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
	return d.UserHasRoleByIdsOnInteraction(i, roleIDs)
}
