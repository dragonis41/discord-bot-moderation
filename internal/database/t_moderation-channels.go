package database

import (
	"fmt"
)

type ModerationChannelsInterface interface {
	AddModerationChannel(guildID, channelID string) error
	RemoveModerationChannel(guildID, channelID string) error
	RemoveModerationChannelsByGuild(guildID string) error
	GetModerationChannelsByGuildId(guildID string) ([]string, error)
}

func (d *Database) MigrateModerationChannel() error {
	createTableQuery := `
	CREATE TABLE IF NOT EXISTS moderation_channels (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		guild_id TEXT NOT NULL,
		channel_id TEXT NOT NULL,
		UNIQUE(guild_id, channel_id)
	);
	`

	_, err := d.db.Exec(createTableQuery)
	if err != nil {
		return fmt.Errorf("failed to create moderation_channels table: %w", err)
	}

	return nil
}

func (d *Database) AddModerationChannel(guildID, channelID string) error {
	insertQuery := `
	INSERT OR IGNORE INTO moderation_channels (guild_id, channel_id)
	VALUES (?, ?);
	`

	_, err := d.db.Exec(insertQuery, guildID, channelID)
	if err != nil {
		return fmt.Errorf("failed to add moderation channel: %w", err)
	}

	return nil
}

func (d *Database) RemoveModerationChannel(guildID, channelID string) error {
	deleteQuery := `
	DELETE FROM moderation_channels
	WHERE guild_id = ? AND channel_id = ?;
	`

	_, err := d.db.Exec(deleteQuery, guildID, channelID)
	if err != nil {
		return fmt.Errorf("failed to remove moderation channel: %w", err)
	}

	return nil
}

func (d *Database) RemoveModerationChannelsByGuild(guildID string) error {
	deleteQuery := `
	DELETE FROM moderation_channels
	WHERE guild_id = ?;
	`

	_, err := d.db.Exec(deleteQuery, guildID)
	if err != nil {
		return fmt.Errorf("failed to remove moderation channels by guild: %w", err)
	}

	return nil
}

func (d *Database) GetModerationChannelsByGuildId(guildID string) ([]string, error) {
	selectQuery := `
	SELECT channel_id FROM moderation_channels
	WHERE guild_id = ?;
	`

	rows, err := d.db.Query(selectQuery, guildID)
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
