package staticfiles

import (
	"errors"
	"net/http"
	"os"
	"path/filepath"
	"strings"
)

var ErrOutsideRoot = errors.New("path outside root")

// StorageAssetExtensions lists file types served from the content storage tree.
var StorageAssetExtensions = map[string]bool{
	".rom": true, ".fdd": true, ".zip": true, ".com": true, ".bin": true,
	".r0m": true, ".png": true, ".jpg": true, ".jpeg": true, ".gif": true,
	".webp": true, ".svg": true, ".pdf": true, ".txt": true,
}

// ResolveUnderRoot maps a URL path (e.g. "/vector06c/foo.rom") to an absolute path under rootDir.
func ResolveUnderRoot(rootDir, urlPath string) (string, error) {
	rootDir = filepath.Clean(rootDir)
	rel := strings.TrimPrefix(urlPath, "/")
	if rel == "" {
		return "", ErrOutsideRoot
	}
	if strings.Contains(rel, "..") {
		return "", ErrOutsideRoot
	}
	for _, seg := range strings.Split(rel, "/") {
		if seg == "" || strings.HasPrefix(seg, ".") {
			return "", ErrOutsideRoot
		}
	}
	full := filepath.Clean(filepath.Join(rootDir, filepath.FromSlash(rel)))
	check, err := filepath.Rel(rootDir, full)
	if err != nil || strings.HasPrefix(check, "..") {
		return "", ErrOutsideRoot
	}
	return full, nil
}

// IsStorageAssetPath reports whether urlPath looks like a storage asset by extension.
func IsStorageAssetPath(urlPath string) bool {
	return StorageAssetExtensions[strings.ToLower(filepath.Ext(urlPath))]
}

// AllowedStorageFile reports whether a resolved storage file may be served.
func AllowedStorageFile(urlPath string, info os.FileInfo) bool {
	if info.IsDir() {
		return false
	}
	return IsStorageAssetPath(urlPath)
}

// StatUnderRoot resolves urlPath under rootDir and ensures the result stays within root after symlink resolution.
func StatUnderRoot(rootDir, urlPath string) (os.FileInfo, string, error) {
	full, err := ResolveUnderRoot(rootDir, urlPath)
	if err != nil {
		return nil, "", err
	}
	info, err := os.Lstat(full)
	if err != nil {
		return nil, "", err
	}
	resolved := full
	if info.Mode()&os.ModeSymlink != 0 {
		resolved, err = filepath.EvalSymlinks(full)
		if err != nil {
			return nil, "", err
		}
	}
	resolved = filepath.Clean(resolved)
	rel, err := filepath.Rel(filepath.Clean(rootDir), resolved)
	if err != nil || strings.HasPrefix(rel, "..") {
		return nil, "", ErrOutsideRoot
	}
	info, err = os.Stat(resolved)
	if err != nil {
		return nil, "", err
	}
	return info, resolved, nil
}

// SetFrontendCacheHeaders applies cache policy for SPA shell vs hashed build assets.
func SetFrontendCacheHeaders(w http.ResponseWriter, urlPath string) {
	switch {
	case urlPath == "/" || urlPath == "/index.html" || strings.HasSuffix(urlPath, "/index.html"):
		w.Header().Set("Cache-Control", "no-cache, must-revalidate")
	case strings.HasPrefix(urlPath, "/assets/"):
		// Hashed filenames change each build; avoid immutable so stale index bundles are dropped.
		w.Header().Set("Cache-Control", "public, max-age=0, must-revalidate")
	}
}

// ServeFile writes a file that was resolved with StatUnderRoot.
func ServeFile(w http.ResponseWriter, r *http.Request, resolvedPath string, info os.FileInfo) {
	if info.IsDir() {
		http.NotFound(w, r)
		return
	}
	SetFrontendCacheHeaders(w, r.URL.Path)
	http.ServeFile(w, r, resolvedPath)
}
