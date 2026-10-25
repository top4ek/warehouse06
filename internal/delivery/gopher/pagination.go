package gopher

import (
	"errors"
	"io"
	"net/url"
	"strconv"
	"strings"
	"unicode"
	"unicode/utf8"

	"warehouse06/internal/domain"
)

const textPageSize = 24 * 1024

// Three maximum-size navigation lines and the terminator fit on every page.
const textMenuBytes = 32*1024 - 3*255 - 3
const textMenuLines = 800 - 3 - 1

func (s *Server) writeDocument(w io.Writer, entry *domain.Entry, q url.Values) error {
	page := 0
	if values, ok := q["page"]; ok {
		if len(values) != 1 || values[0] == "" {
			return errors.New("invalid page")
		}
		var err error
		page, err = strconv.Atoi(values[0])
		if err != nil || page < 0 {
			return errors.New("invalid page")
		}
	}
	lines, err := textLines(entry.ContentHTML)
	if err != nil {
		return err
	}
	if textSize(lines) <= textPageSize {
		if page != 0 {
			return errors.New("invalid page")
		}
		return writeText(w, lines)
	}
	pages := s.textPages(lines)
	if page >= len(pages) {
		return errors.New("invalid page")
	}
	var out strings.Builder
	for _, line := range pages[page] {
		out.WriteString(s.line('i', line, ""))
	}
	navigation := []item{}
	base := link("t", entry.ID, entry.Path)
	if page > 0 {
		navigation = append(navigation, item{'1', "Предыдущая страница", base + "&page=" + strconv.Itoa(page-1)})
	}
	if page+1 < len(pages) {
		navigation = append(navigation, item{'1', "Следующая страница", base + "&page=" + strconv.Itoa(page+1)})
	}
	navigation = append(navigation, item{'1', "К карточке", link("e", entry.ID, entry.Path)})
	for _, it := range navigation {
		line, err := s.menuLine(it)
		if err != nil {
			return err
		}
		out.WriteString(line)
	}
	out.WriteString(".\r\n")
	_, err = io.WriteString(w, out.String())
	return err
}

// menuTextLines wraps content to both display columns and actual wire bytes.
// It never sends content through menuLine's label truncation.
func (s *Server) menuTextLines(line string) []string {
	maxBytes := 255 - len(s.line('i', "", ""))
	var result []string
	for line != "" {
		end, columns, space := 0, 0, -1
		for offset, r := range line {
			n := displayWidth(r)
			size := utf8.RuneLen(r)
			if columns+n > 72 || offset+size > maxBytes {
				break
			}
			columns += n
			end = offset + size
			if r == ' ' {
				space = offset
			}
		}
		if end < len(line) && space > 0 {
			end = space
		}
		result = append(result, line[:end])
		line = strings.TrimLeft(line[end:], " ")
	}
	if len(result) == 0 {
		return []string{""}
	}
	return result
}

func (s *Server) textPages(lines []string) [][]string {
	var pages [][]string
	var current []string
	bytes, textBytes := 0, 0
	flush := func() {
		if len(current) > 0 {
			pages = append(pages, current)
		}
		current = nil
		bytes, textBytes = 0, 0
	}
	// A paragraph includes its following blank separator, keeping order intact.
	for start := 0; start < len(lines); {
		end := start + 1
		for end < len(lines) && lines[end-1] != "" {
			end++
		}
		paragraph := lines[start:end]
		units := [][]string{paragraph}
		wrapped, wireBytes, contentBytes := s.menuTextUnit(paragraph)
		if contentBytes > textPageSize || wireBytes > textMenuBytes || len(wrapped) > textMenuLines {
			units = nil
			for _, sentence := range sentenceParts(strings.TrimSpace(strings.Join(paragraph, "\n"))) {
				units = append(units, strings.Split(sentence, "\n"))
			}
			if paragraph[len(paragraph)-1] == "" && len(units) > 0 {
				last := len(units) - 1
				units[last] = append(units[last], "")
			}
		}
		for _, unit := range units {
			wrapped, wireBytes, contentBytes := s.menuTextUnit(unit)
			if len(current) > 0 && (textBytes+contentBytes > textPageSize || bytes+wireBytes > textMenuBytes || len(current)+len(wrapped) > textMenuLines) {
				flush()
			}
			for i, line := range wrapped {
				size := len(s.line('i', line, ""))
				// Keep a paragraph separator with its final content line. Otherwise
				// a hard limit followed by a large unit can create a blank-only page.
				reserveBytes, reserveLines := 0, 0
				if i+1 < len(wrapped) && wrapped[i+1] == "" {
					reserveBytes = len(s.line('i', "", ""))
					reserveLines = 1
				}
				if bytes+size+reserveBytes > textMenuBytes || len(current)+1+reserveLines > textMenuLines {
					flush()
				}
				current = append(current, line)
				bytes += size
				textBytes += len(line) + 2
			}
		}
		start = end
	}
	flush()
	return pages
}

func (s *Server) menuTextUnit(lines []string) (wrapped []string, wireBytes, contentBytes int) {
	for _, line := range lines {
		for _, part := range s.menuTextLines(line) {
			wrapped = append(wrapped, part)
			wireBytes += len(s.line('i', part, ""))
			contentBytes += len(part) + 2
		}
	}
	return
}

// Sentence boundaries are punctuation followed by whitespace, optionally after
// closing quotes/brackets. This is deliberately a heuristic, not language parsing.
func sentenceParts(text string) []string {
	var parts []string
	start := 0
	ending := false
	for offset, r := range text {
		if ending && unicode.IsSpace(r) {
			if part := strings.TrimSpace(text[start:offset]); part != "" {
				parts = append(parts, part)
			}
			start = offset
			ending = false
			continue
		}
		if strings.ContainsRune(".!?…", r) {
			ending = true
			continue
		}
		if ending && strings.ContainsRune("\"'»”’)]}", r) {
			continue
		}
		ending = false
	}
	if part := strings.TrimSpace(text[start:]); part != "" {
		parts = append(parts, part)
	}
	return parts
}
