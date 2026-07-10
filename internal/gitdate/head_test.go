package gitdate

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestHeadCommit(t *testing.T) {
	dir := t.TempDir()
	runGit(t, dir, "init")
	runGit(t, dir, "config", "user.email", "test@example.com")
	runGit(t, dir, "config", "user.name", "Test")
	require.NoError(t, os.WriteFile(filepath.Join(dir, "README.md"), []byte("hello"), 0o644))
	runGit(t, dir, "add", "README.md")
	runGit(t, dir, "commit", "-m", "Initial import")

	commit, ok, err := HeadCommit(context.Background(), dir)
	require.NoError(t, err)
	require.True(t, ok)
	assert.Len(t, commit.Hash, 40)
	assert.False(t, commit.CommittedAt.IsZero())
	assert.Equal(t, "Initial import", commit.Subject)
}

func TestHeadCommit_notARepo(t *testing.T) {
	dir := t.TempDir()
	_, ok, err := HeadCommit(context.Background(), dir)
	require.NoError(t, err)
	assert.False(t, ok)
}

func runGit(t *testing.T, dir string, args ...string) {
	t.Helper()
	cmd := exec.Command("git", args...)
	cmd.Dir = dir
	cmd.Env = append(os.Environ(), "GIT_CONFIG_GLOBAL=/dev/null", "GIT_CONFIG_SYSTEM=/dev/null")
	out, err := cmd.CombinedOutput()
	require.NoError(t, err, string(out))
}
