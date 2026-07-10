package pathutil

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestUnderRoot_rejectsOutside(t *testing.T) {
	root := t.TempDir()
	outside := t.TempDir()
	outFile := filepath.Join(outside, "secret.txt")
	require.NoError(t, os.WriteFile(outFile, []byte("x"), 0o644))

	err := UnderRoot(root, outFile)
	assert.Error(t, err)
}

func TestUnderRoot_acceptsInside(t *testing.T) {
	root := t.TempDir()
	inside := filepath.Join(root, "vector06c", "README.md")
	require.NoError(t, os.MkdirAll(filepath.Dir(inside), 0o755))
	require.NoError(t, os.WriteFile(inside, []byte("# hi"), 0o644))

	assert.NoError(t, UnderRoot(root, inside))
}
