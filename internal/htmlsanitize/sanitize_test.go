package htmlsanitize

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestSanitize_removesScript(t *testing.T) {
	in := `<p>Hello</p><script>alert(1)</script>`
	out := Sanitize(in)
	assert.NotContains(t, out, "<script")
	assert.Contains(t, out, "Hello")
}

func TestSanitize_keepsSafeTags(t *testing.T) {
	in := `<h2>Title</h2><p>Text with <a href="https://example.com">link</a> and <img src="/x.png" alt="shot">`
	out := Sanitize(in)
	assert.Contains(t, out, "<h2>")
	assert.Contains(t, out, "<a ")
	assert.Contains(t, out, "<img ")
	assert.False(t, strings.Contains(out, "<script"))
}
