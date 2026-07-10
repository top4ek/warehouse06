package pathutil

import (
	"fmt"
	"path/filepath"
	"strings"
)

// UnderRoot reports whether path resolves to a location inside root (after symlink resolution).
func UnderRoot(root, path string) error {
	root = filepath.Clean(root)
	abs, err := filepath.Abs(path)
	if err != nil {
		return fmt.Errorf("resolve path: %w", err)
	}
	resolved := abs
	if link, err := filepath.EvalSymlinks(abs); err == nil {
		resolved = filepath.Clean(link)
	}
	rel, err := filepath.Rel(root, resolved)
	if err != nil || strings.HasPrefix(rel, "..") {
		return fmt.Errorf("path %q is outside storage root", path)
	}
	return nil
}
