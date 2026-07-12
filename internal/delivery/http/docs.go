package http

import (
	_ "embed"
	"net/http"

	swaggerFiles "github.com/swaggo/files/v2"
)

//go:embed docs/openapi.yaml
var openapiSpecYAML []byte

//go:embed docs/index.html
var swaggerDocsIndexHTML []byte

//go:embed docs/init.js
var swaggerDocsInitJS []byte

var swaggerAssets = http.StripPrefix("/api/docs/assets/", http.FileServer(http.FS(swaggerFiles.FS)))

func (h *Handler) handleGetOpenAPISpec(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/yaml; charset=utf-8")
	_, _ = w.Write(openapiSpecYAML)
}

func (h *Handler) handleGetDocsIndex(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	_, _ = w.Write(swaggerDocsIndexHTML)
}

func (h *Handler) handleGetDocsInitJS(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/javascript; charset=utf-8")
	_, _ = w.Write(swaggerDocsInitJS)
}

func (h *Handler) handleGetDocsAssets(w http.ResponseWriter, r *http.Request) {
	swaggerAssets.ServeHTTP(w, r)
}
