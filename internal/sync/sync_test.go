package sync

import (
	"archive/zip"
	"context"
	"database/sql"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"

	"warehouse06/internal/parser"
	"warehouse06/internal/repository"
)

func TestSyncer_tryStart_secondCallReturnsFalse(t *testing.T) {
	s := &Syncer{}
	assert.True(t, s.tryStart())
	assert.False(t, s.tryStart())
	s.finish()
	assert.True(t, s.tryStart())
	s.finish()
}

func writeStorageREADME(t *testing.T, root, relDir, content string) {
	t.Helper()
	dir := filepath.Join(root, relDir)
	require.NoError(t, os.MkdirAll(dir, 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "README.md"), []byte(content), 0o644))
}

func newTestSyncer(t *testing.T, storageDir string) (*Syncer, *repository.Holder, *Status, string, string) {
	t.Helper()
	repo, err := repository.NewSQLiteRepository(":memory:", zap.NewNop())
	require.NoError(t, err)
	t.Cleanup(func() { _ = repo.Close() })

	holder := repository.NewHolder(repo, ":memory:")
	status := NewStatus()
	p := parser.NewParser(storageDir, zap.NewNop())
	exportDir := t.TempDir()
	rebusPath := filepath.Join(exportDir, "rebus-export.zip")
	sqlitePath := filepath.Join(exportDir, "sqlite-export.zip")
	syncer := NewSyncer(storageDir, "", ":memory:", 0, holder, p, status, rebusPath, sqlitePath, zap.NewNop())
	return syncer, holder, status, rebusPath, sqlitePath
}

// extractSQLiteExport unzips the SQLite export and returns the path of the
// database file it contains.
func extractSQLiteExport(t *testing.T, zipPath string) string {
	t.Helper()
	zr, err := zip.OpenReader(zipPath)
	require.NoError(t, err)
	t.Cleanup(func() { _ = zr.Close() })

	require.Len(t, zr.File, 1)
	assert.Equal(t, "warehouse06.sqlite", zr.File[0].Name)

	rc, err := zr.File[0].Open()
	require.NoError(t, err)
	defer func() { _ = rc.Close() }()
	data, err := io.ReadAll(rc)
	require.NoError(t, err)

	dbPath := filepath.Join(t.TempDir(), "export.sqlite")
	require.NoError(t, os.WriteFile(dbPath, data, 0o644))
	return dbPath
}

func TestSyncer_Sync_withoutGit(t *testing.T) {
	storageDir := t.TempDir()
	writeStorageREADME(t, storageDir, "vector06c/demo",
		"---\nname: Demo\ntags:\n  - game\nauthors:\n  - alice\n---\n\nDemo game body.\n")
	writeStorageREADME(t, storageDir, "authors/alice",
		"---\nname: Alice\naddress: Somewhere\n---\n\nAuthor bio.\n")

	syncer, holder, status, exportPath, sqliteExportPath := newTestSyncer(t, storageDir)
	require.NoError(t, syncer.Sync(context.Background()))

	assert.False(t, status.Syncing())
	assert.False(t, status.LastSyncedAt().IsZero())

	entry, err := holder.Get().GetEntryByPath(context.Background(), "vector06c/demo")
	require.NoError(t, err)
	assert.Equal(t, "Demo", entry.Name)
	require.Len(t, entry.Tags, 1)
	assert.Equal(t, "game", entry.Tags[0].Name)

	author, err := holder.Get().GetAuthorByDir(context.Background(), "alice")
	require.NoError(t, err)
	assert.Equal(t, "Alice", author.Name)

	exportData, err := os.ReadFile(exportPath)
	require.NoError(t, err, "rebus export file should exist after a successful rebuild")
	assert.NotEmpty(t, exportData)

	dbPath := extractSQLiteExport(t, sqliteExportPath)
	db, err := sql.Open("sqlite3", dbPath)
	require.NoError(t, err)
	t.Cleanup(func() { _ = db.Close() })

	var name string
	require.NoError(t, db.QueryRow(`SELECT name FROM entries WHERE path = ?`, "vector06c/demo").Scan(&name))
	assert.Equal(t, "Demo", name)

	var tag string
	require.NoError(t, db.QueryRow(`
		SELECT t.name FROM tags t
		JOIN entry_tags et ON et.tag_id = t.id
		JOIN entries e ON e.id = et.entry_id
		WHERE e.path = ?`, "vector06c/demo").Scan(&tag))
	assert.Equal(t, "game", tag)

	// The live catalog keeps its FTS index; only the exported copy drops it.
	found, err := holder.Get().SearchEntries(context.Background(), repository.FormatFTSQuery("Demo"), 10, 0)
	require.NoError(t, err)
	assert.NotEmpty(t, found)
}

