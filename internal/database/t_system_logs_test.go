package database

import (
	"testing"

	"github.com/dragonis41/discord-bot-moderation/pkg/model"
)

func TestSystemLogsAddAndQuery(t *testing.T) {
	d := newTestDB(t)

	if err := d.AddSystemLogEntry("g1", model.TypeInfo, "fn1", "info message"); err != nil {
		t.Fatalf("AddSystemLogEntry: %v", err)
	}
	if err := d.AddSystemLogEntry("g1", model.TypeError, "fn2", "error message"); err != nil {
		t.Fatalf("AddSystemLogEntry: %v", err)
	}

	count, err := d.GetSystemLogEntriesCount("g1")
	if err != nil {
		t.Fatalf("GetSystemLogEntriesCount: %v", err)
	}
	if count != 2 {
		t.Fatalf("count = %d, want 2", count)
	}

	entries, err := d.GetSystemLogEntriesByGuild("g1", 10)
	if err != nil {
		t.Fatalf("GetSystemLogEntriesByGuild: %v", err)
	}
	if len(entries) != 2 {
		t.Fatalf("entries = %d, want 2", len(entries))
	}
}

// TestSystemLogsErrorsByGuildAndSystem checks the query that powers the status
// command: it returns ERROR rows for the guild plus system-wide (”) errors,
// and excludes non-error rows.
func TestSystemLogsErrorsByGuildAndSystem(t *testing.T) {
	d := newTestDB(t)

	_ = d.AddSystemLogEntry("g1", model.TypeError, "fn", "guild error")
	_ = d.AddSystemLogEntry("g1", model.TypeInfo, "fn", "guild info (excluded)")
	_ = d.AddSystemLogEntry("", model.TypeError, "fn", "system error")
	_ = d.AddSystemLogEntry("g2", model.TypeError, "fn", "other guild error (excluded)")

	errs, err := d.GetSystemLogEntriesErrorsByGuildAndSystem("g1", 10)
	if err != nil {
		t.Fatalf("GetSystemLogEntriesErrorsByGuildAndSystem: %v", err)
	}
	if len(errs) != 2 {
		t.Fatalf("got %d error rows, want 2 (guild + system)", len(errs))
	}
	for _, e := range errs {
		if e.LogType != string(model.TypeError) {
			t.Errorf("unexpected non-error row: %+v", e)
		}
		if e.GuildID != "g1" && e.GuildID != "" {
			t.Errorf("leaked row from guild %q", e.GuildID)
		}
	}
}

func TestSystemLogsRetentionCleanup(t *testing.T) {
	d := newTestDB(t)

	if err := d.SetMaxSystemLogEntries("g1", 3); err != nil {
		t.Fatalf("SetMaxSystemLogEntries: %v", err)
	}
	for range 7 {
		if err := d.AddSystemLogEntry("g1", model.TypeInfo, "fn", "x"); err != nil {
			t.Fatalf("AddSystemLogEntry: %v", err)
		}
	}

	count, _ := d.GetSystemLogEntriesCount("g1")
	if count != 3 {
		t.Fatalf("after retention cleanup count = %d, want 3", count)
	}
}
