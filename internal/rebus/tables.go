package rebus

import (
	"fmt"
	"html"
	"strconv"
	"time"

	"github.com/microcosm-cc/bluemonday"

	"warehouse06/internal/domain"
)

// Column names are dBASE II identifiers: at most 10 uppercase letters and
// digits, no underscore (that is a dBASE III addition) - see
// validateFieldName. Every table also stays inside the format's 32-field and
// 1000-byte-record limits; tableDef shrinks Character columns if the data
// would push a record over the latter.
//
// dBASE II has no date and no memo type, so timestamps are written as
// Character "YYYYMMDD" and long text (content_html) is truncated into a plain
// Character field - see buildEntries and buildAuthors.

// truncate cuts s to at most charMax runes, reporting whether it did. Every
// rune koi8Converter.Encode produces is exactly one byte, so a rune-count
// limit here exactly matches the target Character field's byte length.
func truncate(s string, warn func(string), warnMsg string) string {
	r := []rune(s)
	if len(r) <= charMax {
		return s
	}
	warn(warnMsg)
	return string(r[:charMax])
}

// plainText strips HTML markup down to plain text for the CONTENT field - a
// text-mode 1980s DBMS record has no meaningful way to represent markup.
// The result is truncated to a plain Character field's capacity (dBASE II has
// no memo file), which is a real loss of the full README/content text, logged
// as a warning.
func plainText(h string, warn func(string), warnMsg string) string {
	return truncate(html.UnescapeString(bluemonday.StrictPolicy().Sanitize(h)), warn, warnMsg)
}

// num renders a Numeric column value.
func num(v int64) string { return strconv.FormatInt(v, 10) }

// logical renders a Logical column value in dBASE II's T/F form.
func logical(v bool) string {
	if v {
		return "T"
	}
	return "F"
}

// date renders a timestamp for a Character column, dBASE II having no date
// type. A zero time becomes an empty value, which lets the column be dropped
// entirely if no row has a timestamp.
func date(t time.Time) string {
	if t.IsZero() {
		return ""
	}
	return t.Format("20060102")
}

func buildEntries(conv koi8Converter, warn func(string), entries []*domain.Entry) (map[string][]byte, error) {
	t := &tableDef{
		name: "ENTRIES.DBF",
		cols: []colDef{
			{"ID", typeNumeric},
			{"PATH", typeChar},
			{"NAME", typeChar},
			{"PLATFORM", typeChar},
			{"TYPE", typeChar},
			{"DESCR", typeChar},
			{"CONTENT", typeChar},
			{"ENTRYDATE", typeChar},
			{"YOUTUBE", typeChar},
			{"CREATED", typeChar},
			{"UPDATED", typeChar},
		},
	}

	for _, e := range entries {
		t.addRow(
			num(e.ID),
			truncate(e.Path, warn, fmt.Sprintf("entries.PATH truncated for entry id=%d", e.ID)),
			truncate(e.Name, warn, fmt.Sprintf("entries.NAME truncated for entry id=%d", e.ID)),
			truncate(e.Platform, warn, fmt.Sprintf("entries.PLATFORM truncated for entry id=%d", e.ID)),
			truncate(string(e.Type), warn, fmt.Sprintf("entries.TYPE truncated for entry id=%d", e.ID)),
			truncate(e.Description, warn, fmt.Sprintf("entries.DESCR truncated for entry id=%d", e.ID)),
			plainText(e.ContentHTML, warn, fmt.Sprintf("entries.CONTENT truncated for entry id=%d", e.ID)),
			truncate(e.Date, warn, fmt.Sprintf("entries.ENTRYDATE truncated for entry id=%d", e.ID)),
			truncate(e.Youtube, warn, fmt.Sprintf("entries.YOUTUBE truncated for entry id=%d", e.ID)),
			date(e.CreatedAt),
			date(e.UpdatedAt),
		)
	}
	return t.build(conv, warn)
}

