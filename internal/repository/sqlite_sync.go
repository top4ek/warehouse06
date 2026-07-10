package repository

import (
	"context"
	"fmt"

	"warehouse06/internal/domain"
)

func (r *SQLiteRepository) SaveEntriesAndAuthors(ctx context.Context, entries []*domain.Entry, authors []*domain.Author) error {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	// Statements reused across the rebuild loops are prepared once instead of
	// re-parsing the same SQL for every entry. The no-op DO UPDATE (instead of
	// OR IGNORE) makes RETURNING yield the id on the conflict path as well.
	upsertAuthor, err := tx.PrepareContext(ctx,
		`INSERT INTO authors (directory_name, name, address, content_html)
		 VALUES (?, ?, ?, ?)
		 ON CONFLICT(directory_name) DO UPDATE SET
		 name=excluded.name, address=excluded.address, content_html=excluded.content_html
		 RETURNING id`)
	if err != nil {
		return fmt.Errorf("prepare author upsert: %w", err)
	}
	defer upsertAuthor.Close()

	upsertEntry, err := tx.PrepareContext(ctx,
		`INSERT INTO entries (path, name, description, content_html, date, type, youtube, created_at)
		 VALUES (?, ?, ?, ?, ?, ?, ?, COALESCE(?, CURRENT_TIMESTAMP))
		 ON CONFLICT(path) DO UPDATE SET
		 name=excluded.name, description=excluded.description, content_html=excluded.content_html,
		 date=excluded.date, type=excluded.type, youtube=excluded.youtube,
		 created_at=COALESCE(excluded.created_at, entries.created_at)
		 RETURNING id`)
	if err != nil {
		return fmt.Errorf("prepare entry upsert: %w", err)
	}
	defer upsertEntry.Close()

	upsertTag, err := tx.PrepareContext(ctx,
		`INSERT INTO tags (name) VALUES (?)
		 ON CONFLICT(name) DO UPDATE SET name=excluded.name
		 RETURNING id`)
	if err != nil {
		return fmt.Errorf("prepare tag upsert: %w", err)
	}
	defer upsertTag.Close()

	stubAuthor, err := tx.PrepareContext(ctx,
		`INSERT INTO authors (directory_name, name) VALUES (?, ?)
		 ON CONFLICT(directory_name) DO UPDATE SET directory_name=excluded.directory_name
		 RETURNING id`)
	if err != nil {
		return fmt.Errorf("prepare stub author upsert: %w", err)
	}
	defer stubAuthor.Close()

	linkTag, err := tx.PrepareContext(ctx,
		`INSERT OR IGNORE INTO entry_tags (entry_id, tag_id) VALUES (?, ?)`)
	if err != nil {
		return fmt.Errorf("prepare tag link: %w", err)
	}
	defer linkTag.Close()

	linkAuthor, err := tx.PrepareContext(ctx,
		`INSERT OR IGNORE INTO entry_authors (entry_id, author_id) VALUES (?, ?)`)
	if err != nil {
		return fmt.Errorf("prepare author link: %w", err)
	}
	defer linkAuthor.Close()

	insRequire, err := tx.PrepareContext(ctx,
		`INSERT OR IGNORE INTO entry_requires (entry_id, required_path) VALUES (?, ?)`)
	if err != nil {
		return fmt.Errorf("prepare require insert: %w", err)
	}
	defer insRequire.Close()

	insFile, err := tx.PrepareContext(ctx,
		`INSERT INTO files (entry_id, filename, filepath, is_image) VALUES (?, ?, ?, ?)`)
	if err != nil {
		return fmt.Errorf("prepare file insert: %w", err)
	}
	defer insFile.Close()

	authorIDMap := make(map[string]int64)
	for _, a := range authors {
		var id int64
		if err := upsertAuthor.QueryRowContext(ctx,
			a.DirectoryName, a.Name, a.Address, a.ContentHTML).Scan(&id); err != nil {
			return fmt.Errorf("failed to insert author %s: %w", a.DirectoryName, err)
		}
		authorIDMap[a.DirectoryName] = id
	}

	for _, e := range entries {
		var createdAt any
		if !e.CreatedAt.IsZero() {
			createdAt = e.CreatedAt
		}
		var entryID int64
		if err := upsertEntry.QueryRowContext(ctx,
			e.Path, e.Name, e.Description, e.ContentHTML, e.Date, e.Type, e.Youtube, createdAt).Scan(&entryID); err != nil {
			return fmt.Errorf("failed to insert entry %s: %w", e.Path, err)
		}

		for _, table := range []string{"entry_tags", "entry_authors", "entry_requires", "files"} {
			if _, err := tx.ExecContext(ctx, "DELETE FROM "+table+" WHERE entry_id = ?", entryID); err != nil {
				return fmt.Errorf("clear relations for entry %s: %w", e.Path, err)
			}
		}

		for _, t := range e.Tags {
			var tagID int64
			if err := upsertTag.QueryRowContext(ctx, t.Name).Scan(&tagID); err != nil {
				return fmt.Errorf("resolve tag id %q: %w", t.Name, err)
			}
			if _, err := linkTag.ExecContext(ctx, entryID, tagID); err != nil {
				return err
			}
		}

		for _, a := range e.Authors {
			authorID, ok := authorIDMap[a.DirectoryName]
			if !ok {
				if err := stubAuthor.QueryRowContext(ctx, a.DirectoryName, a.DirectoryName).Scan(&authorID); err != nil {
					return fmt.Errorf("insert stub author %s: %w", a.DirectoryName, err)
				}
				authorIDMap[a.DirectoryName] = authorID
			}
			if _, err := linkAuthor.ExecContext(ctx, entryID, authorID); err != nil {
				return err
			}
		}

		for _, req := range e.Requires {
			if _, err := insRequire.ExecContext(ctx, entryID, req); err != nil {
				return err
			}
		}

		allFiles := append(e.Screenshots, e.Files...)
		for _, f := range allFiles {
			if _, err := insFile.ExecContext(ctx, entryID, f.Filename, f.Filepath, f.IsImage); err != nil {
				return err
			}
		}
	}

	return tx.Commit()
}
