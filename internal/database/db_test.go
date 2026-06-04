package database

import (
	"database/sql"
	"path/filepath"
	"testing"
)

// newTestDB returns a *Database backed by a throwaway SQLite file in a temp
// directory, with all migrations applied. SQLite needs no server, so these are
// real database tests that exercise the actual SQL. The file is removed
// automatically when the test ends.
func newTestDB(t *testing.T) *Database {
	t.Helper()

	dbPath := filepath.Join(t.TempDir(), "test.db")
	conn, err := sql.Open("sqlite3", dbPath)
	if err != nil {
		t.Fatalf("failed to open test database: %v", err)
	}
	t.Cleanup(func() {
		if err := conn.Close(); err != nil {
			t.Errorf("failed to close test database: %v", err)
		}
	})

	d := &Database{db: conn}
	if err := d.Migrate(); err != nil {
		t.Fatalf("failed to migrate test database: %v", err)
	}
	return d
}

func TestGetDBConnection(t *testing.T) {
	d := newTestDB(t)
	if d.GetDBConnection() == nil {
		t.Fatal("GetDBConnection() returned nil")
	}
}

func TestCloseDatabase(t *testing.T) {
	// Use a standalone connection (not the helper's, which registers its own
	// cleanup-close) so we can assert Close succeeds exactly once.
	conn, err := sql.Open("sqlite3", filepath.Join(t.TempDir(), "close.db"))
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	d := &Database{db: conn}
	if err := d.CloseDatabase(); err != nil {
		t.Fatalf("CloseDatabase() returned error: %v", err)
	}
}

func TestMigrateIsIdempotent(t *testing.T) {
	d := newTestDB(t)

	// Migrate again on an already-migrated database: the ALTER TABLE that adds
	// the username column must be tolerated on the second run.
	if err := d.Migrate(); err != nil {
		t.Fatalf("second Migrate() returned error: %v", err)
	}
}
