package store

import (
	"database/sql"
	"fmt"
	"log"
	"os"
	"path/filepath"

	_ "modernc.org/sqlite"
)

type Store struct {
	db *sql.DB
}

func NewStore(dbPath string) (*Store, error) {
	if dbPath == "" {
		// Default to current directory if not specified
		cwd, _ := os.Getwd()
		dbPath = filepath.Join(cwd, "normattiva.db")
	}

	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	if err := db.Ping(); err != nil {
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	s := &Store{db: db}
	if err := s.migrate(); err != nil {
		return nil, fmt.Errorf("failed to migrate database: %w", err)
	}

	return s, nil
}

func (s *Store) Close() error {
	return s.db.Close()
}

func (s *Store) migrate() error {
	// Simple migration: create tables if they don't exist
	queries := []string{
		`CREATE TABLE IF NOT EXISTS users (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			name TEXT NOT NULL,
			color TEXT NOT NULL,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP
		);`,
		`CREATE TABLE IF NOT EXISTS bookmarks (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			user_id INTEGER NOT NULL,
			doc_id TEXT NOT NULL,
			title TEXT NOT NULL,
			date TEXT NOT NULL,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			UNIQUE(user_id, doc_id),
			FOREIGN KEY(user_id) REFERENCES users(id) ON DELETE CASCADE
		);`,
		`CREATE TABLE IF NOT EXISTS annotations (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			user_id INTEGER NOT NULL,
			doc_id TEXT NOT NULL,
			selection_data TEXT NOT NULL, -- JSON with start/end offsets or text
			comment TEXT,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			FOREIGN KEY(user_id) REFERENCES users(id) ON DELETE CASCADE
		);`,
	}

	for _, q := range queries {
		if _, err := s.db.Exec(q); err != nil {
			return fmt.Errorf("migration failed for query %s: %w", q, err)
		}
	}

	// Add new columns if they don't exist (SQLite doesn't support 'IF NOT EXISTS' for ADD COLUMN)
	extraMigrations := []struct {
		table string
		query string
	}{
		{"users", "ALTER TABLE users ADD COLUMN theme TEXT DEFAULT 'default'"},
		{"users", "ALTER TABLE users ADD COLUMN ui_language TEXT DEFAULT 'it'"},
		{"users", "ALTER TABLE users ADD COLUMN mode TEXT DEFAULT 'light'"},
		{"users", "ALTER TABLE users ADD COLUMN ui_state TEXT DEFAULT '{}'"},
		{"bookmarks", "ALTER TABLE bookmarks ADD COLUMN category TEXT DEFAULT 'General'"},
		{"annotations", "ALTER TABLE annotations ADD COLUMN location_id TEXT"},
		{"annotations", "ALTER TABLE annotations ADD COLUMN selection_offset INTEGER DEFAULT 0"},
		{"annotations", "ALTER TABLE annotations ADD COLUMN prefix TEXT DEFAULT ''"},
		{"annotations", "ALTER TABLE annotations ADD COLUMN suffix TEXT DEFAULT ''"},
	}

	for _, m := range extraMigrations {
		// Ignore error if column already exists
		_, _ = s.db.Exec(m.query)
	}

	// Check if default user exists, if not create one
	var count int
	err := s.db.QueryRow("SELECT COUNT(*) FROM users").Scan(&count)
	if err == nil && count == 0 {
		_, _ = s.db.Exec(`INSERT INTO users (name, color) VALUES ('Default User', '#3b82f6')`)
		log.Println("Created default user")
	}

	return nil
}
