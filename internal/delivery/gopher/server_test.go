package gopher_test

import (
	"fmt"
	"io"
	"net"
	"net/url"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"time"
	"unicode/utf8"

	"github.com/stretchr/testify/require"
	"go.uber.org/zap"

	"warehouse06/internal/delivery/gopher"
	"warehouse06/internal/domain"
	"warehouse06/internal/repository"
)

type tcpServer struct {
	server  *gopher.Server
	addr    string
	holder  *repository.Holder
	storage string
}

func startServer(t *testing.T) *tcpServer {
	t.Helper()
	repo, err := repository.NewSQLiteRepository(":memory:", zap.NewNop())
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, repo.Close()) })
	holder := repository.NewHolder(repo, ":memory:")
	storage := t.TempDir()
	srv, err := gopher.New(gopher.Config{Host: "gopher.example", Port: 70, StorageDir: storage}, holder, zap.NewNop())
	require.NoError(t, err)
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)
	done := make(chan error, 1)
	go func() { done <- srv.Serve(listener) }()
	t.Cleanup(func() {
		require.NoError(t, srv.Close())
		select {
		case err := <-done:
			require.NoError(t, err)
		case <-time.After(3 * time.Second):
			t.Error("Gopher Serve did not stop after Close")
		}
	})
	return &tcpServer{server: srv, addr: listener.Addr().String(), holder: holder, storage: storage}
}

func (s *tcpServer) request(t *testing.T, selector string) string {
	t.Helper()
	return s.rawRequest(t, selector+"\r\n")
}

func (s *tcpServer) rawRequest(t *testing.T, request string) string {
	t.Helper()
	conn, err := net.DialTimeout("tcp", s.addr, time.Second)
	require.NoError(t, err)
	defer func() { _ = conn.Close() }()
	require.NoError(t, conn.SetDeadline(time.Now().Add(3*time.Second)))
	_, err = io.WriteString(conn, request)
	require.NoError(t, err)
	data, err := io.ReadAll(conn)
	require.NoError(t, err)
	return string(data)
}

type menuItem struct {
	kind           byte
	name, selector string
}

func menu(t *testing.T, response string) []menuItem {
	t.Helper()
	require.True(t, strings.HasSuffix(response, "\r\n.\r\n"), "menu must end with a dot line: %q", response)
	lines := strings.Split(strings.TrimSuffix(response, ".\r\n"), "\r\n")
	var items []menuItem
	for _, line := range lines {
		if line == "" {
			continue
		}
		fields := strings.Split(line, "\t")
		require.Len(t, fields, 4, "invalid menu line: %q", line)
		require.NotEmpty(t, fields[0])
		require.Equal(t, "gopher.example", fields[2])
		require.Equal(t, "70", fields[3])
		items = append(items, menuItem{kind: fields[0][0], name: fields[0][1:], selector: fields[1]})
	}
	return items
}

func TestRootNavigation(t *testing.T) {
	s := startServer(t)
	for _, root := range []string{"", "/"} {
		items := menu(t, s.request(t, root))
		for _, selector := range []string{"/platforms", "/authors", "/tags", "/search"} {
			found := false
			for _, item := range items {
				if item.selector == selector {
					found = true
					if selector == "/search" {
						require.Equal(t, byte('7'), item.kind)
					} else {
						require.Equal(t, byte('1'), item.kind)
					}
				}
			}
			require.True(t, found, "root is missing %s", selector)
		}
	}
}

func saveEntries(t *testing.T, s *tcpServer, entries ...*domain.Entry) {
	t.Helper()
	require.NoError(t, s.holder.Get().SaveEntriesAndAuthors(t.Context(), entries, []*domain.Author{{DirectoryName: "pilot", Name: "Пилот"}}))
}

func findItem(t *testing.T, items []menuItem, name string) menuItem {
	t.Helper()
	for _, item := range items {
		if strings.Contains(item.name, name) {
			return item
		}
	}
	t.Fatalf("menu has no item containing %q: %+v", name, items)
	return menuItem{}
}

