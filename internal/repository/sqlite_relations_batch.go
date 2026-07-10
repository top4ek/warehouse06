package repository

import (
	"context"
	"database/sql"
	"fmt"
	"strings"

	"warehouse06/internal/domain"
)

func entryIDs(entries []*domain.Entry) []any {
	ids := make([]any, len(entries))
	for i, e := range entries {
		ids[i] = e.ID
	}
	return ids
}

func entryByID(entries []*domain.Entry) map[int64]*domain.Entry {
	m := make(map[int64]*domain.Entry, len(entries))
	for _, e := range entries {
		m[e.ID] = e
	}
	return m
}

func placeholders(n int) string {
	if n == 0 {
		return ""
	}
	return strings.TrimRight(strings.Repeat("?,", n), ",")
}

// populateEntriesListRelations batch-loads tags and authors for list/search endpoints.
func (r *SQLiteRepository) populateEntriesListRelations(ctx context.Context, entries []*domain.Entry) error {
	if len(entries) == 0 {
		return nil
	}

	byID := entryByID(entries)
	ids := entryIDs(entries)

	tagQuery := fmt.Sprintf(`
		SELECT et.entry_id, t.id, t.name
		FROM entry_tags et
		JOIN tags t ON t.id = et.tag_id
		WHERE et.entry_id IN (%s)`, placeholders(len(ids)))
	err := r.forEachRow(ctx, tagQuery, ids, func(rows *sql.Rows) error {
		var entryID int64
		var tag domain.Tag
		if err := rows.Scan(&entryID, &tag.ID, &tag.Name); err != nil {
			return err
		}
		if e := byID[entryID]; e != nil {
			e.Tags = append(e.Tags, tag)
		}
		return nil
	})
	if err != nil {
		return fmt.Errorf("batch load tags: %w", err)
	}

	authorQuery := fmt.Sprintf(`
		SELECT ea.entry_id, a.id, a.directory_name, COALESCE(a.name, '')
		FROM entry_authors ea
		JOIN authors a ON a.id = ea.author_id
		WHERE ea.entry_id IN (%s)`, placeholders(len(ids)))
	err = r.forEachRow(ctx, authorQuery, ids, func(rows *sql.Rows) error {
		var entryID int64
		var author domain.Author
		if err := rows.Scan(&entryID, &author.ID, &author.DirectoryName, &author.Name); err != nil {
			return err
		}
		if e := byID[entryID]; e != nil {
			e.Authors = append(e.Authors, author)
		}
		return nil
	})
	if err != nil {
		return fmt.Errorf("batch load authors: %w", err)
	}

	previewQuery := fmt.Sprintf(`
		SELECT entry_id, filepath FROM (
			SELECT entry_id, filepath, ROW_NUMBER() OVER (PARTITION BY entry_id ORDER BY id) AS rn
			FROM files
			WHERE is_image = 1 AND entry_id IN (%s)
		) WHERE rn = 1`, placeholders(len(ids)))
	err = r.forEachRow(ctx, previewQuery, ids, func(rows *sql.Rows) error {
		var entryID int64
		var filepath string
		if err := rows.Scan(&entryID, &filepath); err != nil {
			return err
		}
		if e := byID[entryID]; e != nil {
			e.PreviewImage = filepath
		}
		return nil
	})
	if err != nil {
		return fmt.Errorf("batch load preview images: %w", err)
	}

	return nil
}
