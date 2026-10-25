package gopher

import (
	"context"
	"crypto/sha256"
	"errors"
	"fmt"
	"io"
	"net/url"
	"regexp"
	"strconv"
	"strings"
	"unicode"
	"unicode/utf8"

	"golang.org/x/text/width"

	"warehouse06/internal/domain"
	"warehouse06/internal/repository"
	"warehouse06/internal/staticfiles"
)

const pageSize = 50

var sha256Pattern = regexp.MustCompile(`^[0-9a-fA-F]{8,64}$`)

type item struct {
	kind           byte
	name, selector string
}

func fingerprint(value string) string { return fmt.Sprintf("%x", sha256.Sum256([]byte(value)))[:16] }

func link(kind string, id int64, value string) string {
	return fmt.Sprintf("/%s/%d?v=%s", kind, id, fingerprint(value))
}

func clean(value string) string {
	return strings.Map(func(r rune) rune {
		if unicode.IsControl(r) {
			return ' '
		}
		return r
	}, strings.ToValidUTF8(value, "?"))
}

// menuLine preserves selectors verbatim and trims only the display name.
func (s *Server) menuLine(it item) (string, error) {
	if strings.IndexFunc(it.selector, unicode.IsControl) >= 0 {
		return "", errors.New("invalid selector")
	}
	name := []rune(displayName(it.name))
	for len(name) > 0 && len(s.line(it.kind, string(name), it.selector)) > 255 {
		name = name[:len(name)-1]
	}
	line := s.line(it.kind, string(name), it.selector)
	if len(line) > 255 {
		return "", errors.New("selector exceeds menu line limit")
	}
	return line, nil
}

func displayWidth(r rune) int {
	if k := width.LookupRune(r).Kind(); k == width.EastAsianWide || k == width.EastAsianFullwidth {
		return 2
	}
	if unicode.Is(unicode.Mn, r) || unicode.Is(unicode.Me, r) {
		return 0
	}
	return 1
}

func displayName(value string) string {
	var out strings.Builder
	columns := 0
	for _, r := range clean(value) {
		n := displayWidth(r)
		if columns+n > 72 {
			break
		}
		columns += n
		out.WriteRune(r)
	}
	return out.String()
}

func withOffset(u *url.URL, n int) string {
	v := *u
	q := v.Query()
	q.Set("offset", strconv.Itoa(n))
	v.RawQuery = q.Encode()
	return v.String()
}

// writeMenu paginates by item count and actual wire bytes, including navigation.
func (s *Server) writeMenu(w io.Writer, u *url.URL, items []item, offset, total int) error {
	var out strings.Builder
	root, _ := s.menuLine(item{'1', "Главное меню", "/"})
	out.WriteString(root)
	if offset > 0 {
		prev := offset - pageSize
		if prev < 0 {
			prev = 0
		}
		line, err := s.menuLine(item{'1', "Предыдущая страница", withOffset(u, prev)})
		if err != nil {
			return err
		}
		out.WriteString(line)
	}
	count := 0
	for _, it := range items {
		if count >= pageSize {
			break
		}
		line, err := s.menuLine(it)
		if err != nil {
			return err
		}
		// Reserve one maximum-sized line for Next and the final dot.
		if out.Len()+len(line)+255+3 > 32*1024 {
			break
		}
		out.WriteString(line)
		count++
	}
	if offset+count < total {
		line, err := s.menuLine(item{'1', "Следующая страница", withOffset(u, offset+count)})
		if err != nil {
			return err
		}
		out.WriteString(line)
	}
	if total == 0 {
		line, _ := s.menuLine(item{'i', "Ничего не найдено", ""})
		out.WriteString(line)
	}
	out.WriteString(".\r\n")
	_, err := io.WriteString(w, out.String())
	return err
}

