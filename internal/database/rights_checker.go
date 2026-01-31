package database

import (
	"github.com/bwmarrin/discordgo"
	"github.com/dragonis41/discord-bot-moderation/pkg/model"
)

type HelperInterface interface {
	CheckModerationPermissionOnInteraction(s *discordgo.Session, i *discordgo.InteractionCreate) bool
	UserHasModerationRole(guildID string, member *discordgo.Member) bool
}

// CheckModerationPermissionOnInteraction checks if the user has admin permissions based on roles set in the database
//
//	It sends a follow-up message if the user lacks permissions and returns false.
func (d *Database) CheckModerationPermissionOnInteraction(s *discordgo.Session, i *discordgo.InteractionCreate) bool {
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

// UserHasModerationRole checks if the user has any moderation roles set in the database
//
//	It does not respond to the user; it simply returns true or false.
func (d *Database) UserHasModerationRole(guildID string, member *discordgo.Member) bool {
	// If member is nil, we can't check roles - treat as moderator to prevent banning issues
	if member == nil {
		return true
	}

	modRoles, err := d.GetModerationRolesByGuildId(guildID)
	if err != nil {
		return false
	}
	// If no moderation roles are set, allow all users
	if len(modRoles) == 0 {
		return true
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
