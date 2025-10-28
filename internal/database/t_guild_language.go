package database

import (
	"database/sql"
	"errors"
	"fmt"
)

type GuildLanguageInterface interface {
	MigrateGuildLanguage() error
	GetGuildLanguage(guildID string) (string, error)
	SetGuildLanguage(guildID, language string) error
}

func (d *Database) MigrateGuildLanguage() error {
	createTableQuery := `
	CREATE TABLE IF NOT EXISTS guild_language (
		guild_id TEXT PRIMARY KEY,
		language TEXT DEFAULT 'en_US'
	);
	`

	_, err := d.db.Exec(createTableQuery)
	if err != nil {
		return fmt.Errorf("failed to create guild_language table: %w", err)
	}

	return nil
}

// GetGuildLanguage retrieves the language preference for a guild
func (d *Database) GetGuildLanguage(guildID string) (string, error) {
	var language string
	query := `
	SELECT language FROM guild_language
	WHERE guild_id = ?;
	`

	err := d.db.QueryRow(query, guildID).Scan(&language)
	if err != nil {
		// If no preference exists, return default (English)
		if errors.Is(err, sql.ErrNoRows) {
			return "en_US", nil
		}
		return "", fmt.Errorf("failed to get guild language: %w", err)
	}

	return language, nil
}

// SetGuildLanguage sets the language preference for a guild
func (d *Database) SetGuildLanguage(guildID, language string) error {
	query := `
	INSERT INTO guild_language (guild_id, language)
	VALUES (?, ?)
	ON CONFLICT(guild_id) DO UPDATE SET language = ?;
	`

	_, err := d.db.Exec(query, guildID, language, language)
	if err != nil {
		return fmt.Errorf("failed to set guild language: %w", err)
	}

	return nil
}
