package database

// Create a database that use SQLite to store data
import (
	"database/sql"
	"fmt"

	"github.com/dragonis41/discord-bot-moderation/pkg/utils"
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
	db, err := sql.Open("sqlite3", "sqlite3.db")
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	return &Database{db: db}, nil
}

func (d *Database) GetDBConnection() *sql.DB {
	return d.db
}

func (d *Database) CloseDatabase() error {
	if err := d.db.Close(); err != nil {
		return fmt.Errorf("failed to close database: %w", err)
	}
	utils.LogSuccess("Database connection closed successfully\n")
	return nil
}
