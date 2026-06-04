package database

import (
	"database/sql"
	"errors"
	"fmt"
	"time"
)

type AutomoderationInterface interface {
	// Banned Words
	AddBannedWord(guildID, wordPattern string, isRegex bool) error
	RemoveBannedWord(guildID string, id int) error
	RemoveBannedWordsByGuild(guildID string) error
	GetBannedWordsByGuildId(guildID string) ([]BannedWord, error)

	// Banned Websites
	AddBannedWebsite(guildID, websiteURL string) error
	RemoveBannedWebsite(guildID string, id int) error
	RemoveBannedWebsitesByGuild(guildID string) error
	GetBannedWebsitesByGuildId(guildID string) ([]BannedWebsite, error)

	// Automoderation Settings
	GetAutomoderationSettings(guildID string) (*AutomoderationSettings, error)
	SetAutomoderationSettings(guildID string, settings *AutomoderationSettings) error
}

type BannedWord struct {
	ID          int
	GuildID     string
	WordPattern string
	IsRegex     bool
	CreatedAt   time.Time
}

type BannedWebsite struct {
	ID         int
	GuildID    string
	WebsiteURL string
	CreatedAt  time.Time
}

type AutomoderationSettings struct {
	GuildID               string
	BannedWordsEnabled    bool
	BannedWebsitesEnabled bool
	SpamDetectionEnabled  bool
}

func (d *Database) MigrateAutomoderation() error {
	// Create banned_words table
	createBannedWordsQuery := `
	CREATE TABLE IF NOT EXISTS banned_words (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		guild_id TEXT NOT NULL,
		word_pattern TEXT NOT NULL,
		is_regex BOOLEAN NOT NULL DEFAULT 0,
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP
	);
	`

	_, err := d.db.Exec(createBannedWordsQuery)
	if err != nil {
		return fmt.Errorf("failed to create banned_words table: %w", err)
	}

	// Create index for better performance on guild_id queries
	createBannedWordsIndexQuery := `
	CREATE INDEX IF NOT EXISTS idx_banned_words_guild_id
	ON banned_words(guild_id);
	`

	_, err = d.db.Exec(createBannedWordsIndexQuery)
	if err != nil {
		return fmt.Errorf("failed to create banned_words index: %w", err)
	}

	// Create banned_websites table
	createBannedWebsitesQuery := `
	CREATE TABLE IF NOT EXISTS banned_websites (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		guild_id TEXT NOT NULL,
		website_url TEXT NOT NULL,
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP
	);
	`

	_, err = d.db.Exec(createBannedWebsitesQuery)
	if err != nil {
		return fmt.Errorf("failed to create banned_websites table: %w", err)
	}

	// Create index for better performance on guild_id queries
	createBannedWebsitesIndexQuery := `
	CREATE INDEX IF NOT EXISTS idx_banned_websites_guild_id
	ON banned_websites(guild_id);
	`

	_, err = d.db.Exec(createBannedWebsitesIndexQuery)
	if err != nil {
		return fmt.Errorf("failed to create banned_websites index: %w", err)
	}

	// Create automoderation_settings table
	createSettingsQuery := `
	CREATE TABLE IF NOT EXISTS automoderation_settings (
		guild_id TEXT PRIMARY KEY,
		banned_words_enabled BOOLEAN NOT NULL DEFAULT 1,
		banned_websites_enabled BOOLEAN NOT NULL DEFAULT 1,
		spam_detection_enabled BOOLEAN NOT NULL DEFAULT 1
	);
	`

	_, err = d.db.Exec(createSettingsQuery)
	if err != nil {
		return fmt.Errorf("failed to create automoderation_settings table: %w", err)
	}

	return nil
}

// Banned Words Functions

func (d *Database) AddBannedWord(guildID, wordPattern string, isRegex bool) error {
	insertQuery := `
	INSERT INTO banned_words (guild_id, word_pattern, is_regex)
	VALUES (?, ?, ?);
	`

	_, err := d.db.Exec(insertQuery, guildID, wordPattern, isRegex)
	if err != nil {
		return fmt.Errorf("failed to add banned word: %w", err)
	}

	return nil
}

func (d *Database) RemoveBannedWord(guildID string, id int) error {
	deleteQuery := `
	DELETE FROM banned_words
	WHERE guild_id = ? AND id = ?;
	`

	_, err := d.db.Exec(deleteQuery, guildID, id)
	if err != nil {
		return fmt.Errorf("failed to remove banned word: %w", err)
	}

	return nil
}

func (d *Database) RemoveBannedWordsByGuild(guildID string) error {
	deleteQuery := `
	DELETE FROM banned_words
	WHERE guild_id = ?;
	`

	_, err := d.db.Exec(deleteQuery, guildID)
	if err != nil {
		return fmt.Errorf("failed to remove banned words by guild: %w", err)
	}

	return nil
}

