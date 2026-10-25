package repository

import (
	"context"
	"fmt"

	"warehouse06/internal/domain"
)

// EntryIDByPath resolves navigation without loading content and relations.
func (r *SQLiteRepository) EntryIDByPath(ctx context.Context, path string) (int64, error) {
	var id int64
	if err := r.db.QueryRowContext(ctx, "SELECT id FROM entries WHERE path = ?", path).Scan(&id); err != nil {
		return 0, fmt.Errorf("resolve entry id: %w", err)
	}
	return id, nil
}

// EntryByID resolves compact Gopher selectors without exposing long storage paths.
func (r *SQLiteRepository) EntryByID(ctx context.Context, id int64) (*domain.Entry, error) {
	var path string
	if err := r.db.QueryRowContext(ctx, "SELECT path FROM entries WHERE id = ?", id).Scan(&path); err != nil {
		return nil, fmt.Errorf("resolve entry: %w", err)
	}
	return r.GetEntryByPath(ctx, path)
}

// FileByID returns a catalog file for a compact download selector.
func (r *SQLiteRepository) FileByID(ctx context.Context, id int64) (*domain.File, error) {
	f := &domain.File{}
	if err := r.db.QueryRowContext(ctx, "SELECT id, entry_id, filename, filepath FROM files WHERE id = ?", id).Scan(&f.ID, &f.EntryID, &f.Filename, &f.Filepath); err != nil {
		return nil, fmt.Errorf("resolve file: %w", err)
	}
	return f, nil
}

// GopherFilter resolves numeric selectors into existing catalog filter values.
func (r *SQLiteRepository) GopherFilter(ctx context.Context, kind string, id int64) (string, error) {
	var query string
	switch kind {
	case "p":
		query = "SELECT path FROM entries WHERE id = ?"
	case "a":
		query = "SELECT directory_name FROM authors WHERE id = ?"
	case "g":
		query = "SELECT name FROM tags WHERE id = ?"
	default:
		return "", fmt.Errorf("unknown catalog filter %q", kind)
	}
	var value string
	if err := r.db.QueryRowContext(ctx, query, id).Scan(&value); err != nil {
		return "", fmt.Errorf("resolve filter: %w", err)
	}
	return value, nil
}
