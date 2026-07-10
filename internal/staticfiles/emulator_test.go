package staticfiles

import (
	"os"
	"path/filepath"
	"testing"
)

func TestEmulatorDirPrefersDist(t *testing.T) {
	root := t.TempDir()
	distEmu := filepath.Join(root, "frontend", "dist", "emulator")
	if err := os.MkdirAll(distEmu, 0o755); err != nil {
		t.Fatal(err)
	}
	if got := EmulatorDir(root); got != distEmu {
		t.Fatalf("got %q want %q", got, distEmu)
	}
}
