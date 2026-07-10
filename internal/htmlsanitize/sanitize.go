package htmlsanitize

import (
	"github.com/microcosm-cc/bluemonday"
)

// policy allows typical catalog content (links, images, headings) without scripts.
var policy = bluemonday.UGCPolicy()

func init() {
	policy.RequireParseableURLs(true)
	policy.AddTargetBlankToFullyQualifiedLinks(true)
}

// Sanitize strips dangerous HTML from user-generated markdown output.
func Sanitize(html string) string {
	return policy.Sanitize(html)
}
