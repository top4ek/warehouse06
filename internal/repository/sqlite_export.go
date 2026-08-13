package repository

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"io/fs"
	"os"

	"warehouse06/internal/domain"
)

// SnapshotTo writes a standalone copy of the database to path, for use as a
// downloadable artifact. VACUUM INTO only reads the source and works from a
// :memory: database, so the live catalog is untouched whichever DSN mode is
// configured.
//
// The FTS5 index is dropped from the copy - never from the source - so the
// resulting file opens in SQLite builds compiled without FTS5.
func (r *SQLiteRepository) SnapshotTo(ctx context.Context, path string) error {
	// VACUUM INTO refuses to write to a path that already exists.
	if err := os.Remove(path); err != nil && !errors.Is(err, fs.ErrNotExist) {
		return fmt.Errorf("remove stale snapshot %s: %w", path, err)
	}

	if _, err := r.db.ExecContext(ctx, "VACUUM INTO ?", path); err != nil {
		return fmt.Errorf("vacuum into %s: %w", path, err)
	}

	if err := stripSnapshotFTS(ctx, path); err != nil {
		_ = os.Remove(path)
		return err
	}
	return nil
}

// stripSnapshotFTS removes the FTS5 virtual table and the triggers feeding it
// from a snapshot file, then compacts it to reclaim the shadow tables' pages.
func stripSnapshotFTS(ctx context.Context, path string) error {
	db, err := sql.Open("sqlite3", path)
	if err != nil {
		return fmt.Errorf("open snapshot %s: %w", path, err)
	}
	defer func() { _ = db.Close() }()

	for _, stmt := range []string{
		"DROP TRIGGER IF EXISTS entries_ai",
		"DROP TRIGGER IF EXISTS entries_ad",
		"DROP TRIGGER IF EXISTS entries_au",
		"DROP TABLE IF EXISTS entries_fts",
		"VACUUM",
	} {
		if _, err := db.ExecContext(ctx, stmt); err != nil {
			return fmt.Errorf("snapshot %s: %w", stmt, err)
		}
	}
	return nil
}

// ListAllEntries returns every entry (no pagination), including
// content_html and full relation data (Tags, Authors, Files, Requires), for
// bulk export use cases such as the REBUS export. Unlike GetEntries, whose
// underlying query never selects content_html, this reads the complete row.
// It bypasses the HTTP layer's list-limit cap entirely, so it is intended
// only for internal/trusted callers, never exposed as a raw limit-
// controllable HTTP parameter.
func (r *SQLiteRepository) ListAllEntries(ctx context.Context) ([]*domain.Entry, error) {
	entries := make([]*domain.Entry, 0)
	err := r.forEachRow(ctx, `
		SELECT id, path, name, description, content_html, date, type, youtube, created_at, updated_at
		FROM entries
		ORDER BY path ASC`, nil, func(rows *sql.Rows) error {
		var e domain.Entry
		var createdAt, updatedAt sql.NullTime
		if err := rows.Scan(
			&e.ID, &e.Path, &e.Name, &e.Description, &e.ContentHTML, &e.Date, &e.Type, &e.Youtube, &createdAt, &updatedAt,
		); err != nil {
			return err
		}
		if createdAt.Valid {
			e.CreatedAt = createdAt.Time
		}
		if updatedAt.Valid {
			e.UpdatedAt = updatedAt.Time
		}
		entries = append(entries, &e)
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("list all entries query failed: %w", err)
	}

	for _, e := range entries {
		enrichEntry(e)
	}
	if err := r.populateEntriesListRelations(ctx, entries); err != nil {
		return nil, err
	}
	if err := r.populateEntriesFilesBatch(ctx, entries); err != nil {
		return nil, err
	}
	if err := r.populateEntriesRequiresBatch(ctx, entries); err != nil {
		return nil, err
	}

	return entries, nil
}

// ListAllAuthors returns every author including content_html, for bulk export
// use cases such as the REBUS export. Unlike ListAuthors, whose underlying
// query deliberately omits content_html to keep the /api/authors list payload
// small (and which carries entry counts and preview images this has no use
// for), this reads the author's full text.
func (r *SQLiteRepository) ListAllAuthors(ctx context.Context) ([]*domain.Author, error) {
	authors := make([]*domain.Author, 0)
	err := r.forEachRow(ctx, `
		SELECT id, directory_name, COALESCE(name, ''), COALESCE(address, ''), COALESCE(content_html, '')
		FROM authors
		ORDER BY directory_name ASC`, nil, func(rows *sql.Rows) error {
		var a domain.Author
		if err := rows.Scan(&a.ID, &a.DirectoryName, &a.Name, &a.Address, &a.ContentHTML); err != nil {
			return err
		}
		authors = append(authors, &a)
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("list all authors query failed: %w", err)
	}
	return authors, nil
}

// populateEntriesFilesBatch batch-loads the full files list for each entry
// (populateEntriesListRelations only loads the first preview image).
func (r *SQLiteRepository) populateEntriesFilesBatch(ctx context.Context, entries []*domain.Entry) error {
	if len(entries) == 0 {
		return nil
	}

	byID := entryByID(entries)
	ids := entryIDs(entries)

	query := fmt.Sprintf(`
		SELECT id, entry_id, filename, filepath, is_image, size, sha256
		FROM files
		WHERE entry_id IN (%s)`, placeholders(len(ids)))
	err := r.forEachRow(ctx, query, ids, func(rows *sql.Rows) error {
		var f domain.File
		if err := rows.Scan(&f.ID, &f.EntryID, &f.Filename, &f.Filepath, &f.IsImage, &f.Size, &f.SHA256); err != nil {
			return err
		}
		if e := byID[f.EntryID]; e != nil {
			e.Files = append(e.Files, f)
		}
		return nil
	})
	if err != nil {
		return fmt.Errorf("batch load files: %w", err)
	}
	return nil
}

// populateEntriesRequiresBatch batch-loads entry_requires for each entry.
func (r *SQLiteRepository) populateEntriesRequiresBatch(ctx context.Context, entries []*domain.Entry) error {
	if len(entries) == 0 {
		return nil
	}

	byID := entryByID(entries)
	ids := entryIDs(entries)

	query := fmt.Sprintf(`
		SELECT entry_id, required_path
		FROM entry_requires
		WHERE entry_id IN (%s)`, placeholders(len(ids)))
	err := r.forEachRow(ctx, query, ids, func(rows *sql.Rows) error {
		var entryID int64
		var reqPath string
		if err := rows.Scan(&entryID, &reqPath); err != nil {
			return err
		}
		if e := byID[entryID]; e != nil {
			e.Requires = append(e.Requires, reqPath)
		}
		return nil
	})
	if err != nil {
		return fmt.Errorf("batch load requires: %w", err)
	}
	return nil
}
