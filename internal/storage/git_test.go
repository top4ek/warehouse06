package storage

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

func TestSyncGit_emptyURL(t *testing.T) {
	dir := t.TempDir()
	result, err := SyncGit(context.Background(), dir, "", zap.NewNop())
	require.NoError(t, err)
	assert.False(t, result.Changed)
}

func TestGitPull_notARepo(t *testing.T) {
	dir := t.TempDir()
	err := GitPull(context.Background(), dir, zap.NewNop())
	require.Error(t, err)
}

func TestSyncGit_missingDir_clones(t *testing.T) {
	parent := t.TempDir()
	workDir := filepath.Join(parent, "storage")

	remoteDir := t.TempDir()
	initGitRepo(t, remoteDir)
	require.NoError(t, os.WriteFile(filepath.Join(remoteDir, "README.md"), []byte("x"), 0o644))
	gitCommitAll(t, remoteDir, "initial")

	remoteURL := fmt.Sprintf("file://%s", filepath.ToSlash(remoteDir))

	result, err := SyncGit(context.Background(), workDir, remoteURL, zap.NewNop())
	require.NoError(t, err)
	assert.True(t, result.Changed)
	assert.True(t, result.HasHead)
	_, err = os.Stat(filepath.Join(workDir, ".git"))
	require.NoError(t, err)
}

func TestSyncGit_nonGitDirWithKeep_wipesAndClones(t *testing.T) {
	workDir := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(workDir, ".keep"), nil, 0o644))

	remoteDir := t.TempDir()
	initGitRepo(t, remoteDir)
	require.NoError(t, os.WriteFile(filepath.Join(remoteDir, "README.md"), []byte("x"), 0o644))
	gitCommitAll(t, remoteDir, "initial")

	remoteURL := fmt.Sprintf("file://%s", filepath.ToSlash(remoteDir))

	result, err := SyncGit(context.Background(), workDir, remoteURL, zap.NewNop())
	require.NoError(t, err)
	assert.True(t, result.Changed)
	assert.True(t, result.HasHead)

	_, err = os.Stat(filepath.Join(workDir, ".keep"))
	assert.True(t, os.IsNotExist(err))
	_, err = os.Stat(filepath.Join(workDir, "README.md"))
	require.NoError(t, err)
}

func TestSyncGit_existingRepoWithoutRemote(t *testing.T) {
	dir := t.TempDir()
	initGitRepo(t, dir)
	require.NoError(t, os.WriteFile(filepath.Join(dir, "README.md"), []byte("x"), 0o644))
	gitCommitAll(t, dir, "initial")

	_, err := SyncGit(context.Background(), dir, "https://example.com/repo.git", zap.NewNop())
	require.Error(t, err)
}

func TestSyncGit_pullNoChanges(t *testing.T) {
	remoteDir := t.TempDir()
	initGitRepo(t, remoteDir)
	require.NoError(t, os.WriteFile(filepath.Join(remoteDir, "README.md"), []byte("x"), 0o644))
	gitCommitAll(t, remoteDir, "initial")

	remoteURL := fmt.Sprintf("file://%s", filepath.ToSlash(remoteDir))
	workDir := t.TempDir()

	first, err := SyncGit(context.Background(), workDir, remoteURL, zap.NewNop())
	require.NoError(t, err)
	require.True(t, first.Changed)
	require.True(t, first.HasHead)

	second, err := SyncGit(context.Background(), workDir, remoteURL, zap.NewNop())
	require.NoError(t, err)
	assert.False(t, second.Changed)
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
