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

func (s *Server) writeText(w io.Writer, source string) error {
	doc, err := html.Parse(strings.NewReader(source))
	if err != nil {
		return err
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
	var wire strings.Builder
	blank := false
	for _, line := range strings.Split(strings.TrimSpace(out.String()), "\n") {
		line = strings.Join(strings.Fields(clean(line)), " ")
		if line == "" {
			if !blank {
				wire.WriteString("\r\n")
			}
			blank = true
			continue
		}
		blank = false
		// Wrap at 72 characters without splitting UTF-8; retain list markers.
		for len([]rune(line)) > 72 {
			runes := []rune(line)
			end := 72
			for i := 71; i > 0; i-- {
				if unicode.IsSpace(runes[i]) {
					end = i
					break
				}
			}
			writeTextLine(&wire, string(runes[:end]))
			line = strings.TrimSpace(string(runes[end:]))
		}
		writeTextLine(&wire, line)
	}
	wire.WriteString(".\r\n")
	_, err = io.WriteString(w, wire.String())
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
