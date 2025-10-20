package database

import "github.com/bwmarrin/discordgo"

type HelperInterface interface {
	CheckAdminPermission(s *discordgo.Session, i *discordgo.InteractionCreate) bool
	UserHasRoleByIds(i *discordgo.InteractionCreate, roleIds []string) bool
	UserHasRoleByName(s *discordgo.Session, i *discordgo.InteractionCreate, roleNames []string) bool
}

func (d *Database) CheckAdminPermission(s *discordgo.Session, i *discordgo.InteractionCreate) bool {
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
	if !d.UserHasRoleByIds(i, adminRoleIds) {
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

// UserHasRoleByIds checks if a member has any of the specified role IDs
func (d *Database) UserHasRoleByIds(i *discordgo.InteractionCreate, roleIds []string) bool {
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

// UserHasRoleByName checks if a member has any of the specified roles by name
// This function requires the guild to be passed to resolve role names to IDs
func (d *Database) UserHasRoleByName(s *discordgo.Session, i *discordgo.InteractionCreate, roleNames []string) bool {
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
	return d.UserHasRoleByIds(i, roleIDs)
}
