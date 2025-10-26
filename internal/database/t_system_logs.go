package database

import (
	"database/sql"
	"errors"
	"fmt"

	"github.com/dragonis41/discord-bot-moderation/pkg/model"
)

type SystemLogsInterface interface {
	AddSystemLogEntry(guildID string, logType model.LogType, function string, content string) error
	GetSystemLogEntriesByGuild(guildID string, limit int) ([]SystemLogEntry, error)
	GetSystemLogEntriesErrorsByGuildAndSystem(guildID string, limit int) ([]SystemLogEntry, error)
	GetSystemLogEntriesCount(guildID string) (int, error)
	SetMaxSystemLogEntries(guildID string, maxEntries int) error
	GetMaxSystemLogEntries(guildID string) (int, error)
}

// SystemLogEntry represents a single system log entry
type SystemLogEntry struct {
	ID        int
	GuildID   string
	LogType   string
	Function  string
	Content   string
	CreatedAt string
}

func (d *Database) MigrateSystemLogs() error {
	createTableQuery := `
	CREATE TABLE IF NOT EXISTS system_logs (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		guild_id TEXT NOT NULL,
		log_type TEXT NOT NULL,
		function TEXT NOT NULL,
		content TEXT NOT NULL,
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP
	);
	`

	_, err := d.db.Exec(createTableQuery)
	if err != nil {
		return fmt.Errorf("failed to create system_logs table: %w", err)
	}

	// Create index for better performance on guild_id queries
	createIndexQuery := `
	CREATE INDEX IF NOT EXISTS idx_system_logs_guild_id_created_at
	ON system_logs(guild_id, created_at DESC);
	`

	_, err = d.db.Exec(createIndexQuery)
	if err != nil {
		return fmt.Errorf("failed to create system_logs index: %w", err)
	}

	// Create a table to store max log entries configuration per guild
	createConfigTableQuery := `
	CREATE TABLE IF NOT EXISTS logs_config (
		guild_id TEXT PRIMARY KEY,
		max_entries INTEGER DEFAULT 10000
	);
	`

	_, err = d.db.Exec(createConfigTableQuery)
	if err != nil {
		return fmt.Errorf("failed to create logs_config table: %w", err)
	}

	return nil
}

func (d *Database) AddSystemLogEntry(guildID string, logType model.LogType, function string, content string) error {
	// First, get the max entries limit for this guild
	maxEntries, err := d.getMaxSystemLogEntries(guildID)
	if err != nil {
		// If error or no config exists, use default of 10000
		maxEntries = 10000
	}

	// Insert the new log entry
	insertQuery := `
	INSERT INTO system_logs (guild_id, log_type, function, content)
	VALUES (?, ?, ?, ?);
	`

	_, err = d.db.Exec(insertQuery, guildID, logType, function, content)
	if err != nil {
		return fmt.Errorf("failed to add log entry: %w", err)
	}

	// Clean up old entries if we exceed the limit
	if err := d.cleanupOldSystemLogEntries(guildID, maxEntries); err != nil {
		// Log the error but don't fail the operation
		return fmt.Errorf("log entry added but cleanup failed: %w", err)
	}

	return nil
}

// cleanupOldSystemLogEntries removes old log entries when the count exceeds maxEntries
func (d *Database) cleanupOldSystemLogEntries(guildID string, maxEntries int) error {
	// Delete entries that are beyond the maxEntries limit
	// This query keeps the most recent maxEntries and deletes the rest
	deleteQuery := `
	DELETE FROM system_logs
	WHERE guild_id = ?
	AND id NOT IN (
		SELECT id FROM system_logs
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

// getMaxSystemLogEntries retrieves the max entries configuration for a guild
func (d *Database) getMaxSystemLogEntries(guildID string) (int, error) {
	var maxEntries int
	query := `
	SELECT max_entries FROM logs_config
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

// SetMaxSystemLogEntries sets the maximum number of log entries for a guild
func (d *Database) SetMaxSystemLogEntries(guildID string, maxEntries int) error {
	// Insert or update the configuration
	query := `
	INSERT INTO logs_config (guild_id, max_entries)
	VALUES (?, ?)
	ON CONFLICT(guild_id) DO UPDATE SET max_entries = excluded.max_entries;
	`

	_, err := d.db.Exec(query, guildID, maxEntries)
	if err != nil {
		return fmt.Errorf("failed to set max guild log entries: %w", err)
	}

	// Immediately clean guild's logs if the new limit is lower than current count
	if err := d.cleanupOldSystemLogEntries(guildID, maxEntries); err != nil {
		return fmt.Errorf("failed to cleanup guild logs after setting new limit: %w", err)
	}

	// Also cleanup system logs
	if err := d.cleanupOldSystemLogEntries("", maxEntries); err != nil {
		return fmt.Errorf("failed to cleanup system logs after setting new limit: %w", err)
	}

	return nil
}

// GetMaxSystemLogEntries retrieves the maximum number of log entries configured for a guild
func (d *Database) GetMaxSystemLogEntries(guildID string) (int, error) {
	return d.getMaxSystemLogEntries(guildID)
}

// GetSystemLogEntriesByGuild retrieves log entries for a specific guild
func (d *Database) GetSystemLogEntriesByGuild(guildID string, limit int) ([]SystemLogEntry, error) {
	query := `
	SELECT id, guild_id, log_type, function, content, created_at
	FROM system_logs
	WHERE guild_id = ?
	ORDER BY created_at DESC, id DESC
	LIMIT ?;
	`

	rows, err := d.db.Query(query, guildID, limit)
	if err != nil {
		return nil, fmt.Errorf("failed to get guild log entries: %w", err)
	}
	defer rows.Close()

	var entries []SystemLogEntry
	for rows.Next() {
		var entry SystemLogEntry
		if err := rows.Scan(&entry.ID, &entry.GuildID, &entry.LogType,
			&entry.Function, &entry.Content, &entry.CreatedAt); err != nil {
			return nil, fmt.Errorf("failed to scan guild log entry: %w", err)
		}
		entries = append(entries, entry)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating over rows: %w", err)
	}

	return entries, nil
}

// GetSystemLogEntriesErrorsByGuildAndSystem retrieves log entries for a specific guild and system-wide error logs
func (d *Database) GetSystemLogEntriesErrorsByGuildAndSystem(guildID string, limit int) ([]SystemLogEntry, error) {
	query := `
		SELECT id, guild_id, log_type, function, content, created_at
		FROM system_logs
		WHERE (guild_id = ? OR guild_id = '') AND log_type = ?
		ORDER BY created_at DESC, id DESC
		LIMIT ?;
		`

	rows, err := d.db.Query(query, guildID, model.ErrorType, limit)
	if err != nil {
		return nil, fmt.Errorf("failed to get system log entries: %w", err)
	}
	defer rows.Close()

	var entries []SystemLogEntry
	for rows.Next() {
		var entry SystemLogEntry
		if err := rows.Scan(&entry.ID, &entry.GuildID, &entry.LogType,
			&entry.Function, &entry.Content, &entry.CreatedAt); err != nil {
			return nil, fmt.Errorf("failed to scan system log entry: %w", err)
		}
		entries = append(entries, entry)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating over rows: %w", err)
	}

	return entries, nil
}

// GetSystemLogEntriesCount returns the total number of log entries for a guild
func (d *Database) GetSystemLogEntriesCount(guildID string) (int, error) {
	var count int
	query := `
	SELECT COUNT(*) FROM system_logs
	WHERE guild_id = ?;
	`

	err := d.db.QueryRow(query, guildID).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("failed to get guild log entries count: %w", err)
	}

	return count, nil
}
