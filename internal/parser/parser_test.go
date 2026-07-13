package parser

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

func TestParser_extractFrontmatter(t *testing.T) {
	p := NewParser(t.TempDir(), zap.NewNop())

	fm, body, err := p.extractFrontmatter([]byte("---\nname: Test Game\ntags:\n  - demo\n---\n\nHello"))
	require.NoError(t, err)
	assert.Equal(t, "Test Game", fm.Name)
	assert.Equal(t, []string{"demo"}, fm.Tags)
	assert.Contains(t, string(body), "Hello")
}

func TestParser_extractFrontmatter_noFrontmatter(t *testing.T) {
	p := NewParser(t.TempDir(), zap.NewNop())

	fm, body, err := p.extractFrontmatter([]byte("# Title\n\nBody"))
	require.NoError(t, err)
	assert.Empty(t, fm.Name)
	assert.Equal(t, "# Title\n\nBody", string(body))
}

func TestParser_extractDescription(t *testing.T) {
	p := NewParser(t.TempDir(), zap.NewNop())

	desc := p.extractDescription([]byte("# Title\n\nFirst line.\nSecond line.\n\nMore."))
	assert.Equal(t, "First line. Second line.", desc)
}

func TestParser_ParseFile(t *testing.T) {
	dir := t.TempDir()
	readme := filepath.Join(dir, "README.md")
	content := "---\nname: Plyuk\ntags:\n  - game\n---\n\nVector game.\n"
	require.NoError(t, os.WriteFile(readme, []byte(content), 0o644))

	p := NewParser(dir, zap.NewNop())
	entry, fm, err := p.ParseFile(readme)
	require.NoError(t, err)
	assert.Equal(t, "Plyuk", entry.Name)
	assert.Equal(t, "Plyuk", fm.Name)
	assert.Equal(t, []string{"game"}, fm.Tags)
	assert.Contains(t, entry.ContentHTML, "Vector game")
}

func TestParser_ParseFile_controls(t *testing.T) {
	dir := t.TempDir()
	readme := filepath.Join(dir, "README.md")
	content := "---\nname: Game\ncontrols:\n  rows:\n    - [~, up, f12]\n    - [left, down, right]\n---\n\nText.\n"
	require.NoError(t, os.WriteFile(readme, []byte(content), 0o644))

	p := NewParser(dir, zap.NewNop())
	entry, _, err := p.ParseFile(readme)
	require.NoError(t, err)
	assert.JSONEq(t, `{"rows":[[null,"up","f12"],["left","down","right"]]}`, string(entry.Controls))
}

func TestParser_ParseFile_noControls(t *testing.T) {
	dir := t.TempDir()
	readme := filepath.Join(dir, "README.md")
	require.NoError(t, os.WriteFile(readme, []byte("---\nname: Game\n---\n\nText.\n"), 0o644))

	p := NewParser(dir, zap.NewNop())
	entry, _, err := p.ParseFile(readme)
	require.NoError(t, err)
	assert.Empty(t, entry.Controls)
}

func TestIsRelativeResource(t *testing.T) {
	assert.True(t, isRelativeResource("screenshot.png"))
	assert.False(t, isRelativeResource("/absolute.png"))
	assert.False(t, isRelativeResource("https://example.com/x"))
	assert.False(t, isRelativeResource("#anchor"))
}

func writeREADME(t *testing.T, root, relDir, content string) {
	t.Helper()
	dir := filepath.Join(root, relDir)
	require.NoError(t, os.MkdirAll(dir, 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "README.md"), []byte(content), 0o644))
}

func TestParser_ScanDirectory(t *testing.T) {
	root := t.TempDir()
	writeREADME(t, root, "vector06c/demo",
		"---\nname: Demo\ntags:\n  - game\nauthors:\n  - alice\nscreenshots:\n  - shot.png\n---\n\nGame text.\n")
	writeREADME(t, root, "authors/alice",
		"---\nname: Alice\naddress: City\n---\n\nAuthor page.\n")
	// .git should be skipped
	require.NoError(t, os.MkdirAll(filepath.Join(root, ".git", "objects"), 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(root, ".git", "HEAD"), []byte("ref"), 0o644))
	require.NoError(t, os.WriteFile(filepath.Join(root, "vector06c/demo/shot.png"), []byte("png"), 0o644))
	require.NoError(t, os.WriteFile(filepath.Join(root, "vector06c/demo/extra.bin"), []byte("bin"), 0o644))

	p := NewParser(root, zap.NewNop())
	entries, authors, err := p.ScanDirectory()
	require.NoError(t, err)
	require.Len(t, entries, 1)
	require.Len(t, authors, 1)

	assert.Equal(t, "vector06c/demo", entries[0].Path)
	assert.Equal(t, "Demo", entries[0].Name)
	require.Len(t, entries[0].Tags, 1)
	assert.Equal(t, "game", entries[0].Tags[0].Name)
	assert.Equal(t, "alice", authors[0].DirectoryName)
	assert.NotEmpty(t, entries[0].Files)
}

func TestParser_rewriteResourcePaths(t *testing.T) {
	html := `<img src="./shot.png"><a href="https://example.com/x">link</a>`
	got := rewriteResourcePaths(html, "vector06c/demo")
	assert.Contains(t, got, `src="/vector06c/demo/shot.png"`)
	assert.Contains(t, got, `href="https://example.com/x"`)
}
