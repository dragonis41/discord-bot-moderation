package database

import (
	"sort"
	"testing"
)

func TestModerationRolesCRUD(t *testing.T) {
	d := newTestDB(t)

	roles, err := d.GetModerationRolesByGuildId("g1")
	if err != nil {
		t.Fatalf("GetModerationRolesByGuildId: %v", err)
	}
	if len(roles) != 0 {
		t.Fatalf("expected no roles, got %d", len(roles))
	}

	if err := d.AddModerationRole("g1", "r1"); err != nil {
		t.Fatalf("AddModerationRole: %v", err)
	}
	if err := d.AddModerationRole("g1", "r2"); err != nil {
		t.Fatalf("AddModerationRole: %v", err)
	}
	// Duplicate is ignored by the UNIQUE constraint.
	if err := d.AddModerationRole("g1", "r1"); err != nil {
		t.Fatalf("AddModerationRole (dup): %v", err)
	}

	roles, _ = d.GetModerationRolesByGuildId("g1")
	sort.Strings(roles)
	if len(roles) != 2 || roles[0] != "r1" || roles[1] != "r2" {
		t.Fatalf("roles = %v, want [r1 r2]", roles)
	}

	if err := d.RemoveModerationRole("g1", "r1"); err != nil {
		t.Fatalf("RemoveModerationRole: %v", err)
	}
	roles, _ = d.GetModerationRolesByGuildId("g1")
	if len(roles) != 1 || roles[0] != "r2" {
		t.Fatalf("after removal roles = %v, want [r2]", roles)
	}

	if err := d.RemoveModerationRolesByGuild("g1"); err != nil {
		t.Fatalf("RemoveModerationRolesByGuild: %v", err)
	}
	roles, _ = d.GetModerationRolesByGuildId("g1")
	if len(roles) != 0 {
		t.Fatalf("expected no roles after RemoveModerationRolesByGuild, got %d", len(roles))
	}
}
