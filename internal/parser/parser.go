package parser

import (
	"bytes"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/yuin/goldmark"
	"github.com/yuin/goldmark/extension"
	"github.com/yuin/goldmark/parser"
	"github.com/yuin/goldmark/renderer/html"
	"go.uber.org/zap"
	"gopkg.in/yaml.v3"

	"warehouse06/internal/domain"
	"warehouse06/internal/htmlsanitize"
	"warehouse06/internal/pathutil"
)

const maxReadmeBytes = 2 << 20 // 2 MiB

type Parser struct {
	log        *zap.Logger
	md         goldmark.Markdown
	storageDir string
}

func NewParser(storageDir string, log *zap.Logger) *Parser {
	if resolved, err := filepath.EvalSymlinks(storageDir); err == nil {
		storageDir = resolved
	}

	md := goldmark.New(
		goldmark.WithExtensions(extension.GFM),
		goldmark.WithParserOptions(
			parser.WithAutoHeadingID(),
		),
		goldmark.WithRendererOptions(
			html.WithHardWraps(),
			html.WithXHTML(),
		),
	)

	return &Parser{
		log:        log,
		md:         md,
		storageDir: storageDir,
	}
}

type Frontmatter struct {
	Name        string      `yaml:"name"`
	Tags        []string    `yaml:"tags"`
	Authors     []string    `yaml:"authors"`
	Screenshots []string    `yaml:"screenshots"`
	Date        interface{} `yaml:"date"`
	Youtube     interface{} `yaml:"youtube"`
	Require     []string    `yaml:"require"`
	Address     string      `yaml:"address"` // For authors
}

func (p *Parser) ParseFile(path string) (*domain.Entry, *Frontmatter, error) {
	if err := pathutil.UnderRoot(p.storageDir, path); err != nil {
		return nil, nil, err
	}

	content, err := readFileLimited(path, maxReadmeBytes)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to read file: %w", err)
	}

	fm, mdContent, err := p.extractFrontmatter(content)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to extract frontmatter: %w", err)
	}

	relPath, err := filepath.Rel(p.storageDir, filepath.Dir(path))
	if err != nil {
		relPath = filepath.Dir(path)
	}
	if relPath == "." {
		relPath = ""
	}

	var buf bytes.Buffer
	if err := p.md.Convert(mdContent, &buf); err != nil {
		return nil, nil, fmt.Errorf("failed to convert markdown: %w", err)
	}
	contentHTML := htmlsanitize.Sanitize(rewriteResourcePaths(buf.String(), relPath))

	var dateStr string
	if fm.Date != nil {
		dateStr = fmt.Sprint(fm.Date)
	}

	var youtubeStr string
	if fm.Youtube != nil {
		switch v := fm.Youtube.(type) {
		case string:
			youtubeStr = v
		case []interface{}:
			if len(v) > 0 {
				youtubeStr = fmt.Sprint(v[0])
			}
		}
	}

	entry := &domain.Entry{
		Path:        relPath,
		Name:        fm.Name,
		ContentHTML: contentHTML,
		Date:        dateStr,
		Youtube:     youtubeStr,
		Type:        domain.EntryTypeDirectory,
	}

	// Simple description extraction (first paragraph or so)
	entry.Description = p.extractDescription(mdContent)

	return entry, fm, nil
}

var attrPattern = regexp.MustCompile(`\b(src|href)="([^"]+)"`)

func rewriteResourcePaths(htmlContent, entryPath string) string {
	if entryPath == "" {
		return htmlContent
	}

	return attrPattern.ReplaceAllStringFunc(htmlContent, func(match string) string {
		parts := attrPattern.FindStringSubmatch(match)
		if len(parts) != 3 || !isRelativeResource(parts[2]) {
			return match
		}

		cleanPath := strings.TrimPrefix(parts[2], "./")
		joined := filepath.ToSlash(filepath.Clean(filepath.Join(entryPath, cleanPath)))
		if strings.HasPrefix(joined, "..") || strings.Contains(joined, "/../") {
			return match
		}
		rewritten := "/" + strings.TrimPrefix(joined, "/")
		return fmt.Sprintf(`%s="%s"`, parts[1], rewritten)
	})
}

func isRelativeResource(value string) bool {
	if value == "" ||
		strings.HasPrefix(value, "#") ||
		strings.HasPrefix(value, "/") ||
		strings.HasPrefix(value, "http://") ||
		strings.HasPrefix(value, "https://") ||
		strings.HasPrefix(value, "mailto:") ||
		strings.HasPrefix(value, "data:") {
		return false
	}
	return true
}

