package sync

import (
	"context"
	"fmt"
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

func newTestSyncer(t *testing.T, storageDir string) (*Syncer, *repository.Holder, *Status) {
	t.Helper()
	repo, err := repository.NewSQLiteRepository(":memory:", zap.NewNop())
	require.NoError(t, err)
	t.Cleanup(func() { _ = repo.Close() })

	holder := repository.NewHolder(repo, ":memory:")
	status := NewStatus()
	p := parser.NewParser(storageDir, zap.NewNop())
	syncer := NewSyncer(storageDir, "", ":memory:", 0, holder, p, status, zap.NewNop())
	return syncer, holder, status
}

func TestSyncer_Sync_withoutGit(t *testing.T) {
	storageDir := t.TempDir()
	writeStorageREADME(t, storageDir, "vector06c/demo",
		"---\nname: Demo\ntags:\n  - game\nauthors:\n  - alice\n---\n\nDemo game body.\n")
	writeStorageREADME(t, storageDir, "authors/alice",
		"---\nname: Alice\naddress: Somewhere\n---\n\nAuthor bio.\n")

	syncer, holder, status := newTestSyncer(t, storageDir)
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
}

func TestSyncer_Sync_whileRunningSkips(t *testing.T) {
	storageDir := t.TempDir()
	writeStorageREADME(t, storageDir, "vector06c/demo",
		"---\nname: Demo\n---\n\nBody.\n")

	syncer, _, _ := newTestSyncer(t, storageDir)
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
	syncer := NewSyncer(storageDir, remoteURL, ":memory:", 0, holder, p, status, zap.NewNop())

	require.NoError(t, syncer.Sync(context.Background()))
	firstSyncedAt := status.LastSyncedAt()
	require.False(t, firstSyncedAt.IsZero())

	time.Sleep(time.Millisecond)

	// No changes pushed to remoteDir since the first sync: the second sync
	// must skip the database rebuild but still advance LastSyncedAt.
	require.NoError(t, syncer.Sync(context.Background()))
	secondSyncedAt := status.LastSyncedAt()
	assert.True(t, secondSyncedAt.After(firstSyncedAt),
		"expected LastSyncedAt to advance on an unchanged sync, got first=%v second=%v", firstSyncedAt, secondSyncedAt)
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

	syncer, holder, status := newTestSyncer(t, storageDir)
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
