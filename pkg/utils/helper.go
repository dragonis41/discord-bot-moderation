package utils

import "github.com/bwmarrin/discordgo"

// UserHasRoleByIds checks if a member has any of the specified role IDs
func UserHasRoleByIds(i *discordgo.InteractionCreate, roleIds []string) bool {
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
func UserHasRoleByName(s *discordgo.Session, i *discordgo.InteractionCreate, roleNames []string) bool {
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
	return UserHasRoleByIds(i, roleIDs)
}

func CheckAdminPermission(s *discordgo.Session, i *discordgo.InteractionCreate) bool {
	// TODO : Get roles from the database
	adminRoleNames := []string{"sudoers"}
	if !UserHasRoleByName(s, i, adminRoleNames) {
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