func TestNavigateCatalogToCard(t *testing.T) {
	s := startServer(t)
	saveEntries(t, s,
		&domain.Entry{Path: "vector06c", Name: "Вектор-06Ц", Type: domain.EntryTypeDirectory},
		&domain.Entry{Path: "vector06c/elite", Name: "Элита", Description: "Космическая игра", Type: domain.EntryTypeDirectory,
			Authors: []domain.Author{{DirectoryName: "pilot"}}, Tags: []domain.Tag{{Name: "Космос"}}},
		&domain.Entry{Path: "bk/other", Name: "Другая программа", Type: domain.EntryTypeDirectory})
	for _, section := range []struct{ selector, label string }{
		{"/platforms", "Вектор-06Ц"}, {"/authors", "Пилот"}, {"/tags", "Космос"},
	} {
		t.Run(section.selector, func(t *testing.T) {
			filter := findItem(t, menu(t, s.request(t, section.selector)), section.label)
			entries := menu(t, s.request(t, filter.selector))
			for _, item := range entries {
				require.NotContains(t, item.name, "Другая программа")
			}
			entry := findItem(t, entries, "Элита")
			card := s.request(t, entry.selector)
			menu(t, card)
			for _, text := range []string{"Элита", "Космическая игра", "Пилот", "Космос"} {
				require.Contains(t, card, text)
			}
		})
	}
}

func TestInvalidSelectorsAreGopherErrors(t *testing.T) {
	s := startServer(t)
	for _, selector := range []string{
		"/unknown", "/entry?path=missing", "/file?path=missing.rom",
		"/entries?offset=-1", "/entries?offset=invalid", "/entries?offset=999999999999999999999999",
		"/entry?path=%GG", "/\x00", "/\xff", "/\tquery",
	} {
		t.Run(selector, func(t *testing.T) {
			items := menu(t, s.request(t, selector))
			require.NotEmpty(t, items)
			require.Equal(t, byte('3'), items[0].kind, "invalid request must return an error item")
		})
	}
}

func TestHighOffsetIsValidAndReturnsEmptyPage(t *testing.T) {
	s := startServer(t)
	for _, offset := range []string{"100001", strconv.Itoa(int(^uint(0)>>1) - 50)} {
		response := s.request(t, "/entries?offset="+offset)
		items := menu(t, response)
		findItem(t, items, "Ничего не найдено")
		for _, item := range items {
			require.NotEqual(t, byte('3'), item.kind)
		}
	}
}

func TestCloseInterruptsIdleConnections(t *testing.T) {
	s := startServer(t)
	conn, err := net.DialTimeout("tcp", s.addr, time.Second)
	require.NoError(t, err)
	defer func() { _ = conn.Close() }()
	require.NoError(t, conn.SetDeadline(time.Now().Add(3*time.Second)))
	_, err = io.WriteString(conn, "/")
	require.NoError(t, err)
	// A complete request also ensures the accept loop has progressed past the idle client.
	menu(t, s.request(t, "/"))
	done := make(chan error, 1)
	go func() { done <- s.server.Close() }()
	select {
	case err := <-done:
		require.NoError(t, err)
	case <-time.After(3 * time.Second):
		t.Fatal("Close did not interrupt an incomplete request")
	}
	_, err = io.ReadAll(conn)
	if netErr, ok := err.(net.Error); ok {
		require.False(t, netErr.Timeout(), "idle client remained open after shutdown")
	}
}

func TestFullTextPreservesParagraphsAndLists(t *testing.T) {
	s := startServer(t)
	saveEntries(t, s, &domain.Entry{Path: "vector06c/elite", Name: "Элита", Type: domain.EntryTypeDirectory,
		ContentHTML: "<p>Первый &amp; второй</p><p>.точка</p><ul><li>Один</li><li>Два</li></ul>"})
	card := menu(t, s.request(t, "/entry?path=vector06c%2Felite"))
	link := findItem(t, card, "Полный текст")
	require.Equal(t, byte('0'), link.kind)
	text := s.request(t, link.selector)
	require.True(t, strings.HasSuffix(text, "\r\n.\r\n"), "text must terminate with dot line: %q", text)
	require.NotContains(t, text, "<p>")
	require.NotContains(t, text, "<li>")
	require.Contains(t, text, "Первый & второй\r\n\r\n")
	require.Contains(t, text, "\r\n..точка\r\n")
	require.Contains(t, text, "Один\r\n")
	require.Contains(t, text, "Два\r\n")
	require.NotContains(t, strings.ReplaceAll(text, "\r\n", ""), "\n", "all line endings must be CRLF")
}