func (s *Server) respond(ctx context.Context, w io.Writer, request string) error {
	selector, query, hasQuery := strings.Cut(request, "\t")
	if strings.IndexFunc(selector, unicode.IsControl) >= 0 || strings.Contains(query, "\t") || !utf8.ValidString(query) {
		return errors.New("invalid request")
	}
	if selector == "" {
		selector = "/"
	}
	u, err := url.ParseRequestURI(selector)
	if err != nil {
		return err
	}
	q, err := url.ParseQuery(u.RawQuery)
	if err != nil {
		return err
	}
	if hasQuery {
		if u.Path != "/search" {
			return errors.New("unexpected search query")
		}
		q.Set("q", query)
		u.RawQuery = q.Encode()
	}
	offset := 0
	if raw := q.Get("offset"); raw != "" {
		offset, err = strconv.Atoi(raw)
		if err != nil || offset < 0 || offset > int(^uint(0)>>1)-pageSize {
			return errors.New("invalid offset")
		}
	}
	repo := s.holder.Get()
	if repo == nil {
		return errors.New("catalog unavailable")
	}
	var items []item
	var entries []*domain.Entry
	var total int
	opts := repository.EntryListOptions{Limit: pageSize, Offset: offset, Sort: "name", Order: "asc", Platform: q.Get("platform"), Author: q.Get("author"), Tag: q.Get("tag")}
	parts := strings.Split(strings.TrimPrefix(u.Path, "/"), "/")
	if len(parts) == 2 {
		id, e := strconv.ParseInt(parts[1], 10, 64)
		if e != nil || id < 1 {
			return errors.New("invalid id")
		}
		switch parts[0] {
		case "e", "t":
			entry, e := repo.EntryByID(ctx, id)
			if e != nil {
				return e
			}
			if q.Get("v") != fingerprint(entry.Path) {
				return errors.New("stale selector")
			}
			if parts[0] == "t" {
				return s.writeDocument(w, entry, q)
			}
			items, err = s.card(ctx, repo, entry)
		case "p", "a", "g":
			value, e := repo.GopherFilter(ctx, parts[0], id)
			if e != nil {
				return e
			}
			if q.Get("v") != fingerprint(value) {
				return errors.New("stale selector")
			}
			switch parts[0] {
			case "p":
				opts.Platform = value
			case "a":
				opts.Author = value
			case "g":
				opts.Tag = value
			}
			entries, err = repo.GetEntries(ctx, opts)
			if err != nil {
				return err
			}
			total, err = repo.CountEntries(ctx, opts)
		case "f":
			file, e := repo.FileByID(ctx, id)
			if e != nil {
				return e
			}
			if q.Get("v") != fingerprint(file.Filepath) {
				return errors.New("stale selector")
			}
			return s.download(w, file.Filepath)
		default:
			return errors.New("unknown selector")
		}
	} else {
		switch u.Path {
		case "/":
			items = []item{{'1', "Платформы", "/platforms"}, {'1', "Авторы", "/authors"}, {'1', "Теги", "/tags"}, {'7', "Поиск", "/search"}}
		case "/platforms":
			list, e := repo.ListPlatforms(ctx)
			if e != nil {
				return e
			}
			for _, p := range list {
				id, e := repo.EntryIDByPath(ctx, p.Path)
				if e != nil {
					return e
				}
				items = append(items, item{'1', p.Name, link("p", id, p.Path)})
			}
		case "/authors":
			list, e := repo.ListAuthors(ctx)
			if e != nil {
				return e
			}
			for _, a := range list {
				name := a.Name
				if name == "" {
					name = a.DirectoryName
				}
				items = append(items, item{'1', name, link("a", a.ID, a.DirectoryName)})
			}
		case "/tags":
			list, e := repo.ListTags(ctx)
			if e != nil {
				return e
			}
			for _, t := range list {
				items = append(items, item{'1', t.Name, link("g", t.ID, t.Name)})
			}
		case "/entries":
			entries, err = repo.GetEntries(ctx, opts)
			if err != nil {
				return err
			}
			total, err = repo.CountEntries(ctx, opts)
		case "/search":
			raw := strings.TrimSpace(q.Get("q"))
			if key := q.Get("key"); key != "" && !hasQuery {
				raw, err = s.searchQuery(key)
				if err != nil {
					return err
				}
			}
			if raw == "" {
				items = []item{{'7', "Введите поисковый запрос", "/search"}}
				break
			}
			u = s.searchURL(raw)
			if sha256Pattern.MatchString(raw) {
				entries, err = repo.SearchEntriesBySHA256(ctx, strings.ToLower(raw), pageSize, offset)
				if err != nil {
					return err
				}
				total, err = repo.CountEntriesBySHA256(ctx, strings.ToLower(raw))
				break
			}
			fts := repository.FormatFTSQuery(raw)
			if fts == "" {
				return errors.New("invalid query")
			}
			entries, err = repo.SearchEntries(ctx, fts, pageSize, offset)
			if err != nil {
				return err
			}
			total, err = repo.CountSearchEntries(ctx, fts)
		case "/entry", "/text":
			entry, e := repo.GetEntryByPath(ctx, q.Get("path"))
			if e != nil {
				return e
			}
			if u.Path == "/text" {
				return s.writeDocument(w, entry, q)
			}
			items, err = s.card(ctx, repo, entry)
			u.Path = fmt.Sprintf("/e/%d", entry.ID)
			u.RawQuery = url.Values{"v": {fingerprint(entry.Path)}}.Encode()
		case "/file":
			return s.download(w, q.Get("path"))
		default:
			return errors.New("unknown selector")
		}
	}
	if err != nil {
		return err
	}
	if entries != nil {
		for _, e := range entries {
			items = append(items, item{'1', e.Name, link("e", e.ID, e.Path)})
		}
	} else {
		total = len(items)
		if offset > total {
			items = nil
		} else {
			items = items[offset:]
		}
	}
	return s.writeMenu(w, u, items, offset, total)
}

