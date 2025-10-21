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

func (d *Database) MigrateLogChannel() error {
	createTableQuery := `
	CREATE TABLE IF NOT EXISTS log_channels (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		guild_id TEXT NOT NULL,
		channel_id TEXT NOT NULL,
		UNIQUE(guild_id, channel_id)
	);
	`

	_, err := d.db.Exec(createTableQuery)
	if err != nil {
		return fmt.Errorf("failed to create log_channels table: %w", err)
	}

	return nil
}

func (d *Database) AddLogChannel(guildID, channelID string) error {
	insertQuery := `
	INSERT OR IGNORE INTO log_channels (guild_id, channel_id)
	VALUES (?, ?);
	`

	_, err := d.db.Exec(insertQuery, guildID, channelID)
	if err != nil {
		return fmt.Errorf("failed to add moderation channel: %w", err)
	}

	return nil
}

func (d *Database) RemoveLogChannel(guildID, channelID string) error {
	deleteQuery := `
	DELETE FROM log_channels
	WHERE guild_id = ? AND channel_id = ?;
	`

	_, err := d.db.Exec(deleteQuery, guildID, channelID)
	if err != nil {
		return fmt.Errorf("failed to remove moderation channel: %w", err)
	}

	return nil
}

func (d *Database) RemoveLogChannelsByGuild(guildID string) error {
	deleteQuery := `
	DELETE FROM log_channels
	WHERE guild_id = ?;
	`

	_, err := d.db.Exec(deleteQuery, guildID)
	if err != nil {
		return fmt.Errorf("failed to remove moderation channels by guild: %w", err)
	}

	return nil
}

func (d *Database) GetLogChannelsByGuildId(guildID string) ([]string, error) {
	selectQuery := `
	SELECT channel_id FROM log_channels
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
