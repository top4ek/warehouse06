package repository

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"

	"warehouse06/internal/domain"
)

func newTestRepo(t *testing.T) *SQLiteRepository {
	t.Helper()
	repo, err := NewSQLiteRepository(":memory:", zap.NewNop())
	require.NoError(t, err)
	t.Cleanup(func() { _ = repo.Close() })
	return repo
}

func TestSQLiteRepository_SearchEntries(t *testing.T) {
	repo := newTestRepo(t)
	ctx := context.Background()

	err := repo.SaveEntriesAndAuthors(ctx, []*domain.Entry{
		{
			Path:        "vector06c/exolon",
			Name:        "Exolon",
			Description: "A shooter on Vector-06C",
			ContentHTML: "<p>exolon game</p>",
			Type:        domain.EntryTypeDirectory,
		},
	}, nil)
	require.NoError(t, err)

	entries, err := repo.SearchEntries(ctx, FormatFTSQuery("exolon"), 10, 0)
	require.NoError(t, err)
	require.Len(t, entries, 1)
	assert.Equal(t, "Exolon", entries[0].Name)

	total, err := repo.CountSearchEntries(ctx, FormatFTSQuery("exolon"))
	require.NoError(t, err)
	assert.Equal(t, 1, total)
}

func TestSQLiteRepository_GetEntryByPath_NotFound(t *testing.T) {
	repo := newTestRepo(t)
	_, err := repo.GetEntryByPath(context.Background(), "missing/path")
	assert.Error(t, err)
}

func seedCatalog(t *testing.T, repo *SQLiteRepository) {
	t.Helper()
	ctx := context.Background()
	err := repo.SaveEntriesAndAuthors(ctx, []*domain.Entry{
		{
			Path:        "vector06c",
			Name:        "Vector-06C",
			Description: "Platform root",
			ContentHTML: "<p>platform</p>",
			Type:        domain.EntryTypeDirectory,
		},
		{
			Path:        "vector06c/demo",
			Name:        "Demo",
			Description: "A demo game on Vector-06C",
			ContentHTML: "<p>demo game content</p>",
			Type:        domain.EntryTypeDirectory,
			Tags:        []domain.Tag{{Name: "demo"}},
			Authors:     []domain.Author{{DirectoryName: "alice"}},
			Files: []domain.File{
				{Filename: "cover.png", Filepath: "vector06c/demo/cover.png", IsImage: true},
			},
		},
		{
			Path:        "vector06c/demo/child",
			Name:        "Child Level",
			Description: "Nested entry",
			ContentHTML: "<p>child</p>",
			Type:        domain.EntryTypeDirectory,
		},
		{
			Path:        "bk",
			Name:        "BK",
			Description: "BK platform",
			ContentHTML: "<p>bk</p>",
			Type:        domain.EntryTypeDirectory,
		},
		{
			Path:        "bk/demo",
			Name:        "BK Demo",
			Description: "Another platform demo",
			ContentHTML: "<p>bk demo</p>",
			Type:        domain.EntryTypeDirectory,
			Tags:        []domain.Tag{{Name: "archive"}},
		},
	}, []*domain.Author{
		{
			DirectoryName: "alice",
			Name:          "Alice",
			Address:       "Somewhere",
			ContentHTML:   "<p>alice author</p>",
		},
	})
	require.NoError(t, err)
}

func TestSQLiteRepository_GetEntryByPath_withRelations(t *testing.T) {
	repo := newTestRepo(t)
	seedCatalog(t, repo)
	ctx := context.Background()

	entry, err := repo.GetEntryByPath(ctx, "vector06c/demo")
	require.NoError(t, err)
	assert.Equal(t, "Demo", entry.Name)
	assert.Equal(t, "vector06c", entry.Platform)
	require.Len(t, entry.Tags, 1)
	assert.Equal(t, "demo", entry.Tags[0].Name)
	require.Len(t, entry.Authors, 1)
	assert.Equal(t, "alice", entry.Authors[0].DirectoryName)
	require.NotEmpty(t, entry.Files)
	require.NotEmpty(t, entry.Directories)
	assert.Equal(t, "vector06c/demo/child", entry.Directories[0].Path)
}

func TestSQLiteRepository_GetEntries_filtersAndPagination(t *testing.T) {
	repo := newTestRepo(t)
	seedCatalog(t, repo)
	ctx := context.Background()

	tagEntries, err := repo.GetEntries(ctx, EntryListOptions{Tag: "demo", Limit: 10})
	require.NoError(t, err)
	require.Len(t, tagEntries, 1)
	assert.Equal(t, "vector06c/demo", tagEntries[0].Path)
	assert.Equal(t, "vector06c/demo/cover.png", tagEntries[0].PreviewImage)
	assert.Contains(t, tagEntries[0].DescriptionHTML, "<p>")

	authorEntries, err := repo.GetEntries(ctx, EntryListOptions{Author: "alice", Limit: 10})
	require.NoError(t, err)
	require.Len(t, authorEntries, 1)

	platformEntries, err := repo.GetEntries(ctx, EntryListOptions{Platform: "vector06c", Limit: 10, Sort: "name", Order: "asc"})
	require.NoError(t, err)
	assert.GreaterOrEqual(t, len(platformEntries), 2)

	total, err := repo.CountEntries(ctx, EntryListOptions{Platform: "vector06c"})
	require.NoError(t, err)
	assert.Equal(t, len(platformEntries), total)

	paged, err := repo.GetEntries(ctx, EntryListOptions{Limit: 1, Offset: 0, Sort: "name"})
	require.NoError(t, err)
	require.Len(t, paged, 1)
}

func TestSQLiteRepository_ListTagsAuthorsPlatforms(t *testing.T) {
	repo := newTestRepo(t)
	seedCatalog(t, repo)
	ctx := context.Background()

	tags, err := repo.ListTags(ctx)
	require.NoError(t, err)
	require.GreaterOrEqual(t, len(tags), 2)

	authors, err := repo.ListAuthors(ctx)
	require.NoError(t, err)
	require.Len(t, authors, 1)
	assert.Equal(t, "alice", authors[0].DirectoryName)

	platforms, err := repo.ListPlatforms(ctx)
	require.NoError(t, err)
	require.GreaterOrEqual(t, len(platforms), 2)

	author, err := repo.GetAuthorByDir(ctx, "alice")
	require.NoError(t, err)
	assert.Equal(t, "Alice", author.Name)
	assert.Equal(t, 1, author.EntryCount)
}

func TestSQLiteRepository_SaveEntriesAndAuthors_roundTripFTS(t *testing.T) {
	repo := newTestRepo(t)
	seedCatalog(t, repo)
	ctx := context.Background()

	entries, err := repo.SearchEntries(ctx, FormatFTSQuery("demo game"), 10, 0)
	require.NoError(t, err)
	require.NotEmpty(t, entries)

	total, err := repo.CountSearchEntries(ctx, FormatFTSQuery("demo"))
	require.NoError(t, err)
	assert.GreaterOrEqual(t, total, 1)
}

func TestSQLiteRepository_ClearAll(t *testing.T) {
	repo := newTestRepo(t)
	seedCatalog(t, repo)
	ctx := context.Background()

	require.NoError(t, repo.ClearAll(ctx))

	count, err := repo.CountEntries(ctx, EntryListOptions{})
	require.NoError(t, err)
	assert.Equal(t, 0, count)

	tags, err := repo.ListTags(ctx)
	require.NoError(t, err)
	assert.Empty(t, tags)
}