func (s *Server) card(ctx context.Context, repo *repository.SQLiteRepository, e *domain.Entry) ([]item, error) {
	items := []item{{'i', e.Name, ""}}
	items = append(items, s.descriptionItems(e.Description)...)
	lines, err := textLines(e.ContentHTML)
	if err != nil {
		return nil, err
	}
	kind := byte('0')
	if textSize(lines) > textPageSize {
		kind = '1'
	}
	items = append(items, item{kind, "Полный текст", link("t", e.ID, e.Path)})
	for _, a := range e.Authors {
		items = append(items, item{'1', "Автор: " + a.Name, link("a", a.ID, a.DirectoryName)})
	}
	for _, t := range e.Tags {
		items = append(items, item{'1', "Тег: " + t.Name, link("g", t.ID, t.Name)})
	}
	for _, f := range e.Files {
		if staticfiles.IsStorageAssetPath(f.Filepath) {
			items = append(items, item{'9', f.Filename, link("f", f.ID, f.Filepath)})
		}
	}
	for _, d := range e.Directories {
		id, err := repo.EntryIDByPath(ctx, d.Path)
		if err != nil {
			return nil, err
		}
		items = append(items, item{'1', d.Name, link("e", id, d.Path)})
	}
	return items, nil
}

// Descriptions are content, so wrap them instead of truncating as menu labels.
func (s *Server) descriptionItems(description string) []item {
	remaining := strings.Join(strings.Fields(clean(description)), " ")
	var items []item
	maxBytes := 255 - len(s.line('i', "", ""))
	for remaining != "" {
		line := displayName(remaining)
		for len(line) > maxBytes {
			_, size := utf8.DecodeLastRuneInString(line)
			line = line[:len(line)-size]
		}
		if len(line) < len(remaining) {
			if space := strings.LastIndexByte(line, ' '); space > 0 {
				line = line[:space]
			}
		}
		items = append(items, item{'i', line, ""})
		remaining = strings.TrimSpace(remaining[len(line):])
	}
	return items
}
