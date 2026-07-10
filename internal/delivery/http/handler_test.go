package http

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"

	"warehouse06/internal/domain"
	"warehouse06/internal/gitdate"
	"warehouse06/internal/repository"
	"warehouse06/internal/sync"
)

func newTestHandler(t *testing.T) (*Handler, *repository.Holder, *sync.Status) {
	t.Helper()
	repo, err := repository.NewSQLiteRepository(":memory:", zap.NewNop())
	require.NoError(t, err)
	t.Cleanup(func() { _ = repo.Close() })
	holder := repository.NewHolder(repo, ":memory:")
	status := sync.NewStatus()
	return NewHandler(holder, status, zap.NewNop()), holder, status
}

func serve(t *testing.T, h *Handler, method, target string) *httptest.ResponseRecorder {
	t.Helper()
	r := chi.NewRouter()
	h.RegisterRoutes(r)
	req := httptest.NewRequest(method, target, nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	return w
}

func TestHandler_GetStatus(t *testing.T) {
	h, _, status := newTestHandler(t)
	syncedAt := time.Date(2026, 5, 18, 12, 0, 0, 0, time.UTC)
	status.SetSuccess(syncedAt, gitdate.Commit{
		Hash:        "abc123def456",
		CommittedAt: time.Date(2026, 5, 17, 10, 0, 0, 0, time.UTC),
		Subject:     "Update archive",
	}, true)

	w := serve(t, h, http.MethodGet, "/api/status")
	require.Equal(t, http.StatusOK, w.Code)

	var resp domain.SyncStatus
	require.NoError(t, json.NewDecoder(w.Body).Decode(&resp))
	assert.False(t, resp.Syncing)
	require.NotNil(t, resp.LastSyncedAt)
	assert.True(t, syncedAt.Equal(*resp.LastSyncedAt))
	require.NotNil(t, resp.StorageCommit)
	assert.Equal(t, "abc123def456", resp.StorageCommit.Hash)
	assert.Equal(t, "Update archive", resp.StorageCommit.Subject)
}

func TestHandler_GetStatus_Syncing(t *testing.T) {
	h, _, status := newTestHandler(t)
	status.SetSyncing(true)

	w := serve(t, h, http.MethodGet, "/api/status")
	require.Equal(t, http.StatusOK, w.Code)

	var resp domain.SyncStatus
	require.NoError(t, json.NewDecoder(w.Body).Decode(&resp))
	assert.True(t, resp.Syncing)
}

func TestHandler_SearchEntries_MissingQuery(t *testing.T) {
	h, _, _ := newTestHandler(t)
	w := serve(t, h, http.MethodGet, "/api/entries/search")

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestHandler_SearchEntries_EmptyQuery(t *testing.T) {
	h, _, _ := newTestHandler(t)
	w := serve(t, h, http.MethodGet, "/api/entries/search?q=%20%20")

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestHandler_GetEntries_Empty(t *testing.T) {
	h, _, _ := newTestHandler(t)
	w := serve(t, h, http.MethodGet, "/api/entries")

	require.Equal(t, http.StatusOK, w.Code)

	var result domain.EntryListResult
	require.NoError(t, json.NewDecoder(w.Body).Decode(&result))
	assert.Empty(t, result.Items)
	assert.Equal(t, 0, result.Total)
}

func TestHandler_ListTags_Empty(t *testing.T) {
	h, _, _ := newTestHandler(t)
	w := serve(t, h, http.MethodGet, "/api/tags")

	require.Equal(t, http.StatusOK, w.Code)

	var tags []domain.Tag
	require.NoError(t, json.NewDecoder(w.Body).Decode(&tags))
	assert.Empty(t, tags)
}

func TestParseListLimit_capsAtMax(t *testing.T) {
	assert.Equal(t, 100, parseListLimit("500"))
	assert.Equal(t, 50, parseListLimit(""))
	assert.Equal(t, 25, parseListLimit("25"))
}

func TestHandler_SearchEntries_WithData(t *testing.T) {
	h, holder, _ := newTestHandler(t)
	require.NoError(t, holder.Get().SaveEntriesAndAuthors(t.Context(), []*domain.Entry{
		{
			Path:        "bk/demo",
			Name:        "Demo",
			Description: "test entry",
			ContentHTML: "<p>hello</p>",
			Type:        domain.EntryTypeDirectory,
		},
	}, nil))

	w := serve(t, h, http.MethodGet, "/api/entries/search?q=demo")
	require.Equal(t, http.StatusOK, w.Code)

	var result domain.EntryListResult
	require.NoError(t, json.NewDecoder(w.Body).Decode(&result))
	require.Len(t, result.Items, 1)
	assert.Equal(t, "Demo", result.Items[0].Name)
	assert.Equal(t, 1, result.Total)
}

func seedHandlerCatalog(t *testing.T, holder *repository.Holder) {
	t.Helper()
	require.NoError(t, holder.Get().SaveEntriesAndAuthors(t.Context(), []*domain.Entry{
		{
			Path:        "vector06c",
			Name:        "Vector-06C",
			Description: "Platform",
			ContentHTML: "<p>platform</p>",
			Type:        domain.EntryTypeDirectory,
		},
		{
			Path:        "vector06c/demo",
			Name:        "Demo",
			Description: "demo entry",
			ContentHTML: "<p>demo searchable text</p>",
			Type:        domain.EntryTypeDirectory,
			Tags:        []domain.Tag{{Name: "demo"}},
			Authors:     []domain.Author{{DirectoryName: "alice"}},
		},
		{
			Path:        "bk/other",
			Name:        "Other",
			Description: "other",
			ContentHTML: "<p>other</p>",
			Type:        domain.EntryTypeDirectory,
		},
	}, []*domain.Author{
		{DirectoryName: "alice", Name: "Alice", Address: "City", ContentHTML: "<p>bio</p>"},
	}))
}

func TestIsRequestCanceled(t *testing.T) {
	assert.True(t, isRequestCanceled(context.Canceled))
	assert.True(t, isRequestCanceled(context.DeadlineExceeded))
	assert.False(t, isRequestCanceled(errors.New("other")))
	assert.False(t, isRequestCanceled(nil))
}

func TestHandler_GetEntry(t *testing.T) {
	h, holder, _ := newTestHandler(t)
	seedHandlerCatalog(t, holder)

	w := serve(t, h, http.MethodGet, "/api/entries/vector06c%2Fdemo")
	require.Equal(t, http.StatusOK, w.Code)

	var entry domain.Entry
	require.NoError(t, json.NewDecoder(w.Body).Decode(&entry))
	assert.Equal(t, "Demo", entry.Name)
}

func TestHandler_GetEntry_notFound(t *testing.T) {
	h, _, _ := newTestHandler(t)
	w := serve(t, h, http.MethodGet, "/api/entries/missing/path")
	assert.Equal(t, http.StatusNotFound, w.Code)
}

func TestHandler_GetEntry_pathWithDots(t *testing.T) {
	h, holder, _ := newTestHandler(t)
	require.NoError(t, holder.Get().SaveEntriesAndAuthors(t.Context(), []*domain.Entry{
		{
			Path:        "bk/rev..2",
			Name:        "Dotted",
			Description: "entry with dots in name",
			ContentHTML: "<p>dots</p>",
			Type:        domain.EntryTypeDirectory,
		},
	}, nil))

	// ".." is legal in an entry name: the value is a SQL key, not a
	// filesystem path.
	w := serve(t, h, http.MethodGet, "/api/entries/bk%2Frev..2")
	require.Equal(t, http.StatusOK, w.Code)

	// Unknown paths containing ".." simply do not resolve.
	w = serve(t, h, http.MethodGet, "/api/entries/foo%2F..%2Fbar")
	assert.Equal(t, http.StatusNotFound, w.Code)
}

func TestHandler_GetAuthor(t *testing.T) {
	h, holder, _ := newTestHandler(t)
	seedHandlerCatalog(t, holder)

	w := serve(t, h, http.MethodGet, "/api/authors/alice")
	require.Equal(t, http.StatusOK, w.Code)

	var author domain.Author
	require.NoError(t, json.NewDecoder(w.Body).Decode(&author))
	assert.Equal(t, "Alice", author.Name)
}

func TestHandler_GetAuthor_notFound(t *testing.T) {
	h, _, _ := newTestHandler(t)
	w := serve(t, h, http.MethodGet, "/api/authors/nobody")
	assert.Equal(t, http.StatusNotFound, w.Code)
}

func TestHandler_GetAuthor_badDir(t *testing.T) {
	h, _, _ := newTestHandler(t)
	w := serve(t, h, http.MethodGet, "/api/authors/bad%20dir")
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestHandler_ListAuthors(t *testing.T) {
	h, holder, _ := newTestHandler(t)
	seedHandlerCatalog(t, holder)

	w := serve(t, h, http.MethodGet, "/api/authors")
	require.Equal(t, http.StatusOK, w.Code)

	var authors []*domain.Author
	require.NoError(t, json.NewDecoder(w.Body).Decode(&authors))
	require.Len(t, authors, 1)
}

func TestHandler_ListPlatforms(t *testing.T) {
	h, holder, _ := newTestHandler(t)
	seedHandlerCatalog(t, holder)

	w := serve(t, h, http.MethodGet, "/api/platforms")
	require.Equal(t, http.StatusOK, w.Code)

	var platforms []*domain.Platform
	require.NoError(t, json.NewDecoder(w.Body).Decode(&platforms))
	assert.NotEmpty(t, platforms)
}

func TestHandler_GetEntries_filters(t *testing.T) {
	h, holder, _ := newTestHandler(t)
	seedHandlerCatalog(t, holder)

	w := serve(t, h, http.MethodGet, "/api/entries?tag=demo")
	require.Equal(t, http.StatusOK, w.Code)
	var byTag domain.EntryListResult
	require.NoError(t, json.NewDecoder(w.Body).Decode(&byTag))
	require.Len(t, byTag.Items, 1)

	w = serve(t, h, http.MethodGet, "/api/entries?author=alice")
	require.Equal(t, http.StatusOK, w.Code)
	var byAuthor domain.EntryListResult
	require.NoError(t, json.NewDecoder(w.Body).Decode(&byAuthor))
	require.Len(t, byAuthor.Items, 1)

	w = serve(t, h, http.MethodGet, "/api/entries?platform=vector06c")
	require.Equal(t, http.StatusOK, w.Code)
	var byPlatform domain.EntryListResult
	require.NoError(t, json.NewDecoder(w.Body).Decode(&byPlatform))
	assert.GreaterOrEqual(t, byPlatform.Total, 1)

	w = serve(t, h, http.MethodGet, "/api/entries?limit=1")
	require.Equal(t, http.StatusOK, w.Code)
	var limited domain.EntryListResult
	require.NoError(t, json.NewDecoder(w.Body).Decode(&limited))
	require.Len(t, limited.Items, 1)
}

func TestHandler_SearchEntries_invalidQuery(t *testing.T) {
	h, _, _ := newTestHandler(t)
	w := serve(t, h, http.MethodGet, "/api/entries/search?q=%40%40%40")
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestHandler_SearchEntries_withTotal(t *testing.T) {
	h, holder, _ := newTestHandler(t)
	seedHandlerCatalog(t, holder)

	w := serve(t, h, http.MethodGet, "/api/entries/search?q=searchable")
	require.Equal(t, http.StatusOK, w.Code)

	var result domain.EntryListResult
	require.NoError(t, json.NewDecoder(w.Body).Decode(&result))
	assert.GreaterOrEqual(t, result.Total, 1)
}

func TestHandler_ListTags_nonempty(t *testing.T) {
	h, holder, _ := newTestHandler(t)
	seedHandlerCatalog(t, holder)

	w := serve(t, h, http.MethodGet, "/api/tags")
	require.Equal(t, http.StatusOK, w.Code)

	var tags []domain.Tag
	require.NoError(t, json.NewDecoder(w.Body).Decode(&tags))
	assert.NotEmpty(t, tags)
}
