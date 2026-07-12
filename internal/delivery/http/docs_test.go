package http

import (
	"net/http"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v3"
)

func TestHandler_OpenAPISpec_ServesParsableYAML(t *testing.T) {
	h, _, _ := newTestHandler(t)
	w := serve(t, h, http.MethodGet, "/api/openapi.yaml")

	require.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Header().Get("Content-Type"), "yaml")

	var spec map[string]any
	require.NoError(t, yaml.Unmarshal(w.Body.Bytes(), &spec))
	assert.Contains(t, spec, "openapi")
	assert.Contains(t, spec, "info")
	require.Contains(t, spec, "paths")

	paths, ok := spec["paths"].(map[string]any)
	require.True(t, ok)
	for _, p := range []string{
		"/api/status",
		"/api/entries",
		"/api/entries/search",
		"/api/entries/{path}",
		"/api/authors",
		"/api/authors/{dir}",
		"/api/tags",
		"/api/platforms",
	} {
		assert.Contains(t, paths, p)
	}
}

func TestHandler_OpenAPISpec_ContainsAllSchemas(t *testing.T) {
	h, _, _ := newTestHandler(t)
	w := serve(t, h, http.MethodGet, "/api/openapi.yaml")
	require.Equal(t, http.StatusOK, w.Code)

	var spec map[string]any
	require.NoError(t, yaml.Unmarshal(w.Body.Bytes(), &spec))

	components, ok := spec["components"].(map[string]any)
	require.True(t, ok)
	schemas, ok := components["schemas"].(map[string]any)
	require.True(t, ok)

	for _, name := range []string{
		"Entry",
		"EntryListResult",
		"Author",
		"Tag",
		"Platform",
		"SyncStatus",
		"StorageCommit",
		"File",
		"Directory",
		"Error",
	} {
		assert.Contains(t, schemas, name)
	}
}

func TestHandler_DocsIndex_ServesHTML(t *testing.T) {
	h, _, _ := newTestHandler(t)
	w := serve(t, h, http.MethodGet, "/api/docs")

	require.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Header().Get("Content-Type"), "text/html")
	body := w.Body.String()
	assert.Contains(t, body, `id="swagger-ui"`)
	assert.Contains(t, body, "/api/docs/init.js")
	assert.NotContains(t, body, "<script>SwaggerUIBundle")
}

func TestHandler_DocsInitJS_ReferencesSpecURL(t *testing.T) {
	h, _, _ := newTestHandler(t)
	w := serve(t, h, http.MethodGet, "/api/docs/init.js")

	require.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), "/api/openapi.yaml")
}

func TestHandler_DocsAssets_ServesSwaggerUIBundle(t *testing.T) {
	h, _, _ := newTestHandler(t)
	w := serve(t, h, http.MethodGet, "/api/docs/assets/swagger-ui-bundle.js")

	require.Equal(t, http.StatusOK, w.Code)
	assert.True(t, strings.Contains(w.Header().Get("Content-Type"), "javascript"))
	assert.NotEmpty(t, w.Body.Bytes())
}
