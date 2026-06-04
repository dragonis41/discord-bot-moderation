package database

import (
	"testing"

	"github.com/bwmarrin/discordgo"
)

func TestUserHasModerationRole(t *testing.T) {
	d := newTestDB(t)

	// No moderation roles configured for the guild: everyone is allowed.
	if !d.UserHasModerationRole("g1", &discordgo.Member{Roles: []string{"x"}}) {
		t.Error("with no mod roles configured, expected true")
	}

	// nil member is treated as a moderator (avoids accidental self-bans).
	if !d.UserHasModerationRole("g1", nil) {
		t.Error("nil member should be treated as moderator")
	}

	// Configure moderation roles.
	if err := d.AddModerationRole("g1", "mod-role"); err != nil {
		t.Fatalf("AddModerationRole: %v", err)
	}

	// Member holding the mod role is allowed.
	if !d.UserHasModerationRole("g1", &discordgo.Member{Roles: []string{"other", "mod-role"}}) {
		t.Error("member with mod-role should be allowed")
	}

	// Member without any mod role is denied.
	if d.UserHasModerationRole("g1", &discordgo.Member{Roles: []string{"other"}}) {
		t.Error("member without mod-role should be denied")
	}

	// Member with no roles at all is denied once roles are configured.
	if d.UserHasModerationRole("g1", &discordgo.Member{Roles: nil}) {
		t.Error("member with no roles should be denied")
	}
}
