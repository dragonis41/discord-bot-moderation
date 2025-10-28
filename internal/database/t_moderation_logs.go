package database

import (
	"database/sql"
	"errors"
	"fmt"

	"github.com/dragonis41/discord-bot-moderation/pkg/model"
)

type ModerationLogsInterface interface {
	MigrateModerationLogs() error
	AddModerationLogEntry(guildID string, logType model.ModerationLogAction, trigger string, reason string) error
	SetMaxModerationLogEntries(guildID string, maxEntries int) error
	GetMaxModerationLogEntries(guildID string) (int, error)
	GetModerationLogEntriesByGuild(guildID string, limit int) ([]ModerationLogEntry, error)
	GetModerationLogEntriesByAction(guildID string, action model.ModerationLogAction, limit int) ([]ModerationLogEntry, error)
	GetModerationLogEntriesCount(guildID string) (int, error)
}

// ModerationLogEntry represents a single moderation log entry
type ModerationLogEntry struct {
	ID        int
	GuildID   string
	Action    string
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

	// Create a table to store max log entries configuration per guild
	createConfigTableQuery := `
	CREATE TABLE IF NOT EXISTS moderation_logs_config (
		guild_id TEXT PRIMARY KEY,
		max_entries INTEGER DEFAULT 100
	);
	`

	_, err = d.db.Exec(createConfigTableQuery)
	if err != nil {
		return fmt.Errorf("failed to create moderation_logs_config table: %w", err)
	}

	return nil
}

func (d *Database) AddModerationLogEntry(guildID string, logType model.ModerationLogAction, trigger string, reason string) error {
	// First, get the max entries limit for this guild
	maxEntries, err := d.getMaxModerationLogEntries(guildID)
	if err != nil {
		// If error or no config exists, use default of 100
		maxEntries = 100
	}

	// Insert the new log entry
	insertQuery := `
	INSERT INTO moderation_logs (guild_id, action, trigger, reason)
	VALUES (?, ?, ?, ?);
	`

	_, err = d.db.Exec(insertQuery, guildID, logType, trigger, reason)
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

// getMaxModerationLogEntries retrieves the max entries configuration for a guild
func (d *Database) getMaxModerationLogEntries(guildID string) (int, error) {
	var maxEntries int
	query := `
	SELECT max_entries FROM moderation_logs_config
	WHERE guild_id = ?;
	`

	err := d.db.QueryRow(query, guildID).Scan(&maxEntries)
	if err != nil {
		// If no config exists, return default
		if errors.Is(err, sql.ErrNoRows) {
			return 10000, nil
		}
		return 0, fmt.Errorf("failed to get max guild log entries: %w", err)
	}

	return maxEntries, nil
}

// SetMaxModerationLogEntries sets the maximum number of log entries for a guild
func (d *Database) SetMaxModerationLogEntries(guildID string, maxEntries int) error {
	// Insert or update the configuration
	query := `
	INSERT INTO moderation_logs_config (guild_id, max_entries)
	VALUES (?, ?)
	ON CONFLICT(guild_id) DO UPDATE SET max_entries = excluded.max_entries;
	`

	_, err := d.db.Exec(query, guildID, maxEntries)
	if err != nil {
		return fmt.Errorf("failed to set max guild log entries: %w", err)
	}

	// Immediately clean guild's logs if the new limit is lower than current count
	if err := d.cleanupOldModerationLogEntries(guildID, maxEntries); err != nil {
		return fmt.Errorf("failed to cleanup guild logs after setting new limit: %w", err)
	}

	return nil
}

// GetMaxModerationLogEntries retrieves the maximum number of log entries configured for a guild
func (d *Database) GetMaxModerationLogEntries(guildID string) (int, error) {
	return d.getMaxModerationLogEntries(guildID)
}

// GetModerationLogEntriesByGuild retrieves log entries for a specific guild
func (d *Database) GetModerationLogEntriesByGuild(guildID string, limit int) ([]ModerationLogEntry, error) {
	query := `
	SELECT id, guild_id, action, trigger, reason, created_at
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
		if err := rows.Scan(&entry.ID, &entry.GuildID, &entry.Action,
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
		SELECT id, guild_id, action, trigger, reason, created_at
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
		if err := rows.Scan(&entry.ID, &entry.GuildID, &entry.Action,
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