func buildAuthors(conv koi8Converter, warn func(string), authors []*domain.Author) (map[string][]byte, error) {
	t := &tableDef{
		name: "AUTHORS.DBF",
		cols: []colDef{
			{"ID", typeNumeric},
			{"DIRNAME", typeChar},
			{"NAME", typeChar},
			{"ADDRESS", typeChar},
			{"CONTENT", typeChar},
		},
	}

	for _, a := range authors {
		t.addRow(
			num(a.ID),
			truncate(a.DirectoryName, warn, fmt.Sprintf("authors.DIRNAME truncated for author id=%d", a.ID)),
			truncate(a.Name, warn, fmt.Sprintf("authors.NAME truncated for author id=%d", a.ID)),
			truncate(a.Address, warn, fmt.Sprintf("authors.ADDRESS truncated for author id=%d", a.ID)),
			plainText(a.ContentHTML, warn, fmt.Sprintf("authors.CONTENT truncated for author id=%d", a.ID)),
		)
	}
	return t.build(conv, warn)
}

func buildTags(conv koi8Converter, warn func(string), tags []*domain.Tag) (map[string][]byte, error) {
	t := &tableDef{
		name: "TAGS.DBF",
		cols: []colDef{
			{"ID", typeNumeric},
			{"NAME", typeChar},
		},
	}

	for _, tag := range tags {
		t.addRow(
			num(tag.ID),
			truncate(tag.Name, warn, fmt.Sprintf("tags.NAME truncated for tag id=%d", tag.ID)),
		)
	}
	return t.build(conv, warn)
}

func buildFiles(conv koi8Converter, warn func(string), entries []*domain.Entry) (map[string][]byte, error) {
	t := &tableDef{
		name: "FILES.DBF",
		cols: []colDef{
			{"ID", typeNumeric},
			{"ENTRYID", typeNumeric},
			{"FILENAME", typeChar},
			{"FILEPATH", typeChar},
			{"ISIMAGE", typeLogical},
			{"FILESIZE", typeNumeric},
			{"SHA256", typeChar},
		},
	}

	for _, e := range entries {
		for _, f := range e.Files {
			t.addRow(
				num(f.ID),
				num(f.EntryID),
				truncate(f.Filename, warn, fmt.Sprintf("files.FILENAME truncated for file id=%d", f.ID)),
				truncate(f.Filepath, warn, fmt.Sprintf("files.FILEPATH truncated for file id=%d", f.ID)),
				logical(f.IsImage),
				num(f.Size),
				truncate(f.SHA256, warn, fmt.Sprintf("files.SHA256 truncated for file id=%d", f.ID)),
			)
		}
	}
	return t.build(conv, warn)
}

func buildEntrAuth(conv koi8Converter, warn func(string), entries []*domain.Entry) (map[string][]byte, error) {
	t := &tableDef{
		name: "ENTRAUTH.DBF",
		cols: []colDef{
			{"ENTRYID", typeNumeric},
			{"AUTHORID", typeNumeric},
		},
	}

	for _, e := range entries {
		for _, a := range e.Authors {
			t.addRow(num(e.ID), num(a.ID))
		}
	}
	return t.build(conv, warn)
}

func buildEntrTags(conv koi8Converter, warn func(string), entries []*domain.Entry) (map[string][]byte, error) {
	t := &tableDef{
		name: "ENTRTAGS.DBF",
		cols: []colDef{
			{"ENTRYID", typeNumeric},
			{"TAGID", typeNumeric},
		},
	}

	for _, e := range entries {
		for _, tag := range e.Tags {
			t.addRow(num(e.ID), num(tag.ID))
		}
	}
	return t.build(conv, warn)
}

func buildEntrReq(conv koi8Converter, warn func(string), entries []*domain.Entry) (map[string][]byte, error) {
	t := &tableDef{
		name: "ENTRREQ.DBF",
		cols: []colDef{
			{"ENTRYID", typeNumeric},
			{"REQPATH", typeChar},
		},
	}

	for _, e := range entries {
		for _, reqPath := range e.Requires {
			t.addRow(
				num(e.ID),
				truncate(reqPath, warn, fmt.Sprintf("entrreq.REQPATH truncated for entry id=%d", e.ID)),
			)
		}
	}
	return t.build(conv, warn)
}
