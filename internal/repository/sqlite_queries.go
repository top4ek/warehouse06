package repository

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"warehouse06/internal/domain"
	"warehouse06/internal/parser"
)

type EntryListOptions struct {
	Limit    int
	Offset   int
	Sort     string
	Order    string
	Tag      string
	Author   string
	Platform string
}

func (r *SQLiteRepository) CountSearchEntries(ctx context.Context, query string) (int, error) {
	sqlQuery := `
		SELECT COUNT(*)
		FROM entries e
		JOIN entries_fts fts ON e.id = fts.rowid
		WHERE entries_fts MATCH ?
	`
	var total int
	if err := r.db.QueryRowContext(ctx, sqlQuery, query).Scan(&total); err != nil {
		return 0, fmt.Errorf("count search failed: %w", err)
	}
	return total, nil
}

func (r *SQLiteRepository) SearchEntries(ctx context.Context, query string, limit, offset int) ([]*domain.Entry, error) {
	// Simple FTS search
	sqlQuery := `
		SELECT e.id, e.path, e.name, e.description, e.date, e.type, e.youtube, e.created_at
		FROM entries e
		JOIN entries_fts fts ON e.id = fts.rowid
		WHERE entries_fts MATCH ?
		ORDER BY rank
		LIMIT ? OFFSET ?
	`

	entries := make([]*domain.Entry, 0)
	err := r.forEachRow(ctx, sqlQuery, []any{query, limit, offset}, func(rows *sql.Rows) error {
		e, err := scanEntrySummary(rows)
		if err != nil {
			return err
		}
		entries = append(entries, e)
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("search query failed: %w", err)
	}

	for _, e := range entries {
		enrichEntry(e)
	}
	if err := r.populateEntriesListRelations(ctx, entries); err != nil {
		return nil, err
	}
	decorateListEntries(entries)

	return entries, nil
}

func (r *SQLiteRepository) buildEntriesQuery(opts EntryListOptions, countOnly bool) (string, []interface{}) {
	var joins []string
	var where []string
	var args []interface{}

	if opts.Tag != "" {
		joins = append(joins, "JOIN entry_tags et_filter ON e.id = et_filter.entry_id", "JOIN tags t_filter ON et_filter.tag_id = t_filter.id")
		where = append(where, "t_filter.name = ?")
		args = append(args, opts.Tag)
	}
	if opts.Author != "" {
		joins = append(joins, "JOIN entry_authors ea_filter ON e.id = ea_filter.entry_id", "JOIN authors a_filter ON ea_filter.author_id = a_filter.id")
		where = append(where, "a_filter.directory_name = ?")
		args = append(args, opts.Author)
	}
	if opts.Platform != "" {
		where = append(where, "(e.path = ? OR e.path LIKE ? ESCAPE '\\')")
		args = append(args, opts.Platform, escapeLike(opts.Platform)+"/%")
	}

	selectClause := "SELECT COUNT(DISTINCT e.id)"
	if !countOnly {
		selectClause = "SELECT DISTINCT e.id, e.path, e.name, e.description, e.date, e.type, e.youtube, e.created_at"
	}

	sqlQuery := selectClause + `
		FROM entries e
		` + strings.Join(joins, "\n")
	if len(where) > 0 {
		sqlQuery += " WHERE " + strings.Join(where, " AND ")
	}
	return sqlQuery, args
}

func entryOrderClause(sort, order string) string {
	nameTie := "COALESCE(NULLIF(e.name, ''), e.path) ASC"
	dir := "ASC"
	if strings.EqualFold(order, "desc") {
		dir = "DESC"
	}

	switch sort {
	case "rnd", "random":
		return "RANDOM()"
	case "date":
		return fmt.Sprintf("e.date %s, %s", dir, nameTie)
	case "created_at", "created", "added_at", "added":
		return fmt.Sprintf("e.created_at %s, %s", dir, nameTie)
	case "name", "":
		return fmt.Sprintf("COALESCE(NULLIF(e.name, ''), e.path) %s", dir)
	default:
		return fmt.Sprintf("COALESCE(NULLIF(e.name, ''), e.path) %s", dir)
	}
}

func (r *SQLiteRepository) CountEntries(ctx context.Context, opts EntryListOptions) (int, error) {
	sqlQuery, args := r.buildEntriesQuery(opts, true)
	var total int
	if err := r.db.QueryRowContext(ctx, sqlQuery, args...).Scan(&total); err != nil {
		return 0, fmt.Errorf("count entries failed: %w", err)
	}
	return total, nil
}

func (r *SQLiteRepository) GetEntries(ctx context.Context, opts EntryListOptions) ([]*domain.Entry, error) {
	if opts.Limit <= 0 {
		opts.Limit = 50
	}
	if opts.Offset < 0 {
		opts.Offset = 0
	}

	orderBy := entryOrderClause(opts.Sort, opts.Order)

	sqlQuery, args := r.buildEntriesQuery(opts, false)
	sqlQuery += fmt.Sprintf(`
		ORDER BY %s
		LIMIT ? OFFSET ?
	`, orderBy)
	args = append(args, opts.Limit, opts.Offset)

	entries := make([]*domain.Entry, 0)
	err := r.forEachRow(ctx, sqlQuery, args, func(rows *sql.Rows) error {
		e, err := scanEntrySummary(rows)
		if err != nil {
			return err
		}
		entries = append(entries, e)
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("get entries query failed: %w", err)
	}

	for _, e := range entries {
		enrichEntry(e)
	}
	if err := r.populateEntriesListRelations(ctx, entries); err != nil {
		return nil, err
	}
	decorateListEntries(entries)

	return entries, nil
}

func decorateListEntries(entries []*domain.Entry) {
	for _, e := range entries {
		if e == nil {
			continue
		}
		e.DescriptionHTML = parser.RenderDescription(e.Description, e.Path)
	}
}

func scanEntrySummary(rows interface {
	Scan(dest ...any) error
}) (*domain.Entry, error) {
	var e domain.Entry
	var createdAt sql.NullTime
	if err := rows.Scan(
		&e.ID, &e.Path, &e.Name, &e.Description, &e.Date, &e.Type, &e.Youtube, &createdAt,
	); err != nil {
		return nil, err
	}
	if createdAt.Valid {
		e.CreatedAt = createdAt.Time
	}
	return &e, nil
}

func (r *SQLiteRepository) GetEntryByPath(ctx context.Context, path string) (*domain.Entry, error) {
	sqlQuery := `
		SELECT id, path, name, description, content_html, date, type, youtube, COALESCE(controls, ''), created_at
		FROM entries
		WHERE path = ?
	`

	var e domain.Entry
	var createdAt sql.NullTime
	var controls string
	err := r.db.QueryRowContext(ctx, sqlQuery, path).Scan(
		&e.ID, &e.Path, &e.Name, &e.Description, &e.ContentHTML, &e.Date, &e.Type, &e.Youtube, &controls, &createdAt,
	)
	if controls != "" {
		e.Controls = json.RawMessage(controls)
	}
	if createdAt.Valid {
		e.CreatedAt = createdAt.Time
	}
	if err != nil {
		return nil, fmt.Errorf("get entry by path failed: %w", err)
	}

	enrichEntry(&e)
	if err := r.populateEntryRelations(ctx, &e); err != nil {
		return nil, err
	}

	return &e, nil
}

func (r *SQLiteRepository) populateEntryRelations(ctx context.Context, e *domain.Entry) error {
	err := r.forEachRow(ctx, `
		SELECT t.id, t.name
		FROM tags t
		JOIN entry_tags et ON t.id = et.tag_id
		WHERE et.entry_id = ?`, []any{e.ID}, func(rows *sql.Rows) error {
		var t domain.Tag
		if err := rows.Scan(&t.ID, &t.Name); err != nil {
			return err
		}
		e.Tags = append(e.Tags, t)
		return nil
	})
	if err != nil {
		return fmt.Errorf("load tags for entry %d: %w", e.ID, err)
	}

	err = r.forEachRow(ctx, `
		SELECT a.id, a.directory_name, COALESCE(a.name, '')
		FROM authors a
		JOIN entry_authors ea ON a.id = ea.author_id
		WHERE ea.entry_id = ?`, []any{e.ID}, func(rows *sql.Rows) error {
		var a domain.Author
		if err := rows.Scan(&a.ID, &a.DirectoryName, &a.Name); err != nil {
			return err
		}
		e.Authors = append(e.Authors, a)
		return nil
	})
	if err != nil {
		return fmt.Errorf("load authors for entry %d: %w", e.ID, err)
	}

	err = r.forEachRow(ctx, `
		SELECT id, filename, filepath, is_image
		FROM files
		WHERE entry_id = ?`, []any{e.ID}, func(rows *sql.Rows) error {
		var f domain.File
		if err := rows.Scan(&f.ID, &f.Filename, &f.Filepath, &f.IsImage); err != nil {
			return err
		}
		e.Files = append(e.Files, f)
		if f.IsImage && e.PreviewImage == "" {
			e.PreviewImage = f.Filepath
		}
		return nil
	})
	if err != nil {
		return fmt.Errorf("load files for entry %d: %w", e.ID, err)
	}

	err = r.forEachRow(ctx, `
		SELECT required_path
		FROM entry_requires
		WHERE entry_id = ?`, []any{e.ID}, func(rows *sql.Rows) error {
		var req string
		if err := rows.Scan(&req); err != nil {
			return err
		}
		e.Requires = append(e.Requires, req)
		return nil
	})
	if err != nil {
		return fmt.Errorf("load requires for entry %d: %w", e.ID, err)
	}

	prefix := e.Path
	if prefix != "" {
		prefix += "/"
	}
	seen := make(map[string]bool)
	err = r.forEachRow(ctx, `
		SELECT path, COALESCE(NULLIF(name, ''), path)
		FROM entries
		WHERE path LIKE ? ESCAPE '\' AND path != ?
		ORDER BY COALESCE(NULLIF(name, ''), path) ASC`, []any{escapeLike(prefix) + "%", e.Path}, func(rows *sql.Rows) error {
		var childPath, childName string
		if err := rows.Scan(&childPath, &childName); err != nil {
			return err
		}
		rest := strings.TrimPrefix(childPath, prefix)
		if rest == "" || strings.Contains(rest, "/") {
			return nil
		}
		if !seen[childPath] {
			e.Directories = append(e.Directories, domain.Directory{Name: childName, Path: childPath})
			seen[childPath] = true
		}
		return nil
	})
	if err != nil {
		return fmt.Errorf("load directories for entry %d: %w", e.ID, err)
	}

	return nil
}

func (r *SQLiteRepository) populateAuthorPreview(ctx context.Context, a *domain.Author) error {
	if err := r.db.QueryRowContext(ctx, `
		SELECT COUNT(DISTINCT e.id)
		FROM entries e
		JOIN entry_authors ea ON e.id = ea.entry_id
		WHERE ea.author_id = ?`, a.ID).Scan(&a.EntryCount); err != nil {
		return err
	}

	var previewPath string
	err := r.db.QueryRowContext(ctx, `
		SELECT f.filepath
		FROM files f
		JOIN entries e ON f.entry_id = e.id
		JOIN entry_authors ea ON e.id = ea.entry_id
		WHERE ea.author_id = ? AND f.is_image = 1
		ORDER BY e.path ASC
		LIMIT 1`, a.ID).Scan(&previewPath)
	switch {
	case err == nil:
		a.PreviewImage = previewPath
	case !errors.Is(err, sql.ErrNoRows):
		// No preview is fine; a failing query is not.
		return err
	}

	if a.PreviewImage == "" {
		authorEntryPath := "authors/" + a.DirectoryName
		// Best-effort fallback; a missing preview image is not an error.
		_ = r.forEachRow(ctx, `
			SELECT filepath FROM files
			WHERE entry_id = (SELECT id FROM entries WHERE path = ? LIMIT 1) AND is_image = 1
			LIMIT 1`, []any{authorEntryPath}, func(rows *sql.Rows) error {
			return rows.Scan(&a.PreviewImage)
		})
	}
	return nil
}

func (r *SQLiteRepository) GetAuthorByDir(ctx context.Context, dir string) (*domain.Author, error) {
	sqlQuery := `
		SELECT id, directory_name, COALESCE(name, ''), COALESCE(address, ''), COALESCE(content_html, '')
		FROM authors
		WHERE directory_name = ?
	`

	var a domain.Author
	err := r.db.QueryRowContext(ctx, sqlQuery, dir).Scan(
		&a.ID, &a.DirectoryName, &a.Name, &a.Address, &a.ContentHTML,
	)
	if err != nil {
		return nil, fmt.Errorf("get author by dir failed: %w", err)
	}

	if err := r.populateAuthorPreview(ctx, &a); err != nil {
		return nil, err
	}

	return &a, nil
}

func (r *SQLiteRepository) ListAuthors(ctx context.Context) ([]*domain.Author, error) {
	authors := make([]*domain.Author, 0)
	err := r.forEachRow(ctx, `
		SELECT a.id, a.directory_name, COALESCE(a.name, ''), COALESCE(a.address, ''), COUNT(DISTINCT ea.entry_id)
		FROM authors a
		LEFT JOIN entry_authors ea ON ea.author_id = a.id
		GROUP BY a.id, a.directory_name, a.name, a.address
		ORDER BY COUNT(DISTINCT ea.entry_id) DESC, COALESCE(NULLIF(a.name, ''), a.directory_name) ASC`,
		nil, func(rows *sql.Rows) error {
			var a domain.Author
			if err := rows.Scan(&a.ID, &a.DirectoryName, &a.Name, &a.Address, &a.EntryCount); err != nil {
				return err
			}
			authors = append(authors, &a)
			return nil
		})
	if err != nil {
		return nil, fmt.Errorf("list authors failed: %w", err)
	}

	if err := r.populateAuthorPreviews(ctx, authors); err != nil {
		return nil, err
	}
	return authors, nil
}

// populateAuthorPreviews batch-loads one preview image per author (first image
// across the author's entries by path, falling back to an image attached to
// the author's own "authors/<dir>" entry), avoiding a per-author query loop.
func (r *SQLiteRepository) populateAuthorPreviews(ctx context.Context, authors []*domain.Author) error {
	if len(authors) == 0 {
		return nil
	}

	byID := make(map[int64]*domain.Author, len(authors))
	for _, a := range authors {
		byID[a.ID] = a
	}

	err := r.forEachRow(ctx, `
		SELECT author_id, filepath FROM (
			SELECT ea.author_id, f.filepath,
				ROW_NUMBER() OVER (PARTITION BY ea.author_id ORDER BY e.path ASC) AS rn
			FROM files f
			JOIN entries e ON f.entry_id = e.id
			JOIN entry_authors ea ON e.id = ea.entry_id
			WHERE f.is_image = 1
		) WHERE rn = 1`,
		nil, func(rows *sql.Rows) error {
			var authorID int64
			var filepath string
			if err := rows.Scan(&authorID, &filepath); err != nil {
				return err
			}
			if a := byID[authorID]; a != nil {
				a.PreviewImage = filepath
			}
			return nil
		})
	if err != nil {
		return fmt.Errorf("batch author previews: %w", err)
	}

	byDirEntry := make(map[string]*domain.Author)
	for _, a := range authors {
		if a.PreviewImage == "" {
			byDirEntry["authors/"+a.DirectoryName] = a
		}
	}
	if len(byDirEntry) == 0 {
		return nil
	}

	err = r.forEachRow(ctx, `
		SELECT path, filepath FROM (
			SELECT e.path, f.filepath,
				ROW_NUMBER() OVER (PARTITION BY e.path ORDER BY f.id ASC) AS rn
			FROM files f
			JOIN entries e ON f.entry_id = e.id
			WHERE f.is_image = 1 AND e.path LIKE 'authors/%'
		) WHERE rn = 1`,
		nil, func(rows *sql.Rows) error {
			var path, filepath string
			if err := rows.Scan(&path, &filepath); err != nil {
				return err
			}
			if a := byDirEntry[path]; a != nil {
				a.PreviewImage = filepath
			}
			return nil
		})
	if err != nil {
		return fmt.Errorf("batch author fallback previews: %w", err)
	}
	return nil
}

func (r *SQLiteRepository) ListTags(ctx context.Context) ([]*domain.Tag, error) {
	tags := make([]*domain.Tag, 0)
	err := r.forEachRow(ctx, `
		SELECT t.id, t.name, COUNT(et.entry_id) AS entry_count
		FROM tags t
		LEFT JOIN entry_tags et ON et.tag_id = t.id
		GROUP BY t.id, t.name
		ORDER BY entry_count DESC, t.name ASC`,
		nil, func(rows *sql.Rows) error {
			var t domain.Tag
			if err := rows.Scan(&t.ID, &t.Name, &t.EntryCount); err != nil {
				return err
			}
			tags = append(tags, &t)
			return nil
		})
	if err != nil {
		return nil, fmt.Errorf("list tags failed: %w", err)
	}
	return tags, nil
}

func (r *SQLiteRepository) ListPlatforms(ctx context.Context) ([]*domain.Platform, error) {
	platforms := make([]*domain.Platform, 0)
	err := r.forEachRow(ctx, `
		SELECT path, COALESCE(NULLIF(name, ''), path), COALESCE(description, ''), COALESCE(content_html, '')
		FROM entries
		WHERE path != '' AND instr(path, '/') = 0
		ORDER BY COALESCE(NULLIF(name, ''), path) ASC`,
		nil, func(rows *sql.Rows) error {
			var p domain.Platform
			if err := rows.Scan(&p.Path, &p.Name, &p.Description, &p.ContentHTML); err != nil {
				return err
			}
			platforms = append(platforms, &p)
			return nil
		})
	if err != nil {
		return nil, fmt.Errorf("list platforms failed: %w", err)
	}

	counts := make(map[string]int)
	err = r.forEachRow(ctx, `
		SELECT
			substr(e.path, 1, instr(e.path || '/', '/') - 1) AS platform,
			COUNT(*) AS entry_count
		FROM entries e
		WHERE e.path LIKE '%/%'
		GROUP BY platform`,
		nil, func(rows *sql.Rows) error {
			var platform string
			var count int
			if err := rows.Scan(&platform, &count); err != nil {
				return err
			}
			counts[platform] = count
			return nil
		})
	if err != nil {
		return nil, fmt.Errorf("count platform entries: %w", err)
	}

	for _, p := range platforms {
		p.EntryCount = counts[p.Path]
	}
	return platforms, nil
}

func enrichEntry(e *domain.Entry) {
	if e == nil || e.Path == "" {
		return
	}
	parts := strings.SplitN(e.Path, "/", 2)
	e.Platform = parts[0]
}