func nextPage(t *testing.T, items []menuItem, current int) string {
	t.Helper()
	for _, item := range items {
		u, err := url.Parse(item.selector)
		require.NoError(t, err)
		offset, err := strconv.Atoi(u.Query().Get("offset"))
		if err == nil && offset > current {
			return item.selector
		}
	}
	return ""
}

func TestSearchPaginatesAndPreservesQuery(t *testing.T) {
	s := startServer(t)
	var entries []*domain.Entry
	for i := 0; i < 55; i++ {
		entries = append(entries, &domain.Entry{Path: fmt.Sprintf("vector06c/game%03d", i), Name: fmt.Sprintf("Галактика 100%% %03d", i), Type: domain.EntryTypeDirectory})
	}
	entries = append(entries, &domain.Entry{Path: "bk/other", Name: "Unrelated", Type: domain.EntryTypeDirectory})
	saveEntries(t, s, entries...)
	first := menu(t, s.request(t, "/search\tГалактика 100%"))
	var seen []string
	for _, item := range first {
		if strings.HasPrefix(item.name, "Галактика 100% ") {
			seen = append(seen, item.name)
		}
	}
	require.Len(t, seen, 50)
	next := nextPage(t, first, 0)
	require.NotEmpty(t, next)
	u, err := url.Parse(next)
	require.NoError(t, err)
	require.Equal(t, "Галактика 100%", u.Query().Get("q"))
	second := menu(t, s.request(t, next))
	for _, item := range second {
		if strings.HasPrefix(item.name, "Галактика 100% ") {
			require.NotContains(t, seen, item.name)
			seen = append(seen, item.name)
		}
		require.NotContains(t, item.name, "Unrelated")
	}
	require.Len(t, seen, 55)
	require.Empty(t, nextPage(t, second, 50))
	card := s.request(t, findItem(t, second, "Галактика 100% ").selector)
	require.Contains(t, card, "Галактика 100% ")
	empty := menu(t, s.request(t, "/search\tNoMatchingEntry"))
	for _, item := range empty {
		require.NotContains(t, item.name, "Галактика 100% ")
		require.NotEqual(t, byte('3'), item.kind)
	}
}

func TestLongSearchQueryPaginatesBothDirections(t *testing.T) {
	s := startServer(t)
	query := strings.Repeat("Галактика ", 215) + " "
	require.Len(t, "/search\t"+query+"\r\n", 4096, "exercise the maximum accepted wire request")
	var entries []*domain.Entry
	for i := 0; i < 55; i++ {
		entries = append(entries, &domain.Entry{Path: fmt.Sprintf("vector06c/game%03d", i),
			Name: fmt.Sprintf("Игра %03d", i), Description: query, Type: domain.EntryTypeDirectory})
	}
	saveEntries(t, s, entries...)
	response := s.request(t, "/search\t"+query)
	assertMenuLimits(t, response)
	first := menu(t, response)
	require.Len(t, first, 52)
	next := nextPage(t, first, 0)
	require.NotEmpty(t, next)
	response = s.request(t, next)
	assertMenuLimits(t, response)
	second := menu(t, response)
	require.Len(t, second, 7)
	for i := 0; i < 5; i++ {
		findItem(t, second, fmt.Sprintf("Игра %03d", i+50))
	}
	previous := findItem(t, second, "Предыдущая страница")
	require.Equal(t, first, menu(t, s.request(t, previous.selector)))
}

