package repository

import (
	"context"
	"database/sql"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"warehouse06/internal/domain"
)

func TestSQLiteRepository_ListAllEntries(t *testing.T) {
	repo := newTestRepo(t)
	seedCatalog(t, repo)
	ctx := context.Background()

	entries, err := repo.ListAllEntries(ctx)
	require.NoError(t, err)
	require.NotEmpty(t, entries)

	var demo *domain.Entry
	for _, e := range entries {
		if e.Path == "vector06c/demo" {
			demo = e
			break
		}
	}
	require.NotNil(t, demo, "seeded entry vector06c/demo not found")

	assert.Equal(t, "<p>demo game content</p>", demo.ContentHTML, "GetEntries never selects content_html; ListAllEntries must")
	assert.Equal(t, "vector06c", demo.Platform)
	require.Len(t, demo.Tags, 1)
	assert.Equal(t, "demo", demo.Tags[0].Name)
	require.Len(t, demo.Authors, 1)
	assert.Equal(t, "alice", demo.Authors[0].DirectoryName)
	require.Len(t, demo.Files, 1)
	assert.Equal(t, "cover.png", demo.Files[0].Filename)
	assert.Equal(t, demo.ID, demo.Files[0].EntryID)
}

func TestSQLiteRepository_ListAllAuthors(t *testing.T) {
	repo := newTestRepo(t)
	seedCatalog(t, repo)
	ctx := context.Background()

	authors, err := repo.ListAllAuthors(ctx)
	require.NoError(t, err)
	require.Len(t, authors, 1)

	alice := authors[0]
	assert.Equal(t, "alice", alice.DirectoryName)
	assert.Equal(t, "Alice", alice.Name)
	assert.Equal(t, "Somewhere", alice.Address)
	assert.Equal(t, "<p>alice author</p>", alice.ContentHTML,
		"ListAuthors never selects content_html; ListAllAuthors must")
}

func TestSQLiteRepository_ListAllEntries_populatesRequires(t *testing.T) {
	repo := newTestRepo(t)
	ctx := context.Background()

	err := repo.SaveEntriesAndAuthors(ctx, []*domain.Entry{
		{
			Path: "vector06c/dependency",
			Name: "Dependency",
			Type: domain.EntryTypeDirectory,
		},
		{
			Path:     "vector06c/needs-dependency",
			Name:     "Needs Dependency",
			Type:     domain.EntryTypeDirectory,
			Requires: []string{"vector06c/dependency"},
		},
	}, nil)
	require.NoError(t, err)

	entries, err := repo.ListAllEntries(ctx)
	require.NoError(t, err)

	var withReq *domain.Entry
	for _, e := range entries {
		if e.Path == "vector06c/needs-dependency" {
			withReq = e
			break
		}
	}
	require.NotNil(t, withReq)
	require.Len(t, withReq.Requires, 1)
	assert.Equal(t, "vector06c/dependency", withReq.Requires[0])
}

func TestSQLiteRepository_SnapshotTo(t *testing.T) {
	repo := newTestRepo(t)
	seedCatalog(t, repo)
	ctx := context.Background()

	path := filepath.Join(t.TempDir(), "snapshot.sqlite")
	require.NoError(t, repo.SnapshotTo(ctx, path))
	// A snapshot left over from a previous sync must not block the next one.
	require.NoError(t, repo.SnapshotTo(ctx, path))

	// Opened raw, not through NewSQLiteRepository: initSchema would recreate
	// the very FTS objects this asserts are absent.
	db, err := sql.Open("sqlite3", path)
	require.NoError(t, err)
	t.Cleanup(func() { _ = db.Close() })

	var name, contentHTML string
	require.NoError(t, db.QueryRowContext(ctx,
		`SELECT name, content_html FROM entries WHERE path = ?`, "vector06c/demo").Scan(&name, &contentHTML))
	assert.Equal(t, "Demo", name)
	assert.Equal(t, "<p>demo game content</p>", contentHTML)

	var relations int
	require.NoError(t, db.QueryRowContext(ctx,
		`SELECT (SELECT count(*) FROM entry_tags) + (SELECT count(*) FROM entry_authors) + (SELECT count(*) FROM files)`).Scan(&relations))
	assert.Positive(t, relations, "relation tables must survive the snapshot")

	var ftsObjects int
	require.NoError(t, db.QueryRowContext(ctx,
		`SELECT count(*) FROM sqlite_master WHERE name LIKE 'entries_fts%' OR name IN ('entries_ai', 'entries_ad', 'entries_au')`).Scan(&ftsObjects))
	assert.Zero(t, ftsObjects, "snapshot must open in SQLite builds without FTS5")

	// The live database keeps its index: the snapshot is a copy, never a move.
	found, err := repo.SearchEntries(ctx, FormatFTSQuery("demo"), 10, 0)
	require.NoError(t, err)
	assert.NotEmpty(t, found)
}
