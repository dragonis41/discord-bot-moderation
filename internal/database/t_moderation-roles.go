package database

import (
	"fmt"
)

type ModerationRolesInterface interface {
	AddModerationRole(guildID, roleID string) error
	RemoveModerationRole(guildID, roleID string) error
	RemoveModerationRolesByGuild(guildID string) error
	GetModerationRolesByGuildId(guildID string) ([]string, error)
}

func (d *Database) MigrateModerationRole() error {
	createTableQuery := `
	CREATE TABLE IF NOT EXISTS moderation_roles (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		guild_id TEXT NOT NULL,
		role_id TEXT NOT NULL,
		UNIQUE(guild_id, role_id)
	);
	`

	_, err := d.db.Exec(createTableQuery)
	if err != nil {
		return fmt.Errorf("failed to create moderation_roles table: %w", err)
	}

	return nil
}

func (d *Database) AddModerationRole(guildID, roleID string) error {
	insertQuery := `
	INSERT OR IGNORE INTO moderation_roles (guild_id, role_id)
	VALUES (?, ?);
	`

	_, err := d.db.Exec(insertQuery, guildID, roleID)
	if err != nil {
		return fmt.Errorf("failed to add moderation role: %w", err)
	}

	return nil
}

func (d *Database) RemoveModerationRole(guildID, roleID string) error {
	deleteQuery := `
	DELETE FROM moderation_roles
	WHERE guild_id = ? AND role_id = ?;
	`

	_, err := d.db.Exec(deleteQuery, guildID, roleID)
	if err != nil {
		return fmt.Errorf("failed to remove moderation role: %w", err)
	}

	return nil
}

func (d *Database) RemoveModerationRolesByGuild(guildID string) error {
	deleteQuery := `
	DELETE FROM moderation_roles
	WHERE guild_id = ?;
	`

	_, err := d.db.Exec(deleteQuery, guildID)
	if err != nil {
		return fmt.Errorf("failed to remove moderation roles by guild: %w", err)
	}

	return nil
}

func (d *Database) GetModerationRolesByGuildId(guildID string) ([]string, error) {
	selectQuery := `
	SELECT role_id FROM moderation_roles
	WHERE guild_id = ?;
	`

	rows, err := d.db.Query(selectQuery, guildID)
	if err != nil {
		return nil, fmt.Errorf("failed to get moderation roles: %w", err)
	}
	defer rows.Close()

	var roleIDs []string
	for rows.Next() {
		var roleID string
		if err := rows.Scan(&roleID); err != nil {
			return nil, fmt.Errorf("failed to scan role ID: %w", err)
		}
		roleIDs = append(roleIDs, roleID)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating over rows: %w", err)
	}

	return roleIDs, nil
}
