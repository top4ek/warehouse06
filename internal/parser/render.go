package parser

import (
	"bytes"
	"strings"

	"github.com/yuin/goldmark"
	"github.com/yuin/goldmark/extension"
	"github.com/yuin/goldmark/parser"
	"github.com/yuin/goldmark/renderer/html"

	"warehouse06/internal/htmlsanitize"
)

var descriptionMarkdown = goldmark.New(
	goldmark.WithExtensions(extension.GFM),
	goldmark.WithParserOptions(
		parser.WithAutoHeadingID(),
	),
	goldmark.WithRendererOptions(
		html.WithHardWraps(),
		html.WithXHTML(),
	),
)

// RenderDescription converts list-card description markdown to sanitized HTML.
func RenderDescription(description, entryPath string) string {
	description = strings.TrimSpace(description)
	if description == "" {
		return ""
	}

	var buf bytes.Buffer
	if err := descriptionMarkdown.Convert([]byte(description), &buf); err != nil {
		return ""
	}
	return htmlsanitize.Sanitize(rewriteResourcePaths(buf.String(), entryPath))
}
