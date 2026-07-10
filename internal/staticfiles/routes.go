package staticfiles

import (
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/go-chi/chi/v5"
)

// RoutesConfig holds filesystem roots for static route registration.
type RoutesConfig struct {
	StorageDir  string
	WorkDir     string
	FrontendDir string
}

// RegisterRoutes mounts emulator, frontend assets, storage files, and SPA fallback.
func RegisterRoutes(r chi.Router, cfg RoutesConfig) {
	workDir := cfg.WorkDir
	if workDir == "" {
		workDir, _ = os.Getwd()
	}

	storagePath := cfg.StorageDir
	frontendPath := cfg.FrontendDir
	if frontendPath == "" {
		frontendPath = filepath.Join(workDir, "frontend", "dist")
	}

	emulatorDir := EmulatorDir(workDir)
	emulatorFS := http.FileServer(http.Dir(emulatorDir))
	r.Get("/emulator", func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, "/emulator/", http.StatusPermanentRedirect)
	})
	r.Get("/emulator/*", func(w http.ResponseWriter, r *http.Request) {
		SetFrontendCacheHeaders(w, r.URL.Path)
		http.StripPrefix("/emulator", emulatorFS).ServeHTTP(w, r)
	})

	assetsDir := filepath.Join(frontendPath, "assets")
	r.Get("/assets/*", func(w http.ResponseWriter, r *http.Request) {
		baseName := filepath.Base(r.URL.Path)
		resolved, ok := ResolveLegacyAsset(assetsDir, baseName)
		if !ok && strings.HasPrefix(baseName, "index-") && strings.HasSuffix(baseName, ".js") {
			if current, found := CurrentIndexAsset(frontendPath); found {
				resolved, ok = current, true
			}
		}
		if !ok {
			http.NotFound(w, r)
			return
		}
		info, err := os.Stat(resolved)
		if err != nil || info.IsDir() {
			http.NotFound(w, r)
			return
		}
		SetFrontendCacheHeaders(w, r.URL.Path)
		ServeFile(w, r, resolved, info)
	})

	r.Get("/*", func(w http.ResponseWriter, r *http.Request) {
		urlPath := r.URL.Path
		isStorageExt := IsStorageAssetPath(urlPath)

		if info, resolved, err := StatUnderRoot(storagePath, urlPath); err == nil {
			if AllowedStorageFile(urlPath, info) {
				ServeFile(w, r, resolved, info)
				return
			}
		}

		if info, resolved, err := StatUnderRoot(frontendPath, urlPath); err == nil {
			if !info.IsDir() {
				ServeFile(w, r, resolved, info)
				return
			}
			indexPath := filepath.Join(resolved, "index.html")
			if indexInfo, indexErr := os.Stat(indexPath); indexErr == nil && !indexInfo.IsDir() {
				http.ServeFile(w, r, indexPath)
				return
			}
		}

		if isStorageExt {
			http.NotFound(w, r)
			return
		}

		SetFrontendCacheHeaders(w, "/index.html")
		http.ServeFile(w, r, filepath.Join(frontendPath, "index.html"))
	})
}
