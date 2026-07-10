package parser

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRenderDescription_markdownLinks(t *testing.T) {
	desc := `Адаптация игры. См. также [дисковая версия](exolond.com) и [автора](../../authors/alice).`
	got := RenderDescription(desc, "vector06c/exolon")
	require.NotEmpty(t, got)
	assert.Contains(t, got, `href="/vector06c/exolon/exolond.com"`)
	assert.Contains(t, got, `href="/authors/alice"`)
	assert.NotContains(t, got, "[дисковая версия]")
}
