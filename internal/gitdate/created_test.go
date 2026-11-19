package gitdate

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestBatchFileFirstCommitTimes_followsMovesAndIgnoresEdits(t *testing.T) {
	dir := t.TempDir()
	runGit(t, dir, "init")
	runGit(t, dir, "config", "user.email", "test@example.com")
	runGit(t, dir, "config", "user.name", "Test")
	write := func(path, content string) {
		t.Helper()
		full := filepath.Join(dir, path)
		require.NoError(t, os.MkdirAll(filepath.Dir(full), 0o755))
		require.NoError(t, os.WriteFile(full, []byte(content), 0o644))
	}
	commit := func(date string) {
		t.Helper()
		runGit(t, dir, "add", ".")
		runGit(t, dir, "commit", "--date="+date, "-m", "Update catalog")
	}

	const original = "vector06c/xonix/README.md"
	const intermediate = "vector06c/xonix/1/README.md"
	const moved = "vector06c/игры\t archive/xonix/1.1/README.md"
	const simon = "vector06c/simon_cat/README.md"
	const firstDate = "2026-02-07T11:54:45+03:00"
	const simonDate = "2026-11-18T20:02:58+03:00"

	write(original, "Original Xonix card\n")
	commit(firstDate)
	require.NoError(t, os.MkdirAll(filepath.Dir(filepath.Join(dir, intermediate)), 0o755))
	runGit(t, dir, "mv", original, intermediate)
	commit("2026-07-29T17:16:44+03:00")
	require.NoError(t, os.MkdirAll(filepath.Dir(filepath.Join(dir, moved)), 0o755))
	runGit(t, dir, "mv", intermediate, moved)
	commit("2026-09-01T12:00:00+03:00")
	write(simon, "Simon Cat demo\n")
	commit(simonDate)
	write(moved, "Edited Xonix description\n")
	commit("2026-11-19T12:00:00+03:00")
	for _, paths := range [][]string{{moved, simon}, {moved, simon, "missing/README.md"}} {
		dates, err := BatchFileFirstCommitTimes(context.Background(), dir, paths)
		require.NoError(t, err)
		assert.Equal(t, mustDate(t, firstDate), dates[moved])
		assert.Equal(t, mustDate(t, simonDate), dates[simon])
		assert.True(t, dates[moved].Before(dates[simon]), "editing or moving Xonix must not make it newer than Simon Cat")
		if len(paths) == 3 {
			assert.NotContains(t, dates, "missing/README.md")
		}
	}
}

func mustDate(t *testing.T, value string) time.Time {
	t.Helper()
	date, err := time.Parse(time.RFC3339, value)
	require.NoError(t, err)
	return date
}