func TestLongSearchLinksExpireAfterCacheEviction(t *testing.T) {
	s := startServer(t)
	query := strings.Repeat("Галактика ", 20)
	var entries []*domain.Entry
	for i := 0; i < 51; i++ {
		entries = append(entries, &domain.Entry{Path: fmt.Sprintf("vector06c/game%03d", i),
			Name: fmt.Sprintf("Игра %03d", i), Description: query, Type: domain.EntryTypeDirectory})
	}
	saveEntries(t, s, entries...)
	next := nextPage(t, menu(t, s.request(t, "/search\t"+query)), 0)
	require.NotEmpty(t, next)
	for i := 0; i < 512; i++ {
		s.request(t, "/search\t"+strings.Repeat("я", 60)+strconv.Itoa(i))
	}
	items := menu(t, s.request(t, next))
	require.Equal(t, byte('3'), items[0].kind)
	require.Contains(t, items[0].name, "Search expired")
	// Re-entering the original query creates fresh working page links.
	next = nextPage(t, menu(t, s.request(t, "/search\t"+query)), 0)
	findItem(t, menu(t, s.request(t, next)), "Игра 050")
}

func TestLongDescriptionRemainsReadableAcrossCardPages(t *testing.T) {
	s := startServer(t)
	var words []string
	for i := 0; i < 1000; i++ {
		words = append(words, fmt.Sprintf("Слово%04d", i))
	}
	description := strings.Join(words, " ")
	saveEntries(t, s, &domain.Entry{Path: "vector06c/long", Name: "Описание", Description: description, Type: domain.EntryTypeDirectory})
	selector := "/entry?path=vector06c%2Flong"
	var lines []string
	for page := 0; selector != "" && page < 10; page++ {
		response := s.request(t, selector)
		assertMenuLimits(t, response)
		items := menu(t, response)
		for _, item := range items {
			if item.kind == 'i' && item.name != "Описание" {
				lines = append(lines, item.name)
			}
		}
		u, err := url.Parse(selector)
		require.NoError(t, err)
		offset, _ := strconv.Atoi(u.Query().Get("offset"))
		selector = nextPage(t, items, offset)
	}
	require.Empty(t, selector)
	require.Equal(t, description, strings.Join(lines, " "))
}

func TestDownloadOriginalBytesAndProtectStorage(t *testing.T) {
	s := startServer(t)
	data := []byte{0, 1, 255, '\r', '\n', '.', '\r', '\n', 128, 'X'}
	require.NoError(t, os.MkdirAll(filepath.Join(s.storage, "vector06c", "elite"), 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(s.storage, "vector06c", "elite", "Elite.ROM"), data, 0o644))
	saveEntries(t, s, &domain.Entry{Path: "vector06c/elite", Name: "Элита", Type: domain.EntryTypeDirectory,
		Files: []domain.File{{Filename: "Elite.ROM", Filepath: "vector06c/elite/Elite.ROM"}, {Filename: "blocked.exe", Filepath: "blocked.exe"}}})
	card := menu(t, s.request(t, "/entry?path=vector06c%2Felite"))
	file := findItem(t, card, "Elite.ROM")
	require.Equal(t, byte('9'), file.kind)
	require.Equal(t, data, []byte(s.request(t, file.selector)), "binary transfer must preserve every byte without framing")
	for _, item := range card {
		require.NotContains(t, item.name, "blocked.exe")
	}

	outside := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(outside, "secret.rom"), []byte("SECRET"), 0o644))
	require.NoError(t, os.Symlink(filepath.Join(outside, "secret.rom"), filepath.Join(s.storage, "escape.rom")))
	require.NoError(t, os.Symlink(outside, filepath.Join(s.storage, "escape-dir")))
	require.NoError(t, os.Mkdir(filepath.Join(s.storage, ".hidden"), 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(s.storage, ".hidden", "secret.rom"), []byte("SECRET"), 0o644))
	require.NoError(t, os.WriteFile(filepath.Join(s.storage, "blocked.exe"), []byte("SECRET"), 0o644))
	for _, path := range []string{"../secret.rom", ".hidden/secret.rom", "blocked.exe", "escape.rom", "escape-dir/secret.rom", "missing.rom", "vector06c/elite"} {
		t.Run(path, func(t *testing.T) {
			response := s.request(t, "/file?"+url.Values{"path": {path}}.Encode())
			require.NotContains(t, response, "SECRET")
			items := menu(t, response)
			require.Equal(t, byte('3'), items[0].kind)
		})
	}
}

