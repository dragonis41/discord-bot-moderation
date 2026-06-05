package database

import (
	"database/sql"
	"errors"
	"fmt"
)

// defaultLanguage is returned when a guild has not chosen a language yet. It
// must match i18n.Default (English).
const defaultLanguage = "en"

type GuildSettingsInterface interface {
	MigrateGuildSettings() error
	GetGuildLanguage(guildID string) (string, error)
	SetGuildLanguage(guildID, language string) error
}

func (d *Database) MigrateGuildSettings() error {
	// Create a table to store per-guild settings (currently just the language).
	createTableQuery := `
	CREATE TABLE IF NOT EXISTS guild_settings (
		guild_id TEXT PRIMARY KEY,
		language TEXT NOT NULL DEFAULT 'en'
	);
	`

	_, err := d.db.Exec(createTableQuery)
	if err != nil {
		return fmt.Errorf("failed to create guild_settings table: %w", err)
	}

	return nil
}

// GetGuildLanguage returns the configured language code for a guild, or the
// default language when no row exists yet.
func (d *Database) GetGuildLanguage(guildID string) (string, error) {
	var language string
	query := `
	SELECT language FROM guild_settings
	WHERE guild_id = ?;
	`

	err := d.db.QueryRow(query, guildID).Scan(&language)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return defaultLanguage, nil
		}
		return defaultLanguage, fmt.Errorf("failed to get guild language: %w", err)
	}

	return language, nil
}

// SetGuildLanguage stores (inserts or updates) the language code for a guild.
func (d *Database) SetGuildLanguage(guildID, language string) error {
	query := `
	INSERT INTO guild_settings (guild_id, language)
	VALUES (?, ?)
	ON CONFLICT(guild_id) DO UPDATE SET language = excluded.language;
	`

	_, err := d.db.Exec(query, guildID, language)
	if err != nil {
		return fmt.Errorf("failed to set guild language: %w", err)
	}

	return nil
}
