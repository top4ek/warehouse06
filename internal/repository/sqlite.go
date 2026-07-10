package repository

import (
	"context"
	"database/sql"
	"fmt"
	"strings"

	_ "github.com/mattn/go-sqlite3"
	"go.uber.org/zap"
)

type SQLiteRepository struct {
	db  *sql.DB
	log *zap.Logger
}

func NewSQLiteRepository(dsn string, log *zap.Logger) (*SQLiteRepository, error) {
	inMem := IsInMemoryDSN(dsn)
	openDSN := dsn
	if !inMem {
		openDSN = fileModeDSN(dsn)
	}

	db, err := sql.Open("sqlite3", openDSN)
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	if err := db.PingContext(context.Background()); err != nil {
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	if err := configureSQLite(db, inMem); err != nil {
		return nil, fmt.Errorf("failed to configure sqlite: %w", err)
	}

	repo := &SQLiteRepository{
		db:  db,
		log: log,
	}

	if err := repo.initSchema(); err != nil {
		return nil, fmt.Errorf("failed to initialize schema: %w", err)
	}

	return repo, nil
}

func (r *SQLiteRepository) Close() error {
	return r.db.Close()
}

// fileModeDSN appends per-connection pragmas as DSN parameters so every pooled
// connection is configured identically; a one-off db.Exec would reach only a
// single connection out of the pool.
func fileModeDSN(dsn string) string {
	sep := "?"
	if strings.Contains(dsn, "?") {
		sep = "&"
	}
	return dsn + sep + "_busy_timeout=30000&_journal_mode=WAL&_foreign_keys=on"
}

func configureSQLite(db *sql.DB, inMem bool) error {
	if inMem {
		// :memory: creates a separate database per connection unless the pool is
		// limited to one. With a single connection, pragmas can be set once here.
		db.SetMaxOpenConns(1)
		db.SetMaxIdleConns(1)
		pragmas := []string{"PRAGMA busy_timeout = 30000", "PRAGMA foreign_keys = ON"}
		for _, pragma := range pragmas {
			if _, err := db.ExecContext(context.Background(), pragma); err != nil {
				return fmt.Errorf("%s: %w", pragma, err)
			}
		}
		return nil
	}

	// WAL allows concurrent readers while sync writes; a single connection would
	// block all API handlers for the duration of SaveEntriesAndAuthors.
	// File-mode pragmas ride on the DSN (see fileModeDSN).
	db.SetMaxOpenConns(25)
	db.SetMaxIdleConns(5)
	return nil
}

func (r *SQLiteRepository) initSchema() error {
	queries := []string{
		`CREATE TABLE IF NOT EXISTS entries (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			path TEXT UNIQUE NOT NULL,
			name TEXT,
			description TEXT,
			content_html TEXT,
			date TEXT,
			type TEXT,
			youtube TEXT,
			created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
			updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
		);`,
		`CREATE TABLE IF NOT EXISTS authors (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			directory_name TEXT UNIQUE NOT NULL,
			name TEXT,
			address TEXT,
			content_html TEXT
		);`,
		`CREATE TABLE IF NOT EXISTS tags (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			name TEXT UNIQUE NOT NULL
		);`,
		`CREATE TABLE IF NOT EXISTS entry_authors (
			entry_id INTEGER,
			author_id INTEGER,
			PRIMARY KEY (entry_id, author_id),
			FOREIGN KEY(entry_id) REFERENCES entries(id) ON DELETE CASCADE,
			FOREIGN KEY(author_id) REFERENCES authors(id) ON DELETE CASCADE
		);`,
		`CREATE TABLE IF NOT EXISTS entry_tags (
			entry_id INTEGER,
			tag_id INTEGER,
			PRIMARY KEY (entry_id, tag_id),
			FOREIGN KEY(entry_id) REFERENCES entries(id) ON DELETE CASCADE,
			FOREIGN KEY(tag_id) REFERENCES tags(id) ON DELETE CASCADE
		);`,
		`CREATE TABLE IF NOT EXISTS files (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			entry_id INTEGER,
			filename TEXT NOT NULL,
			filepath TEXT NOT NULL,
			is_image BOOLEAN DEFAULT 0,
			FOREIGN KEY(entry_id) REFERENCES entries(id) ON DELETE CASCADE
		);`,
		`CREATE TABLE IF NOT EXISTS entry_requires (
			entry_id INTEGER,
			required_path TEXT,
			PRIMARY KEY (entry_id, required_path),
			FOREIGN KEY(entry_id) REFERENCES entries(id) ON DELETE CASCADE
		);`,
		// FTS5 Virtual Table for full-text search
		`CREATE VIRTUAL TABLE IF NOT EXISTS entries_fts USING fts5(
			name,
			description,
			content_html,
			content='entries',
			content_rowid='id'
		);`,
		// Triggers to keep FTS table in sync
		`CREATE TRIGGER IF NOT EXISTS entries_ai AFTER INSERT ON entries BEGIN
			INSERT INTO entries_fts(rowid, name, description, content_html) 
			VALUES (new.id, new.name, new.description, new.content_html);
		END;`,
		`CREATE TRIGGER IF NOT EXISTS entries_ad AFTER DELETE ON entries BEGIN
			INSERT INTO entries_fts(entries_fts, rowid, name, description, content_html) 
			VALUES('delete', old.id, old.name, old.description, old.content_html);
		END;`,
		`CREATE TRIGGER IF NOT EXISTS entries_au AFTER UPDATE ON entries BEGIN
			INSERT INTO entries_fts(entries_fts, rowid, name, description, content_html) 
			VALUES('delete', old.id, old.name, old.description, old.content_html);
			INSERT INTO entries_fts(rowid, name, description, content_html) 
			VALUES (new.id, new.name, new.description, new.content_html);
		END;`,
	}

	ctx := context.Background()
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	for _, query := range queries {
		if _, err := tx.ExecContext(ctx, query); err != nil {
			return fmt.Errorf("failed to execute query %q: %w", query, err)
		}
	}

	return tx.Commit()
}

func (r *SQLiteRepository) ClearAll(ctx context.Context) error {
	tables := []string{"entry_tags", "entry_authors", "entry_requires", "files", "tags", "authors", "entries"}

	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	for _, table := range tables {
		if _, err := tx.ExecContext(ctx, fmt.Sprintf("DELETE FROM %s", table)); err != nil {
			return err
		}
	}

	return tx.Commit()
}