func TestDownloadAcrossWriteChunksPreservesBytes(t *testing.T) {
	s := startServer(t)
	data := strings.Repeat("\x00\xff\r\n.\r\n\x80original bytes", 32768)
	require.NoError(t, os.WriteFile(filepath.Join(s.storage, "large.rom"), []byte(data), 0o644))
	require.Equal(t, data, s.request(t, "/file?path=large.rom"))
}

func TestSearchRecognizesFileDigest(t *testing.T) {
	s := startServer(t)
	saveEntries(t, s,
		&domain.Entry{Path: "vector06c/elite", Name: "Элита", Type: domain.EntryTypeDirectory,
			Files: []domain.File{{Filename: "elite.rom", Filepath: "vector06c/elite/elite.rom", SHA256: "0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef"}}},
		&domain.Entry{Path: "bk/text", Name: "89ABCDEF", Type: domain.EntryTypeDirectory})
	items := menu(t, s.request(t, "/search\t89ABCDEF"))
	findItem(t, items, "Элита")
	for _, item := range items {
		require.NotContains(t, item.name, "89ABCDEF")
	}
}

func assertMenuLimits(t *testing.T, response string) {
	t.Helper()
	require.True(t, utf8.ValidString(response), "menu must remain valid UTF-8")
	require.LessOrEqual(t, len(response), 32*1024)
	lines := strings.Split(strings.TrimSuffix(response, "\r\n"), "\r\n")
	require.LessOrEqual(t, len(lines), 800)
	for _, line := range lines {
		require.LessOrEqual(t, len(line), 255)
	}
	for _, item := range menu(t, response) {
		require.LessOrEqual(t, utf8.RuneCountInString(item.name), 72)
	}
}

func TestLongUTF8EntriesStayReachableAcrossPages(t *testing.T) {
	s := startServer(t)
	var entries []*domain.Entry
	for i := 104; i >= 0; i-- {
		entries = append(entries, &domain.Entry{Path: "vector06c/" + strings.Repeat("длинный-путь/", 25) + fmt.Sprintf("game%d", i),
			Name: fmt.Sprintf("Игра %03d \t\r\n", i) + strings.Repeat("Ж", 200), Type: domain.EntryTypeDirectory})
	}
	saveEntries(t, s, entries...)
	selector := "/entries?platform=vector06c"
	count := 0
	for page := 0; selector != "" && page < 5; page++ {
		response := s.request(t, selector)
		assertMenuLimits(t, response)
		items := menu(t, response)
		before := count
		for _, item := range items {
			if !strings.HasPrefix(item.name, "Игра ") {
				continue
			}
			require.True(t, strings.HasPrefix(item.name, fmt.Sprintf("Игра %03d ", count)), "entries must remain sorted and unique: %q", item.name)
			count++
			if count == 105 {
				assertMenuLimits(t, s.request(t, item.selector))
			}
		}
		if page == 0 {
			require.Equal(t, 50, count-before)
		}
		selector = nextPage(t, items, before)
		if selector != "" {
			u, err := url.Parse(selector)
			require.NoError(t, err)
			require.Equal(t, "vector06c", u.Query().Get("platform"))
		}
	}
	require.Equal(t, 105, count, "every record must be reachable without damaged selectors")
	require.Empty(t, selector, "paging must terminate")
}

