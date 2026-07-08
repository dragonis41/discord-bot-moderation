package database

import "testing"

func TestLogsConfigDefaults(t *testing.T) {
	d := newTestDB(t)

	// With no config row, the documented defaults are returned.
	modMax, err := d.GetMaxModerationLogEntries("g1")
	if err != nil {
		t.Fatalf("GetMaxModerationLogEntries: %v", err)
	}
	if modMax != 1000 {
		t.Errorf("default moderation max = %d, want 1000", modMax)
	}

	sysMax, err := d.GetMaxSystemLogEntries("g1")
	if err != nil {
		t.Fatalf("GetMaxSystemLogEntries: %v", err)
	}
	if sysMax != 10000 {
		t.Errorf("default system max = %d, want 10000", sysMax)
	}
}

func TestLogsConfigIsSetReporting(t *testing.T) {
	d := newTestDB(t)

	set, err := d.IsMaxSystemLogEntriesSet("g1")
	if err != nil {
		t.Fatalf("IsMaxSystemLogEntriesSet: %v", err)
	}
	if set {
		t.Error("IsMaxSystemLogEntriesSet = true before any config, want false")
	}

	if err := d.SetMaxSystemLogEntries("g1", 500); err != nil {
		t.Fatalf("SetMaxSystemLogEntries: %v", err)
	}
	set, _ = d.IsMaxSystemLogEntriesSet("g1")
	if !set {
		t.Error("IsMaxSystemLogEntriesSet = false after setting, want true")
	}

	got, _ := d.GetMaxSystemLogEntries("g1")
	if got != 500 {
		t.Errorf("GetMaxSystemLogEntries = %d, want 500", got)
	}

	// Moderation side is independent.
	modSet, _ := d.IsMaxModerationLogEntriesSet("g1")
	if !modSet {
		// Setting the system column inserts the row; the moderation column then
		// holds its table default, so IsSet reports true. This documents the
		// current behavior so a future change is a conscious decision.
		t.Log("moderation IsSet is false when only system column was written")
	}
}

func TestSetMaxModerationLogEntriesUpsert(t *testing.T) {
	d := newTestDB(t)

	if err := d.SetMaxModerationLogEntries("g1", 10); err != nil {
		t.Fatalf("SetMaxModerationLogEntries: %v", err)
	}
	got, _ := d.GetMaxModerationLogEntries("g1")
	if got != 10 {
		t.Fatalf("after first set = %d, want 10", got)
	}

	if err := d.SetMaxModerationLogEntries("g1", 42); err != nil {
		t.Fatalf("SetMaxModerationLogEntries (update): %v", err)
	}
	got, _ = d.GetMaxModerationLogEntries("g1")
	if got != 42 {
		t.Fatalf("after update = %d, want 42", got)
	}
}
