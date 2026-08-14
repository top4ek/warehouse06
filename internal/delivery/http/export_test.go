package http

import (
	"net/http"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"

	"warehouse06/internal/repository"
	"warehouse06/internal/sync"
)

func TestHandler_ExportREBUS_notYetAvailable(t *testing.T) {
	h, _, _ := newTestHandler(t)
	w := serve(t, h, http.MethodGet, "/api/export/rebus")
	assert.Equal(t, http.StatusServiceUnavailable, w.Code)
}

func TestHandler_ExportREBUS_servesStaticFile(t *testing.T) {
	repo, err := repository.NewSQLiteRepository(":memory:", zap.NewNop())
	require.NoError(t, err)
	t.Cleanup(func() { _ = repo.Close() })
	holder := repository.NewHolder(repo, ":memory:")
	status := sync.NewStatus()

	exportPath := filepath.Join(t.TempDir(), "rebus-export.zip")
	want := []byte("PK\x03\x04fake-zip-content-for-test")
	require.NoError(t, os.WriteFile(exportPath, want, 0o644))

	h := NewHandler(holder, status, sync.ExportPaths{Rebus: exportPath}, "", zap.NewNop())
	w := serve(t, h, http.MethodGet, "/api/export/rebus")

	require.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, want, w.Body.Bytes())
	assert.Contains(t, w.Header().Get("Content-Disposition"), "warehouse06-rebus-export.zip")
}

func TestHandler_ExportSQLite_notYetAvailable(t *testing.T) {
	h, _, _ := newTestHandler(t)
	w := serve(t, h, http.MethodGet, "/api/export/sqlite")
	assert.Equal(t, http.StatusServiceUnavailable, w.Code)
}

func TestHandler_ExportSQLite_servesStaticFile(t *testing.T) {
	repo, err := repository.NewSQLiteRepository(":memory:", zap.NewNop())
	require.NoError(t, err)
	t.Cleanup(func() { _ = repo.Close() })
	holder := repository.NewHolder(repo, ":memory:")
	status := sync.NewStatus()

	exportPath := filepath.Join(t.TempDir(), "sqlite-export.zip")
	want := []byte("PK\x03\x04fake-zip-content-for-test")
	require.NoError(t, os.WriteFile(exportPath, want, 0o644))

	h := NewHandler(holder, status, sync.ExportPaths{SQLite: exportPath}, "", zap.NewNop())
	w := serve(t, h, http.MethodGet, "/api/export/sqlite")

	require.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, want, w.Body.Bytes())
	assert.Contains(t, w.Header().Get("Content-Disposition"), "warehouse06-sqlite-export.zip")
}

func TestHandler_ExportStorage_notYetAvailable(t *testing.T) {
	h, _, _ := newTestHandler(t)
	w := serve(t, h, http.MethodGet, "/api/export/storage")
	assert.Equal(t, http.StatusServiceUnavailable, w.Code)
}

func TestHandler_ExportStorage_servesStaticFile(t *testing.T) {
	repo, err := repository.NewSQLiteRepository(":memory:", zap.NewNop())
	require.NoError(t, err)
	t.Cleanup(func() { _ = repo.Close() })
	holder := repository.NewHolder(repo, ":memory:")
	status := sync.NewStatus()

	exportPath := filepath.Join(t.TempDir(), "storage-export.zip")
	want := []byte("PK\x03\x04fake-zip-content-for-test")
	require.NoError(t, os.WriteFile(exportPath, want, 0o644))

	h := NewHandler(holder, status, sync.ExportPaths{Storage: exportPath}, "", zap.NewNop())
	w := serve(t, h, http.MethodGet, "/api/export/storage")

	require.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, want, w.Body.Bytes())
	assert.Contains(t, w.Header().Get("Content-Disposition"), "warehouse06-storage-export.zip")
}
