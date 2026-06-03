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

// Generic helpers shared by all channel types --------------------------------
//
// Log, moderation and excluded channels are all rows in guild_channels that
// differ only by channel_type, so the CRUD logic is implemented once and the
// public, type-specific functions simply delegate with the right type.

func (d *Database) addChannel(guildID, channelID, channelType string) error {
	insertQuery := `
	INSERT OR IGNORE INTO guild_channels (guild_id, channel_id, channel_type)
	VALUES (?, ?, ?);
	`

	if _, err := d.db.Exec(insertQuery, guildID, channelID, channelType); err != nil {
		return fmt.Errorf("failed to add %s channel: %w", channelType, err)
	}
	return nil
}

func (d *Database) removeChannel(guildID, channelID, channelType string) error {
	deleteQuery := `
	DELETE FROM guild_channels
	WHERE guild_id = ? AND channel_id = ? AND channel_type = ?;
	`

	if _, err := d.db.Exec(deleteQuery, guildID, channelID, channelType); err != nil {
		return fmt.Errorf("failed to remove %s channel: %w", channelType, err)
	}
	return nil
}

func (d *Database) removeChannelsByGuild(guildID, channelType string) error {
	deleteQuery := `
	DELETE FROM guild_channels
	WHERE guild_id = ? AND channel_type = ?;
	`

	if _, err := d.db.Exec(deleteQuery, guildID, channelType); err != nil {
		return fmt.Errorf("failed to remove %s channels by guild: %w", channelType, err)
	}
	return nil
}

func (d *Database) getChannelsByGuild(guildID, channelType string) ([]string, error) {
	selectQuery := `
	SELECT channel_id FROM guild_channels
	WHERE guild_id = ? AND channel_type = ?;
	`

	rows, err := d.db.Query(selectQuery, guildID, channelType)
	if err != nil {
		return nil, fmt.Errorf("failed to get %s channels: %w", channelType, err)
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

func (d *Database) isChannel(guildID, channelID, channelType string) bool {
	selectQuery := `
	SELECT 1 FROM guild_channels
	WHERE guild_id = ? AND channel_id = ? AND channel_type = ?;
	`

	var exists int
	return d.db.QueryRow(selectQuery, guildID, channelID, channelType).Scan(&exists) == nil
}

// Log Channel Functions

func (d *Database) AddLogChannel(guildID, channelID string) error {
	return d.addChannel(guildID, channelID, model.ChannelTypeLog)
}

func (d *Database) RemoveLogChannel(guildID, channelID string) error {
	return d.removeChannel(guildID, channelID, model.ChannelTypeLog)
}

func (d *Database) RemoveLogChannelsByGuild(guildID string) error {
	return d.removeChannelsByGuild(guildID, model.ChannelTypeLog)
}

func (d *Database) GetLogChannelsByGuildId(guildID string) ([]string, error) {
	return d.getChannelsByGuild(guildID, model.ChannelTypeLog)
}

func (d *Database) IsLogChannel(guildID, channelID string) bool {
	return d.isChannel(guildID, channelID, model.ChannelTypeLog)
}

// Moderation Channel Functions

func (d *Database) AddModerationChannel(guildID, channelID string) error {
	return d.addChannel(guildID, channelID, model.ChannelTypeModeration)
}

func (d *Database) RemoveModerationChannel(guildID, channelID string) error {
	return d.removeChannel(guildID, channelID, model.ChannelTypeModeration)
}

func (d *Database) RemoveModerationChannelsByGuild(guildID string) error {
	return d.removeChannelsByGuild(guildID, model.ChannelTypeModeration)
}

func (d *Database) GetModerationChannelsByGuildId(guildID string) ([]string, error) {
	return d.getChannelsByGuild(guildID, model.ChannelTypeModeration)
}

func (d *Database) IsModerationChannel(guildID, channelID string) bool {
	return d.isChannel(guildID, channelID, model.ChannelTypeModeration)
}

// Excluded Channel Functions

func (d *Database) AddExcludedChannel(guildID, channelID string) error {
	return d.addChannel(guildID, channelID, model.ChannelTypeExcluded)
}

func (d *Database) RemoveExcludedChannel(guildID, channelID string) error {
	return d.removeChannel(guildID, channelID, model.ChannelTypeExcluded)
}

func (d *Database) RemoveExcludedChannelsByGuild(guildID string) error {
	return d.removeChannelsByGuild(guildID, model.ChannelTypeExcluded)
}

func (d *Database) GetExcludedChannelsByGuildId(guildID string) ([]string, error) {
	return d.getChannelsByGuild(guildID, model.ChannelTypeExcluded)
}

func (d *Database) IsExcludedChannel(guildID, channelID string) bool {
	return d.isChannel(guildID, channelID, model.ChannelTypeExcluded)
}
