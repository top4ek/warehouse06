package staticfiles

import (
	"os"
	"path/filepath"
	"testing"
)

func TestResolveLegacyAsset(t *testing.T) {
	dir := t.TempDir()
	write := func(name, content string) {
		t.Helper()
		if err := os.WriteFile(filepath.Join(dir, name), []byte(content), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	write("LoadingCenter-Tu8LI2JA.js", "lc")
	write("entry.js", "entry")
	write("index-CsZD4Ok1.js", "idx")

	got, ok := ResolveLegacyAsset(dir, "LoadingCenter-CUlW9hhl.js")
	if !ok || filepath.Base(got) != "LoadingCenter-Tu8LI2JA.js" {
		t.Fatalf("LoadingCenter legacy: got %q ok=%v", got, ok)
	}

	got, ok = ResolveLegacyAsset(dir, "Entry-DWbFBUTC.js")
	if !ok || filepath.Base(got) != "entry.js" {
		t.Fatalf("Entry legacy: got %q ok=%v", got, ok)
	}
}
