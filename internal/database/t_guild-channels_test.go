package database

import (
	"sort"
	"testing"
)

// TestGuildChannelsAreIsolatedByType verifies that the three channel kinds (log,
// moderation, excluded) share one table but never bleed into each other.
func TestGuildChannelsAreIsolatedByType(t *testing.T) {
	d := newTestDB(t)

	if err := d.AddLogChannel("g1", "c-log"); err != nil {
		t.Fatalf("AddLogChannel: %v", err)
	}
	if err := d.AddModerationChannel("g1", "c-mod"); err != nil {
		t.Fatalf("AddModerationChannel: %v", err)
	}
	if err := d.AddExcludedChannel("g1", "c-excl"); err != nil {
		t.Fatalf("AddExcludedChannel: %v", err)
	}

	logs, _ := d.GetLogChannelsByGuildId("g1")
	if len(logs) != 1 || logs[0] != "c-log" {
		t.Fatalf("log channels = %v, want [c-log]", logs)
	}
	mods, _ := d.GetModerationChannelsByGuildId("g1")
	if len(mods) != 1 || mods[0] != "c-mod" {
		t.Fatalf("mod channels = %v, want [c-mod]", mods)
	}
	excl, _ := d.GetExcludedChannelsByGuildId("g1")
	if len(excl) != 1 || excl[0] != "c-excl" {
		t.Fatalf("excluded channels = %v, want [c-excl]", excl)
	}

	// Membership checks must respect the type.
	if !d.IsLogChannel("g1", "c-log") {
		t.Error("IsLogChannel(c-log) = false, want true")
	}
	if d.IsLogChannel("g1", "c-mod") {
		t.Error("IsLogChannel(c-mod) = true, want false (wrong type)")
	}
	if !d.IsModerationChannel("g1", "c-mod") {
		t.Error("IsModerationChannel(c-mod) = false, want true")
	}
	if !d.IsExcludedChannel("g1", "c-excl") {
		t.Error("IsExcludedChannel(c-excl) = false, want true")
	}
	if d.IsExcludedChannel("g1", "unknown") {
		t.Error("IsExcludedChannel(unknown) = true, want false")
	}
}

func TestGuildChannelsAddIsIdempotent(t *testing.T) {
	d := newTestDB(t)

	for range 3 {
		if err := d.AddLogChannel("g1", "c1"); err != nil {
			t.Fatalf("AddLogChannel: %v", err)
		}
	}
	logs, _ := d.GetLogChannelsByGuildId("g1")
	if len(logs) != 1 {
		t.Fatalf("expected INSERT OR IGNORE to dedupe, got %d rows", len(logs))
	}
}

func TestGuildChannelsRemove(t *testing.T) {
	d := newTestDB(t)

	_ = d.AddLogChannel("g1", "a")
	_ = d.AddLogChannel("g1", "b")
	_ = d.AddLogChannel("g2", "c")

	if err := d.RemoveLogChannel("g1", "a"); err != nil {
		t.Fatalf("RemoveLogChannel: %v", err)
	}
	logs, _ := d.GetLogChannelsByGuildId("g1")
	if len(logs) != 1 || logs[0] != "b" {
		t.Fatalf("after RemoveLogChannel, got %v want [b]", logs)
	}

	if err := d.RemoveLogChannelsByGuild("g1"); err != nil {
		t.Fatalf("RemoveLogChannelsByGuild: %v", err)
	}
	logs, _ = d.GetLogChannelsByGuildId("g1")
	if len(logs) != 0 {
		t.Fatalf("expected no g1 log channels, got %v", logs)
	}

	// Other guild untouched.
	g2, _ := d.GetLogChannelsByGuildId("g2")
	sort.Strings(g2)
	if len(g2) != 1 || g2[0] != "c" {
		t.Fatalf("g2 channels = %v, want [c]", g2)
	}
}

func TestModerationChannelRemoval(t *testing.T) {
	d := newTestDB(t)

	_ = d.AddModerationChannel("g1", "a")
	_ = d.AddModerationChannel("g1", "b")

	if err := d.RemoveModerationChannel("g1", "a"); err != nil {
		t.Fatalf("RemoveModerationChannel: %v", err)
	}
	mods, _ := d.GetModerationChannelsByGuildId("g1")
	if len(mods) != 1 || mods[0] != "b" {
		t.Fatalf("after RemoveModerationChannel = %v, want [b]", mods)
	}

	if err := d.RemoveModerationChannelsByGuild("g1"); err != nil {
		t.Fatalf("RemoveModerationChannelsByGuild: %v", err)
	}
	mods, _ = d.GetModerationChannelsByGuildId("g1")
	if len(mods) != 0 {
		t.Fatalf("expected no mod channels, got %v", mods)
	}
}

func TestExcludedChannelRemoval(t *testing.T) {
	d := newTestDB(t)

	_ = d.AddExcludedChannel("g1", "a")
	_ = d.AddExcludedChannel("g1", "b")

	if err := d.RemoveExcludedChannel("g1", "a"); err != nil {
		t.Fatalf("RemoveExcludedChannel: %v", err)
	}
	excl, _ := d.GetExcludedChannelsByGuildId("g1")
	if len(excl) != 1 || excl[0] != "b" {
		t.Fatalf("after RemoveExcludedChannel = %v, want [b]", excl)
	}

	if err := d.RemoveExcludedChannelsByGuild("g1"); err != nil {
		t.Fatalf("RemoveExcludedChannelsByGuild: %v", err)
	}
	excl, _ = d.GetExcludedChannelsByGuildId("g1")
	if len(excl) != 0 {
		t.Fatalf("expected no excluded channels, got %v", excl)
	}
}
