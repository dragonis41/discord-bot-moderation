package database

import (
	"database/sql"
	"errors"
	"fmt"
	"os"

	_ "github.com/mattn/go-sqlite3"
)

type DbInterface interface {
	Migrate() error
	GetDBConnection() *sql.DB
	CloseDatabase() error
}

type Database struct {
	db *sql.DB
}

func NewDatabase() (*Database, error) {
	// Create the ./data directory if it doesn't exist
	if _, err := os.Stat("data"); errors.Is(err, os.ErrNotExist) {
		err := os.Mkdir("data", os.ModePerm)
		if err != nil {
			return nil, fmt.Errorf("failed to create data directory: %w", err)
		}
	}

	db, err := sql.Open("sqlite3", "./data/sqlite3.db")
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	return &Database{db: db}, nil
}

func (d *Database) Migrate() error {
	if err := d.MigrateAutomoderation(); err != nil {
		return err
	}
	if err := d.MigrateGuildChannels(); err != nil {
		return err
	}
	if err := d.MigrateLogsConfigs(); err != nil {
		return err
	}
	if err := d.MigrateModerationRole(); err != nil {
		return err
	}
	if err := d.MigrateGuildSettings(); err != nil {
		return err
	}
	if err := d.MigrateModerationLogs(); err != nil {
		return err
	}
	if err := d.MigrateSystemLogs(); err != nil {
		return err
	}

	return nil
}

func (d *Database) GetDBConnection() *sql.DB {
	return d.db
}

func (d *Database) CloseDatabase() error {
	if err := d.db.Close(); err != nil {
		return fmt.Errorf("failed to close database: %w", err)
	}
	return nil
}
