package database

import (
	"github.com/dragonis41/discord-bot-moderation/pkg/utils"
)

func (d *Database) Migrate() error {
	utils.LogInfo("Starting database migration...")

	if err := d.MigrateLogs(); err != nil {
		return err
	}
	if err := d.MigrateLogChannel(); err != nil {
		return err
	}
	if err := d.MigrateModerationChannel(); err != nil {
		return err
	}
	if err := d.MigrateModerationRole(); err != nil {
		return err
	}

	utils.LogSuccess("Database initialized successfully")
	return nil
}
