package database

import (
	"testing"

	"github.com/dragonis41/discord-bot-moderation/pkg/model"
)

func TestModerationLogsAddAndQuery(t *testing.T) {
	d := newTestDB(t)

	if err := d.AddModerationLogEntry("g1", model.ActionBan, "u1", "alice", "spam_detection", "spamming"); err != nil {
		t.Fatalf("AddModerationLogEntry: %v", err)
	}
	if err := d.AddModerationLogEntry("g1", model.ActionDeleteMessage, "u2", "bob", "banned_word_check", "bad word"); err != nil {
		t.Fatalf("AddModerationLogEntry: %v", err)
	}
	if err := d.AddModerationLogEntry("g2", model.ActionBan, "u3", "carol", "spam_detection", "other guild"); err != nil {
		t.Fatalf("AddModerationLogEntry: %v", err)
	}

	count, err := d.GetModerationLogEntriesCount("g1")
	if err != nil {
		t.Fatalf("GetModerationLogEntriesCount: %v", err)
	}
	if count != 2 {
		t.Fatalf("g1 count = %d, want 2", count)
	}

	entries, err := d.GetModerationLogEntriesByGuild("g1", 10)
	if err != nil {
		t.Fatalf("GetModerationLogEntriesByGuild: %v", err)
	}
	if len(entries) != 2 {
		t.Fatalf("g1 entries = %d, want 2", len(entries))
	}
	// Username is persisted (added in a later migration) and round-trips.
	for _, e := range entries {
		if e.Username == "" {
			t.Errorf("entry %d has empty username", e.ID)
		}
	}

	// Filtering by action.
	bans, err := d.GetModerationLogEntriesByAction("g1", model.ActionBan, 10)
	if err != nil {
		t.Fatalf("GetModerationLogEntriesByAction: %v", err)
	}
	if len(bans) != 1 {
		t.Fatalf("g1 BAN entries = %d, want 1", len(bans))
	}
	if bans[0].UserID != "u1" || bans[0].Action != string(model.ActionBan) {
		t.Errorf("unexpected ban entry: %+v", bans[0])
	}
}

// TestModerationLogsRetentionCleanup verifies that adding entries beyond the
// configured max trims the oldest ones.
func TestModerationLogsRetentionCleanup(t *testing.T) {
	d := newTestDB(t)

	if err := d.SetMaxModerationLogEntries("g1", 3); err != nil {
		t.Fatalf("SetMaxModerationLogEntries: %v", err)
	}

	for i := range 6 {
		if err := d.AddModerationLogEntry("g1", model.ActionWarn, "u", "user", "trigger", string(rune('a'+i))); err != nil {
			t.Fatalf("AddModerationLogEntry: %v", err)
		}
	}

	count, _ := d.GetModerationLogEntriesCount("g1")
	if count != 3 {
		t.Fatalf("after retention cleanup count = %d, want 3", count)
	}
}

// TestSetMaxModerationLogEntriesTrimsImmediately verifies that lowering the limit
// prunes existing rows right away.
func TestSetMaxModerationLogEntriesTrimsImmediately(t *testing.T) {
	d := newTestDB(t)

	if err := d.SetMaxModerationLogEntries("g1", 1000); err != nil {
		t.Fatalf("SetMaxModerationLogEntries: %v", err)
	}
	for range 5 {
		if err := d.AddModerationLogEntry("g1", model.ActionWarn, "u", "user", "trigger", "x"); err != nil {
			t.Fatalf("AddModerationLogEntry: %v", err)
		}
	}

	if err := d.SetMaxModerationLogEntries("g1", 2); err != nil {
		t.Fatalf("SetMaxModerationLogEntries (lower): %v", err)
	}
	count, _ := d.GetModerationLogEntriesCount("g1")
	if count != 2 {
		t.Fatalf("after lowering limit count = %d, want 2", count)
	}
}