func (p *Parser) extractFrontmatter(content []byte) (*Frontmatter, []byte, error) {
	fm := &Frontmatter{}

	if !bytes.HasPrefix(content, []byte("---\n")) {
		return fm, content, nil
	}

	parts := bytes.SplitN(content, []byte("\n---\n"), 2)
	if len(parts) != 2 {
		return fm, content, nil
	}

	fmContent := parts[0][4:] // Skip initial "---\n"
	mdContent := parts[1]

	if err := yaml.Unmarshal(fmContent, fm); err != nil {
		return nil, nil, err
	}

	// Handle single string require as slice
	var raw map[string]interface{}
	if err := yaml.Unmarshal(fmContent, &raw); err == nil {
		if req, ok := raw["require"]; ok {
			switch v := req.(type) {
			case string:
				fm.Require = []string{v}
			case []interface{}:
				fm.Require = make([]string, len(v))
				for i, val := range v {
					fm.Require[i] = fmt.Sprint(val)
				}
			}
		}
	}

	return fm, mdContent, nil
}

func (p *Parser) extractDescription(mdContent []byte) string {
	lines := strings.Split(string(mdContent), "\n")
	var descLines []string
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			if len(descLines) > 0 {
				break
			}
			continue
		}
		if strings.HasPrefix(line, "#") || strings.HasPrefix(line, "!") || strings.HasPrefix(line, "[") {
			continue
		}
		descLines = append(descLines, line)
		if len(descLines) >= 3 { // Max 3 lines for description
			break
		}
	}
	return strings.Join(descLines, " ")
}

func (p *Parser) ScanDirectory() ([]*domain.Entry, []*domain.Author, error) {
	var entries []*domain.Entry
	var authors []*domain.Author

	err := filepath.WalkDir(p.storageDir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		// Skip hidden directories (like .git)
		if d.IsDir() && strings.HasPrefix(d.Name(), ".") {
			return fs.SkipDir
		}

		if !d.IsDir() && d.Name() == "README.md" {
			if err := pathutil.UnderRoot(p.storageDir, path); err != nil {
				p.log.Warn("skipping file outside storage", zap.String("path", path), zap.Error(err))
				return nil
			}
			entry, fm, err := p.ParseFile(path)
			if err != nil {
				p.log.Error("failed to parse file", zap.String("path", path), zap.Error(err))
				return nil
			}

			// Check if it's an author
			relPath := entry.Path
			if strings.HasPrefix(relPath, "authors/") {
				author := &domain.Author{
					DirectoryName: filepath.Base(relPath),
					Name:          fm.Name,
					Address:       fm.Address,
					ContentHTML:   entry.ContentHTML,
				}
				authors = append(authors, author)
				return nil // Don't add authors to entries list directly
			}

			// Populate relations for entry
			for _, t := range fm.Tags {
				entry.Tags = append(entry.Tags, domain.Tag{Name: t})
			}
			for _, a := range fm.Authors {
				entry.Authors = append(entry.Authors, domain.Author{DirectoryName: a})
			}
			for _, s := range fm.Screenshots {
				entry.Screenshots = append(entry.Screenshots, domain.File{
					Filename: s,
					Filepath: filepath.Join(entry.Path, s),
					IsImage:  true,
				})
			}
			entry.Requires = fm.Require

			// Find other files in the same directory
			dir := filepath.Dir(path)
			entriesInDir, readErr := os.ReadDir(dir)
			if readErr != nil {
				p.log.Warn("read entry directory", zap.String("dir", dir), zap.Error(readErr))
			}
			for _, e := range entriesInDir {
				if e.IsDir() || e.Name() == "README.md" {
					continue
				}

				isImage := strings.HasSuffix(strings.ToLower(e.Name()), ".png") ||
					strings.HasSuffix(strings.ToLower(e.Name()), ".jpg") ||
					strings.HasSuffix(strings.ToLower(e.Name()), ".jpeg") ||
					strings.HasSuffix(strings.ToLower(e.Name()), ".gif")

				// Skip screenshots as they are already added
				isScreenshot := false
				for _, s := range entry.Screenshots {
					if s.Filename == e.Name() {
						isScreenshot = true
						break
					}
				}

				if !isScreenshot {
					entry.Files = append(entry.Files, domain.File{
						Filename: e.Name(),
						Filepath: filepath.Join(entry.Path, e.Name()),
						IsImage:  isImage,
					})
				}
			}

			entries = append(entries, entry)
		}

		return nil
	})

	return entries, authors, err
}

func readFileLimited(path string, maxBytes int64) ([]byte, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	limited := io.LimitReader(f, maxBytes+1)
	data, err := io.ReadAll(limited)
	if err != nil {
		return nil, err
	}
	if int64(len(data)) > maxBytes {
		return nil, fmt.Errorf("file exceeds %d byte limit", maxBytes)
	}
	return data, nil
}