func TestSyncer_Sync_whileRunningSkips(t *testing.T) {
	storageDir := t.TempDir()
	writeStorageREADME(t, storageDir, "vector06c/demo",
		"---\nname: Demo\n---\n\nBody.\n")

	syncer, _, _, _, _ := newTestSyncer(t, storageDir)
	require.True(t, syncer.tryStart())
	defer syncer.finish()

	err := syncer.Sync(context.Background())
	require.NoError(t, err)
}

func TestSyncer_Sync_unchangedGitStillUpdatesLastSyncedAt(t *testing.T) {
	remoteDir := t.TempDir()
	initGitRepo(t, remoteDir)
	writeStorageREADME(t, remoteDir, "vector06c/demo", "---\nname: Demo\n---\n\nBody.\n")
	gitCommitAll(t, remoteDir, "initial")
	remoteURL := fmt.Sprintf("file://%s", filepath.ToSlash(remoteDir))

	storageDir := t.TempDir()
	repo, err := repository.NewSQLiteRepository(":memory:", zap.NewNop())
	require.NoError(t, err)
	t.Cleanup(func() { _ = repo.Close() })
	holder := repository.NewHolder(repo, ":memory:")
	status := NewStatus()
	p := parser.NewParser(storageDir, zap.NewNop())
	exportDir := t.TempDir()
	exportPath := filepath.Join(exportDir, "rebus-export.zip")
	sqliteExportPath := filepath.Join(exportDir, "sqlite-export.zip")
	syncer := NewSyncer(storageDir, remoteURL, ":memory:", 0, holder, p, status, exportPath, sqliteExportPath, zap.NewNop())

	require.NoError(t, syncer.Sync(context.Background()))
	firstSyncedAt := status.LastSyncedAt()
	require.False(t, firstSyncedAt.IsZero())

	firstExportData, err := os.ReadFile(exportPath)
	require.NoError(t, err)
	firstExportInfo, err := os.Stat(exportPath)
	require.NoError(t, err)

	firstSQLiteData, err := os.ReadFile(sqliteExportPath)
	require.NoError(t, err)
	firstSQLiteInfo, err := os.Stat(sqliteExportPath)
	require.NoError(t, err)

	time.Sleep(time.Millisecond)

	// No changes pushed to remoteDir since the first sync: the second sync
	// must skip the database rebuild but still advance LastSyncedAt.
	require.NoError(t, syncer.Sync(context.Background()))
	secondSyncedAt := status.LastSyncedAt()
	assert.True(t, secondSyncedAt.After(firstSyncedAt),
		"expected LastSyncedAt to advance on an unchanged sync, got first=%v second=%v", firstSyncedAt, secondSyncedAt)

	// The exports are derived artifacts of the DB rebuild; skipping the
	// rebuild must leave them untouched rather than needlessly regenerating them.
	secondExportData, err := os.ReadFile(exportPath)
	require.NoError(t, err)
	assert.Equal(t, firstExportData, secondExportData)
	secondExportInfo, err := os.Stat(exportPath)
	require.NoError(t, err)
	assert.Equal(t, firstExportInfo.ModTime(), secondExportInfo.ModTime())

	secondSQLiteData, err := os.ReadFile(sqliteExportPath)
	require.NoError(t, err)
	assert.Equal(t, firstSQLiteData, secondSQLiteData)
	secondSQLiteInfo, err := os.Stat(sqliteExportPath)
	require.NoError(t, err)
	assert.Equal(t, firstSQLiteInfo.ModTime(), secondSQLiteInfo.ModTime())
}

func initGitRepo(t *testing.T, dir string) {
	t.Helper()
	runGit(t, dir, "init")
	runGit(t, dir, "config", "user.email", "test@example.com")
	runGit(t, dir, "config", "user.name", "Test")
}

func gitCommitAll(t *testing.T, dir, message string) {
	t.Helper()
	runGit(t, dir, "add", "-A")
	runGit(t, dir, "commit", "-m", message)
}

func runGit(t *testing.T, dir string, args ...string) {
	t.Helper()
	cmd := exec.Command("git", args...)
	cmd.Dir = dir
	cmd.Env = append(os.Environ(), "GIT_CONFIG_GLOBAL=/dev/null", "GIT_CONFIG_SYSTEM=/dev/null")
	out, err := cmd.CombinedOutput()
	require.NoError(t, err, strings.TrimSpace(string(out)))
}

func TestSyncer_Run_intervalZero(t *testing.T) {
	storageDir := t.TempDir()
	writeStorageREADME(t, storageDir, "vector06c/demo",
		"---\nname: Demo\n---\n\nBody.\n")

	syncer, holder, status, _, _ := newTestSyncer(t, storageDir)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	done := make(chan struct{})
	go func() {
		syncer.Run(ctx)
		close(done)
	}()

	select {
	case <-done:
	case <-time.After(5 * time.Second):
		t.Fatal("Run did not return")
	}

	assert.False(t, status.LastSyncedAt().IsZero())
	_, err := holder.Get().GetEntryByPath(context.Background(), "vector06c/demo")
	require.NoError(t, err)
}
