package database

import (
	"fmt"

	"github.com/dragonis41/discord-bot-moderation/pkg/model"
)

type ModerationLogsInterface interface {
	MigrateModerationLogs() error
	AddModerationLogEntry(guildID string, action model.ModerationLogAction, userID string, trigger string, reason string) error
	GetModerationLogEntriesByGuild(guildID string, limit int) ([]ModerationLogEntry, error)
	GetModerationLogEntriesByAction(guildID string, action model.ModerationLogAction, limit int) ([]ModerationLogEntry, error)
	GetModerationLogEntriesCount(guildID string) (int, error)
}

// ModerationLogEntry represents a single moderation log entry
type ModerationLogEntry struct {
	ID        int
	GuildID   string
	Action    string
	UserID    string
	Trigger   string
	Reason    string
	CreatedAt string
}

func (d *Database) MigrateModerationLogs() error {
	createTableQuery := `
	CREATE TABLE IF NOT EXISTS moderation_logs (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		guild_id TEXT NOT NULL,
		action TEXT NOT NULL,
		user_id text NOT NULL,
		trigger TEXT NOT NULL,
		reason TEXT NOT NULL,
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP
	);
	`

	_, err := d.db.Exec(createTableQuery)
	if err != nil {
		return fmt.Errorf("failed to create moderation_logs table: %w", err)
	}

	// Create index for better performance on guild_id queries
	createIndexQuery := `
	CREATE INDEX IF NOT EXISTS idx_moderation_logs_guild_id_created_at
	ON moderation_logs(guild_id, created_at DESC);
	`

	_, err = d.db.Exec(createIndexQuery)
	if err != nil {
		return fmt.Errorf("failed to create moderation_logs index: %w", err)
	}

	return nil
}

func (d *Database) AddModerationLogEntry(guildID string, action model.ModerationLogAction, userID string, trigger string, reason string) error {
	// First, get the max entries limit for this guild
	maxEntries, err := d.GetMaxModerationLogEntries(guildID)
	if err != nil {
		return fmt.Errorf("failed to get max guild log entries: %w", err)
	}

	// Insert the new log entry
	insertQuery := `
	INSERT INTO moderation_logs (guild_id, action, user_id, trigger, reason)
	VALUES (?, ?, ?, ?, ?);
	`

	_, err = d.db.Exec(insertQuery, guildID, action, userID, trigger, reason)
	if err != nil {
		return fmt.Errorf("failed to add log entry: %w", err)
	}

	// Clean up old entries if we exceed the limit
	if err := d.cleanupOldModerationLogEntries(guildID, maxEntries); err != nil {
		// Log the error but don't fail the operation
		return fmt.Errorf("log entry added but cleanup failed: %w", err)
	}

	return nil
}

// cleanupOldModerationLogEntries removes old log entries when the count exceeds maxEntries
func (d *Database) cleanupOldModerationLogEntries(guildID string, maxEntries int) error {
	// Delete entries that are beyond the maxEntries limit
	// This query keeps the most recent maxEntries and deletes the rest
	deleteQuery := `
	DELETE FROM moderation_logs
	WHERE guild_id = ?
	AND id NOT IN (
		SELECT id FROM moderation_logs
		WHERE guild_id = ?
		ORDER BY created_at DESC, id DESC
		LIMIT ?
	);
	`

	_, err := d.db.Exec(deleteQuery, guildID, guildID, maxEntries)
	if err != nil {
		return fmt.Errorf("failed to cleanup old guild log entries: %w", err)
	}

	return nil
}

// GetModerationLogEntriesByGuild retrieves log entries for a specific guild
func (d *Database) GetModerationLogEntriesByGuild(guildID string, limit int) ([]ModerationLogEntry, error) {
	query := `
	SELECT id, guild_id, action, user_id, trigger, reason, created_at
	FROM moderation_logs
	WHERE guild_id = ?
	ORDER BY created_at DESC, id DESC
	LIMIT ?;
	`

	rows, err := d.db.Query(query, guildID, limit)
	if err != nil {
		return nil, fmt.Errorf("failed to get guild log entries: %w", err)
	}
	defer rows.Close()

	var entries []ModerationLogEntry
	for rows.Next() {
		var entry ModerationLogEntry
		if err := rows.Scan(&entry.ID, &entry.GuildID, &entry.Action, &entry.UserID,
			&entry.Trigger, &entry.Reason, &entry.CreatedAt); err != nil {
			return nil, fmt.Errorf("failed to scan guild log entry: %w", err)
		}
		entries = append(entries, entry)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating over rows: %w", err)
	}

	return entries, nil
}

// GetModerationLogEntriesByAction retrieves log entries for a specific guild filtered by action type
func (d *Database) GetModerationLogEntriesByAction(guildID string, action model.ModerationLogAction, limit int) ([]ModerationLogEntry, error) {
	query := `
		SELECT id, guild_id, action, user_id, trigger, reason, created_at
		FROM moderation_logs
		WHERE guild_id = ? AND action = ?
		ORDER BY created_at DESC, id DESC
		LIMIT ?;
		`

	rows, err := d.db.Query(query, guildID, action, limit)
	if err != nil {
		return nil, fmt.Errorf("failed to get moderation log entries: %w", err)
	}
	defer rows.Close()

	var entries []ModerationLogEntry
	for rows.Next() {
		var entry ModerationLogEntry
		if err := rows.Scan(&entry.ID, &entry.GuildID, &entry.Action, &entry.UserID,
			&entry.Trigger, &entry.Reason, &entry.CreatedAt); err != nil {
			return nil, fmt.Errorf("failed to scan moderation log entry: %w", err)
		}
		entries = append(entries, entry)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating over rows: %w", err)
	}

	return entries, nil
}

// GetModerationLogEntriesCount returns the total number of log entries for a guild
func (d *Database) GetModerationLogEntriesCount(guildID string) (int, error) {
	var count int
	query := `
	SELECT COUNT(*) FROM moderation_logs
	WHERE guild_id = ?;
	`

	err := d.db.QueryRow(query, guildID).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("failed to get guild log entries count: %w", err)
	}

	return count, nil
}
