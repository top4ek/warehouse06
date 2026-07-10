package repository

import (
	"fmt"
	"strings"
	"unicode"
)

// FormatFTSQuery turns user input into an FTS5 prefix query (partial word match).
func FormatFTSQuery(raw string) string {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return ""
	}

	terms := strings.Fields(raw)
	parts := make([]string, 0, len(terms))
	for _, term := range terms {
		term = strings.Trim(term, `"`)
		if term == "" || !isValidFTSTerm(term) {
			continue
		}
		var escaped strings.Builder
		for _, r := range term {
			if r == '"' {
				escaped.WriteString(`""`)
			} else {
				escaped.WriteRune(r)
			}
		}
		parts = append(parts, fmt.Sprintf(`"%s"*`, escaped.String()))
	}
	if len(parts) == 0 {
		return ""
	}
	return strings.Join(parts, " AND ")
}

func isValidFTSTerm(term string) bool {
	for _, r := range term {
		if r == '-' || r == '_' {
			continue
		}
		if !unicode.IsLetter(r) && !unicode.IsNumber(r) {
			return false
		}
	}
	return true
}
