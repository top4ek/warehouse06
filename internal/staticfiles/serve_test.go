package staticfiles

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestResolveUnderRoot_blocksTraversal(t *testing.T) {
	root := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(root, "secret.txt"), []byte("x"), 0o644))

	_, err := ResolveUnderRoot(root, "/../../../etc/passwd")
	assert.ErrorIs(t, err, ErrOutsideRoot)

	_, err = ResolveUnderRoot(root, "/../secret.txt")
	assert.ErrorIs(t, err, ErrOutsideRoot)

	_, err = ResolveUnderRoot(root, "/.git/config")
	assert.ErrorIs(t, err, ErrOutsideRoot)
}

func TestResolveUnderRoot_validPath(t *testing.T) {
	root := t.TempDir()
	sub := filepath.Join(root, "vector06c")
	require.NoError(t, os.Mkdir(sub, 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(sub, "game.rom"), []byte("rom"), 0o644))

	got, err := ResolveUnderRoot(root, "/vector06c/game.rom")
	require.NoError(t, err)
	assert.Equal(t, filepath.Join(sub, "game.rom"), got)
}

func TestStatUnderRoot_rejectsSymlinkEscape(t *testing.T) {
	root := t.TempDir()
	outside := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(outside, "leak.txt"), []byte("secret"), 0o644))
	linkDir := filepath.Join(root, "vector06c")
	require.NoError(t, os.Mkdir(linkDir, 0o755))
	require.NoError(t, os.Symlink(filepath.Join(outside, "leak.txt"), filepath.Join(linkDir, "evil.rom")))

	_, _, err := StatUnderRoot(root, "/vector06c/evil.rom")
	assert.ErrorIs(t, err, ErrOutsideRoot)
}

func TestAllowedStorageFile(t *testing.T) {
	assert.True(t, AllowedStorageFile("/a/b.rom", fileInfo(t, false)))
	assert.False(t, AllowedStorageFile("/a/README.md", fileInfo(t, false)))
}

func fileInfo(t *testing.T, dir bool) os.FileInfo {
	t.Helper()
	f, err := os.CreateTemp(t.TempDir(), "f")
	require.NoError(t, err)
	require.NoError(t, f.Close())
	info, err := os.Stat(f.Name())
	require.NoError(t, err)
	if dir {
		require.NoError(t, os.Remove(f.Name()))
		d, err := os.MkdirTemp(t.TempDir(), "d")
		require.NoError(t, err)
		info, err = os.Stat(d)
		require.NoError(t, err)
	}
	return info
}
