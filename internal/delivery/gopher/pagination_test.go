package gopher_test

import (
	"fmt"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
	"warehouse06/internal/domain"
)

func textLink(t *testing.T, s *tcpServer, html string) menuItem {
	t.Helper()
	saveEntries(t, s, &domain.Entry{Path: "vector06c/journal", Name: "Журнал", Type: domain.EntryTypeDirectory, ContentHTML: html})
	return findItem(t, menu(t, s.request(t, "/entry?path=vector06c%2Fjournal")), "Полный текст")
}

func navigation(items []menuItem, name string) string {
	for _, item := range items {
		if item.kind == '1' && strings.Contains(item.name, name) {
			return item.selector
		}
	}
	return ""
}

func pageText(items []menuItem) string {
	var lines []string
	for _, item := range items {
		if item.kind == 'i' {
			lines = append(lines, item.name)
		}
	}
	return strings.Join(lines, "\n")
}

func TestFullTextPaginationFollowsLinksWithoutLosingText(t *testing.T) {
	s := startServer(t)
	var html strings.Builder
	var expected []string
	for i := 0; i < 1800; i++ {
		sentence := fmt.Sprintf("Запись %04d: проверяем полный текст.", i)
		expected = append(expected, sentence)
		fmt.Fprintf(&html, "<p>%s</p>", sentence)
	}
	link := textLink(t, s, html.String())
	require.Equal(t, byte('1'), link.kind)
	selector := link.selector
	var seen []string
	var previousSelector string
	pages := 0
	for ; selector != "" && pages < 30; pages++ {
		response := s.request(t, selector)
		assertMenuLimits(t, response)
		items := menu(t, response)
		previous := navigation(items, "Предыдущая")
		if pages == 0 {
			require.Empty(t, previous)
		} else {
			require.NotEmpty(t, previous)
			require.Equal(t, s.request(t, previousSelector), s.request(t, previous))
		}
		card := navigation(items, "карточк")
		require.NotEmpty(t, card)
		findItem(t, menu(t, s.request(t, card)), "Полный текст")
		for _, item := range items {
			if item.kind == 'i' && item.name != "" {
				seen = append(seen, item.name)
			}
		}
		previousSelector = selector
		selector = navigation(items, "Следующая")
	}
	require.Empty(t, selector, "pagination must terminate")
	require.Greater(t, pages, 2)
	require.Equal(t, expected, seen, "all paragraphs must remain ordered, complete and unique")
}

func TestFullTextPaginationCountsRenderedBytes(t *testing.T) {
	// 341 complete 70-byte lines plus CRLF consume 24552 bytes. The final
	// line determines the exact boundary, independently of HTML source size.
	for _, tc := range []struct {
		name string
		tail int
		kind byte
	}{
		{"below", 21, '0'}, {"exact", 22, '0'}, {"above", 23, '1'},
	} {
		t.Run(tc.name, func(t *testing.T) {
			s := startServer(t)
			html := strings.Repeat(strings.Repeat("я", 35)+"<br>", 341) + strings.Repeat("x", tc.tail)
			link := textLink(t, s, html)
			require.Equal(t, tc.kind, link.kind)
			response := s.request(t, link.selector)
			if tc.kind == '0' {
				require.Len(t, response, 341*72+tc.tail+2+3)
			} else {
				assertMenuLimits(t, response)
			}
		})
	}
	t.Run("large HTML small text", func(t *testing.T) {
		s := startServer(t)
		link := textLink(t, s, "<p title='"+strings.Repeat("attribute", 5000)+"'>Короткий текст.</p>")
		require.Equal(t, byte('0'), link.kind)
		require.Equal(t, "Короткий текст.\r\n.\r\n", s.request(t, link.selector))
	})
}

func TestFullTextPaginationSplitsOversizedParagraphAtSentence(t *testing.T) {
	s := startServer(t)
	// Each sentence fits on a page; together they exceed both target and wire
	// limits. The second sentence must start on the second page, intact.
	first := strings.Repeat("первое ", 1400) + "Конец первого."
	second := strings.Repeat("второе ", 1400) + "Конец второго."
	link := textLink(t, s, "<p>"+first+" "+second+"</p>")
	require.Equal(t, byte('1'), link.kind)
	one := menu(t, s.request(t, link.selector))
	require.Equal(t, strings.Fields(first), strings.Fields(pageText(one)))
	next := navigation(one, "Следующая")
	require.NotEmpty(t, next)
	response := s.request(t, next)
	assertMenuLimits(t, response)
	two := menu(t, response)
	require.Equal(t, strings.Fields(second), strings.Fields(pageText(two)))
	require.Empty(t, navigation(two, "Следующая"))
}

func TestFullTextPaginationPreservesSentenceAboveTarget(t *testing.T) {
	s := startServer(t)
	sentence := strings.Repeat("слово ", 2250) + "конец."
	link := textLink(t, s, "<p>"+sentence+"</p><p>Последний абзац.</p>")
	require.Equal(t, byte('1'), link.kind)
	response := s.request(t, link.selector)
	assertMenuLimits(t, response)
	one := menu(t, response)
	require.Equal(t, strings.Fields(sentence), strings.Fields(pageText(one)))
	next := navigation(one, "Следующая")
	require.NotEmpty(t, next)
	two := menu(t, s.request(t, next))
	require.Equal(t, "Последний абзац.", strings.TrimSpace(pageText(two)))
	require.Empty(t, navigation(two, "Следующая"))
}

