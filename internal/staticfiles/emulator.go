package staticfiles

import (
	"os"
	"path/filepath"
)

// EmulatorDir returns the vector06js static tree (dist copy, else public/).
func EmulatorDir(workDir string) string {
	dist := filepath.Join(workDir, "frontend", "dist", "emulator")
	if st, err := os.Stat(dist); err == nil && st.IsDir() {
		return dist
	}
	return filepath.Join(workDir, "frontend", "public", "emulator")
}
