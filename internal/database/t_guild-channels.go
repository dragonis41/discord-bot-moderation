package database

import (
	"fmt"
)

type LogChannelsInterface interface {
	AddLogChannel(guildID, channelID string) error
	RemoveLogChannel(guildID, channelID string) error
	RemoveLogChannelsByGuild(guildID string) error
	GetLogChannelsByGuildId(guildID string) ([]string, error)
}

type ModerationChannelsInterface interface {
	AddModerationChannel(guildID, channelID string) error
	RemoveModerationChannel(guildID, channelID string) error
	RemoveModerationChannelsByGuild(guildID string) error
	GetModerationChannelsByGuildId(guildID string) ([]string, error)
}

const (
	ChannelTypeLog        = "log"
	ChannelTypeModeration = "moderation"
)

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

	_, err := d.db.Exec(insertQuery, guildID, channelID, ChannelTypeLog)
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

	_, err := d.db.Exec(deleteQuery, guildID, channelID, ChannelTypeLog)
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

	_, err := d.db.Exec(deleteQuery, guildID, ChannelTypeLog)
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

	rows, err := d.db.Query(selectQuery, guildID, ChannelTypeLog)
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

// Moderation Channel Functions

func (d *Database) AddModerationChannel(guildID, channelID string) error {
	insertQuery := `
	INSERT OR IGNORE INTO guild_channels (guild_id, channel_id, channel_type)
	VALUES (?, ?, ?);
	`

	_, err := d.db.Exec(insertQuery, guildID, channelID, ChannelTypeModeration)
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

	_, err := d.db.Exec(deleteQuery, guildID, channelID, ChannelTypeModeration)
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

	_, err := d.db.Exec(deleteQuery, guildID, ChannelTypeModeration)
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

	rows, err := d.db.Query(selectQuery, guildID, ChannelTypeModeration)
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
