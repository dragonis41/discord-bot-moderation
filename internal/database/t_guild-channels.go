package database

import (
	"fmt"

	"github.com/dragonis41/discord-bot-moderation/pkg/model"
)

type LogChannelsInterface interface {
	AddLogChannel(guildID, channelID string) error
	RemoveLogChannel(guildID, channelID string) error
	RemoveLogChannelsByGuild(guildID string) error
	GetLogChannelsByGuildId(guildID string) ([]string, error)
	IsLogChannel(guildID, channelID string) bool
}

type ModerationChannelsInterface interface {
	AddModerationChannel(guildID, channelID string) error
	RemoveModerationChannel(guildID, channelID string) error
	RemoveModerationChannelsByGuild(guildID string) error
	GetModerationChannelsByGuildId(guildID string) ([]string, error)
	IsModerationChannel(guildID, channelID string) bool
}

type ExcludedChannelsInterface interface {
	AddExcludedChannel(guildID, channelID string) error
	RemoveExcludedChannel(guildID, channelID string) error
	RemoveExcludedChannelsByGuild(guildID string) error
	GetExcludedChannelsByGuildId(guildID string) ([]string, error)
	IsExcludedChannel(guildID, channelID string) bool
}

func (d *Database) MigrateGuildChannels() error {
	createTableQuery := `
	CREATE TABLE IF NOT EXISTS guild_channels (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		guild_id TEXT NOT NULL,
		channel_id TEXT NOT NULL,
		channel_type TEXT NOT NULL,
		UNIQUE(guild_id, channel_id, channel_type)
	);
	`

	_, err := d.db.Exec(createTableQuery)
	if err != nil {
		return fmt.Errorf("failed to create guild_channels table: %w", err)
	}

	return nil
}

// Log Channel Functions

func (d *Database) AddLogChannel(guildID, channelID string) error {
	insertQuery := `
	INSERT OR IGNORE INTO guild_channels (guild_id, channel_id, channel_type)
	VALUES (?, ?, ?);
	`

	_, err := d.db.Exec(insertQuery, guildID, channelID, model.ChannelTypeLog)
	if err != nil {
		return fmt.Errorf("failed to add log channel: %w", err)
	}

	return nil
}

func (d *Database) RemoveLogChannel(guildID, channelID string) error {
	deleteQuery := `
	DELETE FROM guild_channels
	WHERE guild_id = ? AND channel_id = ? AND channel_type = ?;
	`

	_, err := d.db.Exec(deleteQuery, guildID, channelID, model.ChannelTypeLog)
	if err != nil {
		return fmt.Errorf("failed to remove log channel: %w", err)
	}

	return nil
}

func (d *Database) RemoveLogChannelsByGuild(guildID string) error {
	deleteQuery := `
	DELETE FROM guild_channels
	WHERE guild_id = ? AND channel_type = ?;
	`

	_, err := d.db.Exec(deleteQuery, guildID, model.ChannelTypeLog)
	if err != nil {
		return fmt.Errorf("failed to remove log channels by guild: %w", err)
	}

	return nil
}

func (d *Database) GetLogChannelsByGuildId(guildID string) ([]string, error) {
	selectQuery := `
	SELECT channel_id FROM guild_channels
	WHERE guild_id = ? AND channel_type = ?;
	`

	rows, err := d.db.Query(selectQuery, guildID, model.ChannelTypeLog)
	if err != nil {
		return nil, fmt.Errorf("failed to get log channels: %w", err)
	}
	defer rows.Close()

	var channelIDs []string
	for rows.Next() {
		var channelID string
		if err := rows.Scan(&channelID); err != nil {
			return nil, fmt.Errorf("failed to scan channel ID: %w", err)
		}
		channelIDs = append(channelIDs, channelID)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating over rows: %w", err)
	}

	return channelIDs, nil
}

func (d *Database) IsLogChannel(guildID, channelID string) bool {
	selectQuery := `
	SELECT 1 FROM guild_channels
	WHERE guild_id = ? AND channel_id = ? AND channel_type = ?;
	`

	row := d.db.QueryRow(selectQuery, guildID, channelID, model.ChannelTypeLog)
	var exists int
	err := row.Scan(&exists)
	if err != nil {
		if err.Error() == "sql: no rows in result set" {
			return false
		}
		return false
	}

	return true
}

// Moderation Channel Functions

func (d *Database) AddModerationChannel(guildID, channelID string) error {
	insertQuery := `
	INSERT OR IGNORE INTO guild_channels (guild_id, channel_id, channel_type)
	VALUES (?, ?, ?);
	`

	_, err := d.db.Exec(insertQuery, guildID, channelID, model.ChannelTypeModeration)
	if err != nil {
		return fmt.Errorf("failed to add moderation channel: %w", err)
	}

	return nil
}

