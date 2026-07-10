package staticfiles

import (
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRegisterRoutes_servesSPA(t *testing.T) {
	root := t.TempDir()
	storage := filepath.Join(root, "storage")
	frontend := filepath.Join(root, "frontend", "dist")
	require.NoError(t, os.MkdirAll(storage, 0o755))
	require.NoError(t, os.MkdirAll(frontend, 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(frontend, "index.html"), []byte("<html>app</html>"), 0o644))

	r := chi.NewRouter()
	RegisterRoutes(r, RoutesConfig{
		StorageDir:  storage,
		WorkDir:     root,
		FrontendDir: frontend,
	})

	req := httptest.NewRequest(http.MethodGet, "/browse", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), "app")
}

func TestRegisterRoutes_servesStorageAsset(t *testing.T) {
	root := t.TempDir()
	storage := filepath.Join(root, "storage")
	platform := filepath.Join(storage, "vector06c")
	frontend := filepath.Join(root, "frontend", "dist")
	require.NoError(t, os.MkdirAll(platform, 0o755))
	require.NoError(t, os.MkdirAll(frontend, 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(platform, "game.rom"), []byte("rom"), 0o644))
	require.NoError(t, os.WriteFile(filepath.Join(frontend, "index.html"), []byte("app"), 0o644))

	r := chi.NewRouter()
	RegisterRoutes(r, RoutesConfig{
		StorageDir:  storage,
		WorkDir:     root,
		FrontendDir: frontend,
	})

	req := httptest.NewRequest(http.MethodGet, "/vector06c/game.rom", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, "rom", w.Body.String())
}
