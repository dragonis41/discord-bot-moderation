package database

import (
	"database/sql"
	"errors"
	"fmt"
)

type LogsConfigInterface interface {
	MigrateLogsConfigs() error
	GetMaxModerationLogEntries(guildID string) (int, error)
	SetMaxModerationLogEntries(guildID string, maxEntries int) error
	GetMaxSystemLogEntries(guildID string) (int, error)
	SetMaxSystemLogEntries(guildID string, maxEntries int) error
}

func (d *Database) MigrateLogsConfigs() error {
	// Create a table to store max log entries configuration per guild
	createConfigTableQuery := `
	CREATE TABLE IF NOT EXISTS logs_config (
		guild_id TEXT PRIMARY KEY,
		max_moderation_log_entries INTEGER DEFAULT 100,
		max_system_log_entries INTEGER DEFAULT 10000
	);
	`

	_, err := d.db.Exec(createConfigTableQuery)
	if err != nil {
		return fmt.Errorf("failed to create logs_config table: %w", err)
	}

	return nil
}

// GetMaxModerationLogEntries retrieves the max entries configuration for a guild
func (d *Database) GetMaxModerationLogEntries(guildID string) (int, error) {
	var maxEntries int
	query := `
	SELECT max_moderation_log_entries FROM logs_config
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
	INSERT INTO logs_config (guild_id, max_moderation_log_entries)
	VALUES (?, ?)
	ON CONFLICT(guild_id) DO UPDATE SET max_moderation_log_entries = excluded.max_moderation_log_entries;
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

// GetMaxSystemLogEntries retrieves the max entries configuration for a guild
func (d *Database) GetMaxSystemLogEntries(guildID string) (int, error) {
	var maxEntries int
	query := `
	SELECT max_system_log_entries FROM logs_config
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
	INSERT INTO logs_config (guild_id, max_system_log_entries)
	VALUES (?, ?)
	ON CONFLICT(guild_id) DO UPDATE SET max_system_log_entries = excluded.max_system_log_entries;
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
