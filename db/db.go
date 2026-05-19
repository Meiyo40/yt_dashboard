package db

import (
	"database/sql"
	"fmt"
	"log"

	_ "modernc.org/sqlite"
)

// DB is the global database connection pool.
var DB *sql.DB

// InitDB initializes the SQLite database connection and sets up tables.
func InitDB(dbPath string) (*sql.DB, error) {
	var err error
	// Open the database file. If it doesn't exist, it will be created automatically.
	DB, err = sql.Open("sqlite", dbPath)
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	// Ping the database to verify the connection is alive.
	if err = DB.Ping(); err != nil {
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	// Enable foreign key constraints in SQLite.
	if _, err = DB.Exec("PRAGMA foreign_keys = ON;"); err != nil {
		return nil, fmt.Errorf("failed to enable foreign keys: %w", err)
	}

	// Create tables if they do not exist.
	if err = createTables(); err != nil {
		return nil, fmt.Errorf("failed to create tables: %w", err)
	}

	log.Printf("Successfully connected to SQLite database at: %s", dbPath)
	return DB, nil
}

func createTables() error {
	// Table: own_comments
	// Stores the YouTube comments posted by the user that we are tracking.
	ownCommentsTable := `
	CREATE TABLE IF NOT EXISTS own_comments (
		comment_id TEXT PRIMARY KEY,
		video_id TEXT NOT NULL,
		video_title TEXT,
		channel_title TEXT,
		text_original TEXT NOT NULL,
		posted_at TEXT NOT NULL,
		last_checked_at TEXT,
		reply_count INTEGER DEFAULT 0 NOT NULL
	);`

	// Table: replies
	// Stores replies made by other users on our comments.
	repliesTable := `
	CREATE TABLE IF NOT EXISTS replies (
		id TEXT PRIMARY KEY,
		parent_id TEXT NOT NULL,
		author_display_name TEXT NOT NULL,
		author_profile_image_url TEXT NOT NULL,
		author_channel_id TEXT NOT NULL,
		text_display TEXT NOT NULL,
		published_at TEXT NOT NULL,
		seen INTEGER DEFAULT 0 NOT NULL,
		FOREIGN KEY (parent_id) REFERENCES own_comments(comment_id) ON DELETE CASCADE
	);`

	// Table: oauth_tokens
	// Singleton-like table to persist the user's OAuth tokens.
	oauthTokensTable := `
	CREATE TABLE IF NOT EXISTS oauth_tokens (
		id TEXT PRIMARY KEY DEFAULT 'current' CHECK (id = 'current'),
		access_token TEXT NOT NULL,
		refresh_token TEXT NOT NULL,
		token_type TEXT NOT NULL,
		expiry TEXT NOT NULL
	);`

	queries := []string{ownCommentsTable, repliesTable, oauthTokensTable}
	for _, query := range queries {
		if _, err := DB.Exec(query); err != nil {
			return err
		}
	}

	return nil
}
