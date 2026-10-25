package gopher

import (
	"errors"
	"io"
	"os"
	"strings"
	"unicode"

	"golang.org/x/net/html"

	"warehouse06/internal/staticfiles"
)

func textLines(source string) ([]string, error) {
	doc, err := html.Parse(strings.NewReader(source))
	if err != nil {
		return nil, err
	}
	var out strings.Builder
	var visit func(*html.Node)
	visit = func(n *html.Node) {
		if n.Type == html.ElementNode && (n.Data == "script" || n.Data == "style") {
			return
		}
		block := n.Type == html.ElementNode && strings.Contains("|p|div|section|article|h1|h2|h3|h4|h5|h6|ul|ol|blockquote|pre|table|tr|", "|"+n.Data+"|")
		if block {
			out.WriteString("\n\n")
		}
		if n.Type == html.ElementNode && n.Data == "br" {
			out.WriteByte('\n')
		}
		if n.Type == html.ElementNode && n.Data == "li" {
			out.WriteString("\n- ")
		}
		if n.Type == html.TextNode {
			out.WriteString(n.Data)
		}
		for child := n.FirstChild; child != nil; child = child.NextSibling {
			visit(child)
		}
		if block {
			out.WriteString("\n\n")
		}
	}
	visit(doc)
	var lines []string
	blank := false
	for _, line := range strings.Split(strings.TrimSpace(out.String()), "\n") {
		line = strings.Join(strings.Fields(clean(line)), " ")
		if line == "" {
			if !blank {
				lines = append(lines, "")
			}
			blank = true
			continue
		}
		blank = false
		lines = append(lines, line)
	}
	return lines, nil
}

// wrapTextLine scans each rune once, including exceptionally long words.
func wrapTextLine(line string) []string {
	runes := []rune(line)
	var lines []string
	for len(runes) > 72 {
		end := 72
		for i := 71; i > 0; i-- {
			if unicode.IsSpace(runes[i]) {
				end = i
				break
			}
		}
		lines = append(lines, string(runes[:end]))
		runes = runes[end:]
		for len(runes) > 0 && unicode.IsSpace(runes[0]) {
			runes = runes[1:]
		}
	}
	return append(lines, string(runes))
}

// textSize measures the existing text presentation: UTF-8 plus wrapped CRLFs,
// excluding protocol dot-stuffing and the final terminator.
func textSize(lines []string) int {
	size := 0
	for _, line := range lines {
		for _, wrapped := range wrapTextLine(line) {
			size += len(wrapped) + 2
		}
	}
	return size
}

func writeText(w io.Writer, lines []string) error {
	var wire strings.Builder
	for _, line := range lines {
		for _, wrapped := range wrapTextLine(line) {
			writeTextLine(&wire, wrapped)
		}
	}
	wire.WriteString(".\r\n")
	_, err := io.WriteString(w, wire.String())
	return err
}

func writeTextLine(w *strings.Builder, line string) {
	if strings.HasPrefix(line, ".") {
		w.WriteByte('.')
	}
	w.WriteString(line)
	w.WriteString("\r\n")
}

func (s *Server) download(w io.Writer, path string) error {
	if !staticfiles.IsStorageAssetPath(path) {
		return errors.New("file type is not allowed")
	}
	if _, err := staticfiles.ResolveUnderRoot(s.cfg.StorageDir, path); err != nil {
		return err
	}
	// OpenRoot also blocks traversal through symlinked parent directories and
	// keeps that guarantee if filesystem links change during the request.
	root, err := os.OpenRoot(s.cfg.StorageDir)
	if err != nil {
		return err
	}
	defer func() { _ = root.Close() }()
	file, err := root.Open(strings.TrimPrefix(path, "/"))
	if err != nil {
		return err
	}
	defer file.Close()
	info, err := file.Stat()
	if err != nil {
		return err
	}
	if !info.Mode().IsRegular() {
		return errors.New("not a regular file")
	}
	_, err = io.Copy(w, file)
	return err
}