func (d *Database) GetBannedWordsByGuildId(guildID string) ([]BannedWord, error) {
	selectQuery := `
	SELECT id, guild_id, word_pattern, is_regex, created_at
	FROM banned_words
	WHERE guild_id = ?
	ORDER BY created_at DESC;
	`

	rows, err := d.db.Query(selectQuery, guildID)
	if err != nil {
		return nil, fmt.Errorf("failed to get banned words: %w", err)
	}
	defer rows.Close()

	var bannedWords []BannedWord
	for rows.Next() {
		var word BannedWord
		var createdAt string
		if err := rows.Scan(&word.ID, &word.GuildID, &word.WordPattern, &word.IsRegex, &createdAt); err != nil {
			return nil, fmt.Errorf("failed to scan banned word: %w", err)
		}

		// Parse the timestamp
		word.CreatedAt, err = time.Parse("2006-01-02 15:04:05", createdAt)
		if err != nil {
			// If parsing fails, use current time as fallback
			word.CreatedAt = time.Now()
		}

		bannedWords = append(bannedWords, word)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating over rows: %w", err)
	}

	return bannedWords, nil
}

// Banned Websites Functions

func (d *Database) AddBannedWebsite(guildID, websiteURL string) error {
	insertQuery := `
	INSERT INTO banned_websites (guild_id, website_url)
	VALUES (?, ?);
	`

	_, err := d.db.Exec(insertQuery, guildID, websiteURL)
	if err != nil {
		return fmt.Errorf("failed to add banned website: %w", err)
	}

	return nil
}

func (d *Database) RemoveBannedWebsite(guildID string, id int) error {
	deleteQuery := `
	DELETE FROM banned_websites
	WHERE guild_id = ? AND id = ?;
	`

	_, err := d.db.Exec(deleteQuery, guildID, id)
	if err != nil {
		return fmt.Errorf("failed to remove banned website: %w", err)
	}

	return nil
}

func (d *Database) RemoveBannedWebsitesByGuild(guildID string) error {
	deleteQuery := `
	DELETE FROM banned_websites
	WHERE guild_id = ?;
	`

	_, err := d.db.Exec(deleteQuery, guildID)
	if err != nil {
		return fmt.Errorf("failed to remove banned websites by guild: %w", err)
	}

	return nil
}

func (d *Database) GetBannedWebsitesByGuildId(guildID string) ([]BannedWebsite, error) {
	selectQuery := `
	SELECT id, guild_id, website_url, created_at
	FROM banned_websites
	WHERE guild_id = ?
	ORDER BY created_at DESC;
	`

	rows, err := d.db.Query(selectQuery, guildID)
	if err != nil {
		return nil, fmt.Errorf("failed to get banned websites: %w", err)
	}
	defer rows.Close()

	var bannedWebsites []BannedWebsite
	for rows.Next() {
		var website BannedWebsite
		var createdAt string
		if err := rows.Scan(&website.ID, &website.GuildID, &website.WebsiteURL, &createdAt); err != nil {
			return nil, fmt.Errorf("failed to scan banned website: %w", err)
		}

		// Parse the timestamp
		website.CreatedAt, err = time.Parse("2006-01-02 15:04:05", createdAt)
		if err != nil {
			// If parsing fails, use current time as fallback
			website.CreatedAt = time.Now()
		}

		bannedWebsites = append(bannedWebsites, website)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating over rows: %w", err)
	}

	return bannedWebsites, nil
}

// Automoderation Settings Functions

func (d *Database) GetAutomoderationSettings(guildID string) (*AutomoderationSettings, error) {
	query := `
	SELECT guild_id, banned_words_enabled, banned_websites_enabled, spam_detection_enabled
	FROM automoderation_settings
	WHERE guild_id = ?;
	`

	settings := &AutomoderationSettings{
		GuildID:               guildID,
		BannedWordsEnabled:    true, // Default values
		BannedWebsitesEnabled: true,
		SpamDetectionEnabled:  true,
	}

	err := d.db.QueryRow(query, guildID).Scan(
		&settings.GuildID,
		&settings.BannedWordsEnabled,
		&settings.BannedWebsitesEnabled,
		&settings.SpamDetectionEnabled,
	)

	if err != nil {
		// If no row exists, return default settings (all enabled)
		if errors.Is(err, sql.ErrNoRows) {
			return settings, nil
		}
		return nil, fmt.Errorf("failed to get automoderation settings: %w", err)
	}

	return settings, nil
}

func (d *Database) SetAutomoderationSettings(guildID string, settings *AutomoderationSettings) error {
	query := `
	INSERT INTO automoderation_settings (guild_id, banned_words_enabled, banned_websites_enabled, spam_detection_enabled)
	VALUES (?, ?, ?, ?)
	ON CONFLICT(guild_id) DO UPDATE SET
		banned_words_enabled = excluded.banned_words_enabled,
		banned_websites_enabled = excluded.banned_websites_enabled,
		spam_detection_enabled = excluded.spam_detection_enabled;
	`

	_, err := d.db.Exec(query, guildID, settings.BannedWordsEnabled, settings.BannedWebsitesEnabled, settings.SpamDetectionEnabled)
	if err != nil {
		return fmt.Errorf("failed to set automoderation settings: %w", err)
	}

	return nil
}
