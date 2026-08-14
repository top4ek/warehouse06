package storage

import (
	"archive/zip"
	"context"
	"io"
	"os"
	"path/filepath"
	"sort"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func writeFile(t *testing.T, root, rel, content string) {
	t.Helper()
	path := filepath.Join(root, rel)
	require.NoError(t, os.MkdirAll(filepath.Dir(path), 0o755))
	require.NoError(t, os.WriteFile(path, []byte(content), 0o644))
}

func archiveMembers(t *testing.T, zipPath string) map[string]string {
	t.Helper()
	zr, err := zip.OpenReader(zipPath)
	require.NoError(t, err)
	defer func() { _ = zr.Close() }()

	members := make(map[string]string, len(zr.File))
	for _, f := range zr.File {
		rc, err := f.Open()
		require.NoError(t, err)
		data, err := io.ReadAll(rc)
		require.NoError(t, rc.Close())
		require.NoError(t, err)
		members[f.Name] = string(data)
	}
	return members
}

func TestArchiveDir_skipsDotEntriesAndNonRegularFiles(t *testing.T) {
	root := t.TempDir()
	writeFile(t, root, "README.md", "root readme")
	writeFile(t, root, "vector06c/demo/README.md", "demo readme")
	writeFile(t, root, "vector06c/demo/demo.rom", "rom bytes")
	writeFile(t, root, ".git/HEAD", "ref: refs/heads/master")
	writeFile(t, root, ".keep", "")
	writeFile(t, root, "vector06c/.hidden", "hidden")
	require.NoError(t, os.Symlink(filepath.Join(root, "README.md"), filepath.Join(root, "link.md")))

	destPath := filepath.Join(t.TempDir(), "storage.zip")
	size, err := ArchiveDir(context.Background(), root, destPath)
	require.NoError(t, err)
	assert.Positive(t, size)

	members := archiveMembers(t, destPath)
	names := make([]string, 0, len(members))
	for name := range members {
		names = append(names, name)
	}
	sort.Strings(names)

	assert.Equal(t, []string{
		"README.md",
		"vector06c/demo/README.md",
		"vector06c/demo/demo.rom",
	}, names)
	assert.Equal(t, "demo readme", members["vector06c/demo/README.md"])

	info, err := os.Stat(destPath)
	require.NoError(t, err)
	assert.Equal(t, size, info.Size())
}

func TestArchiveDir_emptyDirProducesEmptyArchive(t *testing.T) {
	destPath := filepath.Join(t.TempDir(), "storage.zip")
	_, err := ArchiveDir(context.Background(), t.TempDir(), destPath)
	require.NoError(t, err)

	assert.Empty(t, archiveMembers(t, destPath))
}

func TestArchiveDir_canceledContext(t *testing.T) {
	root := t.TempDir()
	writeFile(t, root, "README.md", "root readme")

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	destPath := filepath.Join(t.TempDir(), "storage.zip")
	_, err := ArchiveDir(ctx, root, destPath)
	require.ErrorIs(t, err, context.Canceled)
}

func TestArchiveDir_missingDir(t *testing.T) {
	destPath := filepath.Join(t.TempDir(), "storage.zip")
	_, err := ArchiveDir(context.Background(), filepath.Join(t.TempDir(), "absent"), destPath)
	require.Error(t, err)
}