func (d *Database) RemoveModerationChannel(guildID, channelID string) error {
	deleteQuery := `
	DELETE FROM guild_channels
	WHERE guild_id = ? AND channel_id = ? AND channel_type = ?;
	`

	_, err := d.db.Exec(deleteQuery, guildID, channelID, model.ChannelTypeModeration)
	if err != nil {
		return fmt.Errorf("failed to remove moderation channel: %w", err)
	}

	return nil
}

func (d *Database) RemoveModerationChannelsByGuild(guildID string) error {
	deleteQuery := `
	DELETE FROM guild_channels
	WHERE guild_id = ? AND channel_type = ?;
	`

	_, err := d.db.Exec(deleteQuery, guildID, model.ChannelTypeModeration)
	if err != nil {
		return fmt.Errorf("failed to remove moderation channels by guild: %w", err)
	}

	return nil
}

func (d *Database) GetModerationChannelsByGuildId(guildID string) ([]string, error) {
	selectQuery := `
	SELECT channel_id FROM guild_channels
	WHERE guild_id = ? AND channel_type = ?;
	`

	rows, err := d.db.Query(selectQuery, guildID, model.ChannelTypeModeration)
	if err != nil {
		return nil, fmt.Errorf("failed to get moderation channels: %w", err)
	}
	defer rows.Close()

	var channelIDs []string
	for rows.Next() {
		var channelID string
		if err := rows.Scan(&channelID); err != nil {
			return nil, fmt.Errorf("failed to scan channel ID: %w", err)
		}
		channelIDs = append(channelIDs, channelID)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating over rows: %w", err)
	}

	return channelIDs, nil
}

func (d *Database) IsModerationChannel(guildID, channelID string) bool {
	selectQuery := `
	SELECT 1 FROM guild_channels
	WHERE guild_id = ? AND channel_id = ? AND channel_type = ?;
	`

	row := d.db.QueryRow(selectQuery, guildID, channelID, model.ChannelTypeModeration)
	var exists int
	err := row.Scan(&exists)
	if err != nil {
		if err.Error() == "sql: no rows in result set" {
			return false
		}
		return false
	}

	return true
}

// Excluded Channel Functions

func (d *Database) AddExcludedChannel(guildID, channelID string) error {
	insertQuery := `
	INSERT OR IGNORE INTO guild_channels (guild_id, channel_id, channel_type)
	VALUES (?, ?, ?);
	`

	_, err := d.db.Exec(insertQuery, guildID, channelID, model.ChannelTypeExcluded)
	if err != nil {
		return fmt.Errorf("failed to add excluded channel: %w", err)
	}

	return nil
}

func (d *Database) RemoveExcludedChannel(guildID, channelID string) error {
	deleteQuery := `
	DELETE FROM guild_channels
	WHERE guild_id = ? AND channel_id = ? AND channel_type = ?;
	`

	_, err := d.db.Exec(deleteQuery, guildID, channelID, model.ChannelTypeExcluded)
	if err != nil {
		return fmt.Errorf("failed to remove excluded channel: %w", err)
	}

	return nil
}

func (d *Database) RemoveExcludedChannelsByGuild(guildID string) error {
	deleteQuery := `
	DELETE FROM guild_channels
	WHERE guild_id = ? AND channel_type = ?;
	`

	_, err := d.db.Exec(deleteQuery, guildID, model.ChannelTypeExcluded)
	if err != nil {
		return fmt.Errorf("failed to remove excluded channels by guild: %w", err)
	}

	return nil
}

func (d *Database) GetExcludedChannelsByGuildId(guildID string) ([]string, error) {
	selectQuery := `
	SELECT channel_id FROM guild_channels
	WHERE guild_id = ? AND channel_type = ?;
	`

	rows, err := d.db.Query(selectQuery, guildID, model.ChannelTypeExcluded)
	if err != nil {
		return nil, fmt.Errorf("failed to get excluded channels: %w", err)
	}
	defer rows.Close()

	var channelIDs []string
	for rows.Next() {
		var channelID string
		if err := rows.Scan(&channelID); err != nil {
			return nil, fmt.Errorf("failed to scan channel ID: %w", err)
		}
		channelIDs = append(channelIDs, channelID)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating over rows: %w", err)
	}

	return channelIDs, nil
}

func (d *Database) IsExcludedChannel(guildID, channelID string) bool {
	selectQuery := `
	SELECT 1 FROM guild_channels
	WHERE guild_id = ? AND channel_id = ? AND channel_type = ?;
	`

	row := d.db.QueryRow(selectQuery, guildID, channelID, model.ChannelTypeExcluded)
	var exists int
	err := row.Scan(&exists)
	if err != nil {
		if err.Error() == "sql: no rows in result set" {
			return false
		}
		return false
	}

	return true
}