func TestCardPaginatesFilesWithoutLosingDownloads(t *testing.T) {
	s := startServer(t)
	entry := &domain.Entry{Path: "vector06c/disks", Name: "Коллекция", Description: "Описание\tполе\r\n1Injected\tselector", Type: domain.EntryTypeDirectory}
	for i := 0; i < 105; i++ {
		name := fmt.Sprintf("disk%03d.rom", i)
		entry.Files = append(entry.Files, domain.File{Filename: name, Filepath: "vector06c/" + strings.Repeat("папка/", 30) + name})
	}
	last := entry.Files[104]
	full := filepath.Join(s.storage, filepath.FromSlash(last.Filepath))
	require.NoError(t, os.MkdirAll(filepath.Dir(full), 0o755))
	require.NoError(t, os.WriteFile(full, []byte("last original"), 0o644))
	saveEntries(t, s, entry)
	selector := "/entry?path=vector06c%2Fdisks"
	count := 0
	for page := 0; selector != "" && page < 5; page++ {
		response := s.request(t, selector)
		assertMenuLimits(t, response)
		items := menu(t, response)
		before := count
		for _, item := range items {
			require.NotEqual(t, "Injected", item.name, "metadata must never inject a menu item")
			if item.kind != '9' {
				continue
			}
			require.Equal(t, fmt.Sprintf("disk%03d.rom", count), item.name)
			count++
			if count == 105 {
				require.Equal(t, "last original", s.request(t, item.selector))
			}
		}
		selector = nextPage(t, items, before)
	}
	require.Equal(t, 105, count)
	require.Empty(t, selector)
}

func TestMetadataMenusAndCardRelationsPaginate(t *testing.T) {
	s := startServer(t)
	main := &domain.Entry{Path: "vector06c/collection", Name: "Метаданные", Type: domain.EntryTypeDirectory}
	var authors []*domain.Author
	entries := []*domain.Entry{main}
	for i := 0; i < 55; i++ {
		dir := fmt.Sprintf("author%03d", i) + strings.Repeat("д", 100)
		author := &domain.Author{DirectoryName: dir, Name: fmt.Sprintf("Автор %03d ", i) + strings.Repeat("Ж", 100)}
		authors = append(authors, author)
		main.Authors = append(main.Authors, *author)
		main.Tags = append(main.Tags, domain.Tag{Name: fmt.Sprintf("Тег %03d ", i) + strings.Repeat("Ж", 100)})
		entries = append(entries, &domain.Entry{Path: fmt.Sprintf("platform%03d", i) + strings.Repeat("д", 100), Name: fmt.Sprintf("Платформа %03d ", i) + strings.Repeat("Ж", 100), Type: domain.EntryTypeDirectory})
	}
	require.NoError(t, s.holder.Get().SaveEntriesAndAuthors(t.Context(), entries, authors))
	for _, section := range []struct{ selector, prefix string }{{"/platforms", "Платформа "}, {"/authors", "Автор "}, {"/tags", "Тег "}} {
		t.Run(section.selector, func(t *testing.T) {
			selector, count := section.selector, 0
			for page := 0; selector != "" && page < 4; page++ {
				response := s.request(t, selector)
				assertMenuLimits(t, response)
				items := menu(t, response)
				before := count
				for _, item := range items {
					if strings.HasPrefix(item.name, section.prefix) {
						count++
						menu(t, s.request(t, item.selector))
					}
				}
				selector = nextPage(t, items, before)
			}
			require.Equal(t, 55, count)
			require.Empty(t, selector)
		})
	}
	selector, count := "/entry?path=vector06c%2Fcollection", 0
	for page := 0; selector != "" && page < 4; page++ {
		response := s.request(t, selector)
		assertMenuLimits(t, response)
		items := menu(t, response)
		before := count
		for _, item := range items {
			if strings.HasPrefix(item.name, "Автор: ") || strings.HasPrefix(item.name, "Тег: ") {
				count++
			}
		}
		selector = nextPage(t, items, before)
	}
	require.Equal(t, 110, count, "card must expose all authors and tags")
	require.Empty(t, selector)
}