func TestFullTextPaginationWrapsWideLongWordWithoutLoss(t *testing.T) {
	s := startServer(t)
	word := strings.Repeat("界Ж😀", 12000)
	link := textLink(t, s, "<p>"+word+"</p>")
	require.Equal(t, byte('1'), link.kind)
	selector := link.selector
	var actual strings.Builder
	pages := 0
	for ; selector != "" && pages < 30; pages++ {
		response := s.request(t, selector)
		assertMenuLimits(t, response)
		items := menu(t, response)
		text := strings.ReplaceAll(pageText(items), "\n", "")
		require.NotEmpty(t, text, "no empty final page")
		actual.WriteString(text)
		selector = navigation(items, "Следующая")
	}
	require.Empty(t, selector)
	require.Greater(t, pages, 2)
	require.Equal(t, word, actual.String())
}

func TestFullTextPaginationDoesNotCreateBlankPageAtLineLimit(t *testing.T) {
	s := startServer(t)
	first := strings.Repeat("x<br>", 795) + "x"
	second := strings.Repeat("слово ", 2250) + "конец."
	link := textLink(t, s, "<p>"+first+"</p><p>"+second+"</p>")
	require.Equal(t, byte('1'), link.kind)
	selector := link.selector
	var got []string
	pages := 0
	for ; selector != "" && pages < 10; pages++ {
		response := s.request(t, selector)
		assertMenuLimits(t, response)
		items := menu(t, response)
		content := pageText(items)
		require.NotEmpty(t, strings.TrimSpace(content), "every page must contain document text")
		got = append(got, strings.Fields(content)...)
		selector = navigation(items, "Следующая")
	}
	require.Empty(t, selector)
	require.Equal(t, append(strings.Fields(strings.Repeat("x ", 796)), strings.Fields(second)...), got)
}

func TestFullTextPaginationHardSplitsSentenceAtWords(t *testing.T) {
	s := startServer(t)
	var words []string
	for i := 0; i < 8000; i++ {
		words = append(words, fmt.Sprintf("слово%04d", i))
	}
	document := strings.Join(words, " ") + "."
	link := textLink(t, s, "<p>"+document+"</p>")
	selector := link.selector
	var got []string
	pages := 0
	for ; selector != "" && pages < 30; pages++ {
		response := s.request(t, selector)
		assertMenuLimits(t, response)
		items := menu(t, response)
		fields := strings.Fields(pageText(items))
		require.NotEmpty(t, fields)
		got = append(got, fields...)
		selector = navigation(items, "Следующая")
	}
	require.Empty(t, selector)
	require.Greater(t, pages, 2)
	require.Equal(t, strings.Fields(document), got, "hard limits must never split an ordinary word")
}

func TestFullTextPaginationRejectsInvalidPagesAndSupportsDirectSelector(t *testing.T) {
	s := startServer(t)
	link := textLink(t, s, "<p>"+strings.Repeat("слово ", 5000)+"конец.</p>")
	first := s.request(t, link.selector)
	require.Equal(t, first, s.request(t, "/text?path=vector06c%2Fjournal"))
	for _, suffix := range []string{"&page=-1", "&page=bad", "&page=", "&page=999999", "&page=9999999999999999999999999999", "&page=0&page=1"} {
		t.Run(suffix, func(t *testing.T) {
			items := menu(t, s.request(t, link.selector+suffix))
			require.NotEmpty(t, items)
			require.Equal(t, byte('3'), items[0].kind)
		})
	}
	missing := menu(t, s.request(t, "/text?path=missing&page=1"))
	require.Equal(t, byte('3'), missing[0].kind)
	require.Equal(t, first, s.request(t, link.selector), "invalid requests must not damage subsequent reads")
}

func TestFullTextPaginationMovesWholeParagraphToNextPage(t *testing.T) {
	s := startServer(t)
	first := strings.Repeat("Первый абзац. ", 650)
	second := strings.Repeat("Второй абзац. ", 650)
	link := textLink(t, s, "<p>"+first+"</p><p>"+second+"</p>")
	require.Equal(t, byte('1'), link.kind)
	response := s.request(t, link.selector)
	assertMenuLimits(t, response)
	one := menu(t, response)
	require.Equal(t, strings.Fields(first), strings.Fields(pageText(one)))
	next := navigation(one, "Следующая")
	require.NotEmpty(t, next)
	response = s.request(t, next)
	assertMenuLimits(t, response)
	two := menu(t, response)
	require.Equal(t, strings.Fields(second), strings.Fields(pageText(two)))
	require.Empty(t, navigation(two, "Следующая"))
}

func TestFullTextPaginationPreservesHTMLListsAndProtocolCharacters(t *testing.T) {
	s := startServer(t)
	var html strings.Builder
	var expected []string
	for i := 0; i < 600; i++ {
		fmt.Fprintf(&html, "<p>Абзац %04d &amp; текст.</p><ul><li>Один %04d</li><li>.точка %04d</li></ul>", i, i, i)
		expected = append(expected, fmt.Sprintf("Абзац %04d & текст.", i), fmt.Sprintf("- Один %04d", i), fmt.Sprintf("- .точка %04d", i))
	}
	link := textLink(t, s, html.String())
	selector := link.selector
	var got []string
	blank := false
	for page := 0; selector != "" && page < 30; page++ {
		response := s.request(t, selector)
		assertMenuLimits(t, response)
		require.NotContains(t, strings.ReplaceAll(response, "\r\n", ""), "\n")
		for _, line := range strings.Split(strings.TrimSuffix(response, "\r\n"), "\r\n") {
			require.LessOrEqual(t, len(line)+2, 255, "wire line includes CRLF")
		}
		items := menu(t, response)
		for _, item := range items {
			if item.kind == 'i' {
				if item.name == "" {
					blank = true
				} else {
					got = append(got, item.name)
				}
			}
		}
		selector = navigation(items, "Следующая")
	}
	require.Empty(t, selector)
	require.True(t, blank, "paragraph separation must survive HTML conversion")
	require.Equal(t, expected, got)
}