func TestHolderSwapUsesCurrentCatalogAndRejectsStaleLinks(t *testing.T) {
	s := startServer(t)
	saveEntries(t, s, &domain.Entry{Path: "old/game", Name: "Старая запись", Type: domain.EntryTypeDirectory,
		Files: []domain.File{{Filename: "old.rom", Filepath: "old.rom"}}})
	oldEntry := findItem(t, menu(t, s.request(t, "/entries")), "Старая запись")
	oldFile := findItem(t, menu(t, s.request(t, oldEntry.selector)), "old.rom")
	next, err := repository.NewSQLiteRepository(":memory:", zap.NewNop())
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, next.Close()) })
	require.NoError(t, next.SaveEntriesAndAuthors(t.Context(), []*domain.Entry{{Path: "new/game", Name: "Новая запись", Type: domain.EntryTypeDirectory,
		Files: []domain.File{{Filename: "new.rom", Filepath: "new.rom"}}}}, nil))
	require.NoError(t, os.WriteFile(filepath.Join(s.storage, "new.rom"), []byte("NEW ORIGINAL"), 0o644))
	s.holder.Swap(next, "next")
	current := menu(t, s.request(t, "/entries"))
	newEntry := findItem(t, current, "Новая запись")
	for _, item := range current {
		require.NotContains(t, item.name, "Старая запись")
	}
	require.Contains(t, s.request(t, newEntry.selector), "Новая запись")
	for _, stale := range []string{oldEntry.selector, oldFile.selector} {
		response := s.request(t, stale)
		require.NotContains(t, response, "NEW ORIGINAL")
		require.Equal(t, byte('3'), menu(t, response)[0].kind)
	}
}

func TestLongParentAndChildPathsKeepWorkingLinks(t *testing.T) {
	s := startServer(t)
	parent := "vector06c/" + strings.Repeat("длинная-папка/", 20) + "collection"
	saveEntries(t, s,
		&domain.Entry{Path: parent, Name: "Родитель", Type: domain.EntryTypeDirectory},
		&domain.Entry{Path: parent + "/child", Name: "Потомок", Type: domain.EntryTypeDirectory})
	items := menu(t, s.request(t, "/entries"))
	response := s.request(t, findItem(t, items, "Родитель").selector)
	assertMenuLimits(t, response)
	child := findItem(t, menu(t, response), "Потомок")
	response = s.request(t, child.selector)
	assertMenuLimits(t, response)
	require.Contains(t, response, "Потомок")
}

func TestInvalidPublicAddressIsRejected(t *testing.T) {
	repo, err := repository.NewSQLiteRepository(":memory:", zap.NewNop())
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, repo.Close()) })
	holder := repository.NewHolder(repo, ":memory:")
	for _, cfg := range []gopher.Config{
		{Host: "", Port: 70}, {Host: "bad\thost", Port: 70}, {Host: "bad\r\nhost", Port: 70},
		{Host: "host name", Port: 70}, {Host: "host", Port: 0}, {Host: "host", Port: -1}, {Host: "host", Port: 65536},
	} {
		_, err := gopher.New(cfg, holder, zap.NewNop())
		require.Error(t, err, "invalid public address must fail before serving: %+v", cfg)
	}
}

func TestConcurrentCloseIsSafe(t *testing.T) {
	s := startServer(t)
	menu(t, s.request(t, "/"))
	done := make(chan error, 8)
	for i := 0; i < cap(done); i++ {
		go func() { done <- s.server.Close() }()
	}
	for i := 0; i < cap(done); i++ {
		select {
		case err := <-done:
			require.NoError(t, err)
		case <-time.After(3 * time.Second):
			t.Fatal("concurrent Close did not finish")
		}
	}
}

func TestMalformedWireRequestsReturnErrors(t *testing.T) {
	s := startServer(t)
	for _, request := range []string{"/\n", strings.Repeat("x", 4096)} {
		items := menu(t, s.rawRequest(t, request))
		require.Equal(t, byte('3'), items[0].kind)
	}
}

func TestWideTitlesFit72DisplayColumns(t *testing.T) {
	s := startServer(t)
	saveEntries(t, s, &domain.Entry{Path: "wide/game", Name: strings.Repeat("界", 80), Type: domain.EntryTypeDirectory})
	items := menu(t, s.request(t, "/entries"))
	item := findItem(t, items, "界")
	// Each of these CJK ideographs takes two display columns.
	require.LessOrEqual(t, utf8.RuneCountInString(item.name), 36)
	require.Contains(t, s.request(t, item.selector), "界")
}
