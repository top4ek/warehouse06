package rebus

import (
	"archive/zip"
	"bytes"
	"encoding/binary"
	"io"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"golang.org/x/text/encoding/charmap"

	"warehouse06/internal/domain"
)

func testInput() ExportInput {
	longPath := "vector06c/" + strings.Repeat("x", 260)
	return ExportInput{
		Entries: []*domain.Entry{
			{
				ID:          1,
				Path:        "vector06c/exolon",
				Name:        "Эксолон",
				Platform:    "vector06c",
				Type:        domain.EntryTypeDirectory,
				Description: "Шутер для Вектор-06Ц",
				ContentHTML: "<p>Полное описание <b>игры</b> с историей.</p>",
				Date:        "1988",
				CreatedAt:   time.Date(2026, 1, 2, 0, 0, 0, 0, time.UTC),
				UpdatedAt:   time.Date(2026, 1, 3, 0, 0, 0, 0, time.UTC),
				Tags:        []domain.Tag{{ID: 5, Name: "shooter"}},
				Authors:     []domain.Author{{ID: 7, DirectoryName: "author7"}},
				Files: []domain.File{
					{ID: 9, EntryID: 1, Filename: "cover.png", Filepath: "vector06c/exolon/cover.png", IsImage: true, Size: 1024, SHA256: strings.Repeat("a", 64)},
				},
				Requires: []string{"vector06c/microdos"},
			},
			{
				// Overlong path forces truncation, and name has a rune with
				// no KOI8-R mapping to force the unmappable fallback.
				ID:          2,
				Path:        longPath,
				Name:        "ok\U0001F600ok",
				Platform:    "vector06c",
				Type:        domain.EntryTypeFile,
				Description: "second",
				ContentHTML: "plain",
				CreatedAt:   time.Date(2026, 1, 4, 0, 0, 0, 0, time.UTC),
				UpdatedAt:   time.Date(2026, 1, 4, 0, 0, 0, 0, time.UTC),
			},
		},
		Authors: []*domain.Author{
			{ID: 7, DirectoryName: "author7", Name: "Автор Семёнов", Address: "Москва", ContentHTML: "<p>Биография</p>"},
		},
		Tags: []*domain.Tag{
			{ID: 5, Name: "shooter"},
		},
	}
}

func TestExport_ProducesExpectedFileSet(t *testing.T) {
	zipData, warnings, err := Export(testInput())
	require.NoError(t, err)

	files := unzip(t, zipData)
	assert.ElementsMatch(t, []string{
		"ENTRIES.DBF",
		"AUTHORS.DBF",
		"TAGS.DBF",
		"FILES.DBF",
		"ENTRAUTH.DBF",
		"ENTRTAGS.DBF",
		"ENTRREQ.DBF",
	}, keys(files))

	assert.NotEmpty(t, warnings, "expected truncation and unmappable-rune warnings")
	joined := strings.Join(warnings, "\n")
	assert.Contains(t, joined, "PATH truncated")
	assert.Contains(t, joined, "unmappable character")
}

// TestExport_EntriesFieldsMatchDBaseIISpec hand-parses the raw ENTRIES.DBF
// bytes against the dBASE II header, field-descriptor and record layout that
// testdata/sample_dbase2.dbf pins down (see TestWriteTable_MatchesSampleFile):
// a fixed 521-byte header, 16-byte field descriptors, a 16-bit record count
// and a trailing 0x1A EOF marker.
func TestExport_EntriesFieldsMatchDBaseIISpec(t *testing.T) {
	zipData, _, err := Export(testInput())
	require.NoError(t, err)
	files := unzip(t, zipData)
	data := files["ENTRIES.DBF"]
	require.NotEmpty(t, data)

	// --- header ---
	require.Greater(t, len(data), dbase2HeaderLen)
	assert.Equal(t, byte(0x02), data[0], "version byte: dBASE II")
	recordCount := binary.LittleEndian.Uint16(data[1:3])
	assert.EqualValues(t, 2, recordCount)
	assert.Equal(t, []byte{0, 0, 0}, data[3:6], "last-update date is left at zero, as in the sample")
	recordLen := binary.LittleEndian.Uint16(data[6:8])

	// 10 fields: ID,PATH,NAME,PLATFORM,TYPE,DESCR,CONTENT,ENTRYDATE,CREATED,
	// UPDATED. YOUTUBE is absent: no seeded entry has one, so the column is
	// never created. Every width below is the longest value in testInput():
	// ids 1 and 2 are one digit, "vector06c" is 9 characters, the overlong
	// path was cut to the 254-byte Character maximum, and so on. CREATED and
	// UPDATED are Character "YYYYMMDD" because dBASE II has no date type.
	wantFields := []struct {
		name   string
		typ    byte
		length byte
	}{
		{"ID", 'N', 1}, {"PATH", 'C', 254}, {"NAME", 'C', 7}, {"PLATFORM", 'C', 9},
		{"TYPE", 'C', 9}, {"DESCR", 'C', 20}, {"CONTENT", 'C', 32}, {"ENTRYDATE", 'C', 4},
		{"CREATED", 'C', 8}, {"UPDATED", 'C', 8},
	}

	// --- field descriptors ---
	fields := parseFieldDescriptors(t, data)
	require.Len(t, fields, len(wantFields))
	wantRecordLen := 1 // delete flag
	for i, wf := range wantFields {
		assert.Equal(t, wf.name, fields[i].name, "field %d name", i)
		assert.Equal(t, wf.typ, fields[i].typ, "field %d type", i)
		assert.Equal(t, wf.length, fields[i].length, "field %d length", i)
		wantRecordLen += int(wf.length)
	}
	assert.EqualValues(t, wantRecordLen, recordLen)
	// Field descriptor array is terminated by 0x0D.
	assert.Equal(t, byte(0x0D), data[8+16*len(wantFields)])

	// --- records ---
	recStart := dbase2HeaderLen
	require.Equal(t, recStart+int(recordCount)*int(recordLen)+1, len(data), "records plus the EOF marker")
	assert.Equal(t, byte(0x1A), data[len(data)-1], "EOF marker")
	rec0 := data[recStart : recStart+int(recordLen)]
	assert.Equal(t, byte(0x20), rec0[0], "delete flag: active")

	// Walk field offsets within the record to isolate PATH, NAME and CONTENT.
	off := 1
	var pathBytes, nameBytes, contentBytes []byte
	for _, wf := range wantFields {
		val := rec0[off : off+int(wf.length)]
		switch wf.name {
		case "PATH":
			pathBytes = val
		case "NAME":
			nameBytes = val
		case "CONTENT":
			contentBytes = val
		}
		off += int(wf.length)
	}
	require.NotNil(t, pathBytes)
	require.NotNil(t, nameBytes)
	require.NotNil(t, contentBytes)

	pathDecoded, err := charmap.KOI8R.NewDecoder().Bytes(pathBytes)
	require.NoError(t, err)
	assert.Equal(t, "vector06c/exolon", strings.TrimRight(string(pathDecoded), " "))

	nameDecoded, err := charmap.KOI8R.NewDecoder().Bytes(nameBytes)
	require.NoError(t, err)
	assert.Equal(t, "Эксолон", strings.TrimRight(string(nameDecoded), " "))

	contentDecoded, err := charmap.KOI8R.NewDecoder().Bytes(contentBytes)
	require.NoError(t, err)
	got := strings.TrimRight(string(contentDecoded), " ")
	assert.Contains(t, got, "Полное описание")
	assert.Contains(t, got, "игры")
	assert.NotContains(t, got, "<p>", "HTML tags must have been stripped")
	assert.NotContains(t, got, "<b>", "HTML tags must have been stripped")
}

// TestExport_TruncatesOverlongPath verifies the second entry's overlong PATH
// was cut to the field's 254-byte capacity rather than erroring the export.
func TestExport_TruncatesOverlongPath(t *testing.T) {
	zipData, _, err := Export(testInput())
	require.NoError(t, err)
	files := unzip(t, zipData)
	data := files["ENTRIES.DBF"]

	recordLen := int(binary.LittleEndian.Uint16(data[6:8]))
	fields := parseFieldDescriptors(t, data)

	rec1 := data[dbase2HeaderLen+recordLen : dbase2HeaderLen+2*recordLen]
	off := 1
	for _, f := range fields {
		if f.name == "PATH" {
			assert.Len(t, strings.TrimRight(string(rec1[off:off+int(f.length)]), " "), 254)
			break
		}
		off += int(f.length)
	}
}

// TestExport_OmitsCharacterColumnsWithNoValues covers the schema-shrinking
// rule: a text column nobody fills is not written at all, rather than costing
// every fixed-length record its full width.
func TestExport_OmitsCharacterColumnsWithNoValues(t *testing.T) {
	input := ExportInput{
		Entries: []*domain.Entry{{
			ID: 1, Path: "vector06c/x", Name: "X", Platform: "vector06c",
			Type: domain.EntryTypeDirectory,
		}},
		Authors: []*domain.Author{{ID: 7, DirectoryName: "author7", Name: "Автор"}},
	}

	zipData, warnings, err := Export(input)
	require.NoError(t, err)
	files := unzip(t, zipData)

	entryFields := fieldNames(parseFieldDescriptors(t, files["ENTRIES.DBF"]))
	assert.NotContains(t, entryFields, "YOUTUBE", "no entry has a youtube link")
	assert.NotContains(t, entryFields, "DESCR", "no entry has a description")
	assert.NotContains(t, entryFields, "CONTENT", "no entry has content")
	assert.Contains(t, entryFields, "PATH", "PATH is filled and must survive")

	authorFields := fieldNames(parseFieldDescriptors(t, files["AUTHORS.DBF"]))
	assert.NotContains(t, authorFields, "CONTENT", "no author has content")
	assert.NotContains(t, authorFields, "ADDRESS", "no author has an address")
	assert.Contains(t, authorFields, "NAME")

	joined := strings.Join(warnings, "\n")
	assert.Contains(t, joined, "AUTHORS.DBF column CONTENT omitted")
}

// TestExport_SizesColumnsToLongestValue pins the width of a text column to the
// single longest value across all rows, not to a fixed 254.
func TestExport_SizesColumnsToLongestValue(t *testing.T) {
	input := ExportInput{
		Entries: []*domain.Entry{
			{ID: 1, Path: "a", Name: "short", Type: domain.EntryTypeDirectory},
			{ID: 2, Path: "b", Name: "Длинное имя записи", Type: domain.EntryTypeDirectory},
		},
	}

	zipData, _, err := Export(input)
	require.NoError(t, err)
	fields := parseFieldDescriptors(t, unzip(t, zipData)["ENTRIES.DBF"])

	name := fieldByName(t, fields, "NAME")
	assert.EqualValues(t, len([]rune("Длинное имя записи")), name.length)
	path := fieldByName(t, fields, "PATH")
	assert.EqualValues(t, 1, path.length)
}

// TestExport_SizesNumericColumnsToLargestValue does the same for Numeric
// columns: an ID field is as wide as the biggest id needs, no wider.
func TestExport_SizesNumericColumnsToLargestValue(t *testing.T) {
	input := ExportInput{
		Entries: []*domain.Entry{
			{ID: 4, Path: "a", Name: "a", Type: domain.EntryTypeDirectory},
			{ID: 123456, Path: "b", Name: "b", Type: domain.EntryTypeDirectory},
		},
	}

	zipData, _, err := Export(input)
	require.NoError(t, err)
	data := unzip(t, zipData)["ENTRIES.DBF"]
	fields := parseFieldDescriptors(t, data)
	assert.EqualValues(t, 6, fieldByName(t, fields, "ID").length)

	// The narrowed field must still read back as the original value.
	recordLen := int(binary.LittleEndian.Uint16(data[6:8]))
	rec1 := data[dbase2HeaderLen+recordLen : dbase2HeaderLen+2*recordLen]
	assert.Equal(t, "123456", strings.TrimSpace(string(rec1[1:1+fieldByName(t, fields, "ID").length])))
}

// TestExport_ShrinksColumnsToFitRecordLimit covers the dBASE II record
// budget: four maximum-width text columns would need 1017 bytes, so the
// widest ones are narrowed until the record fits 1000 bytes, and the data is
// cut to the narrowed widths.
func TestExport_ShrinksColumnsToFitRecordLimit(t *testing.T) {
	long := strings.Repeat("щ", 300) // 300 runes, 300 KOI8-R bytes
	input := ExportInput{
		Entries: []*domain.Entry{{
			ID:          1,
			Path:        long,
			Name:        long,
			Description: long,
			ContentHTML: long,
			Platform:    "vector06c",
			Type:        domain.EntryTypeDirectory,
		}},
	}

	zipData, warnings, err := Export(input)
	require.NoError(t, err)
	data := unzip(t, zipData)["ENTRIES.DBF"]

	recordLen := int(binary.LittleEndian.Uint16(data[6:8]))
	assert.LessOrEqual(t, recordLen, dbase2MaxRecordLen)
	assert.Equal(t, dbase2HeaderLen+recordLen+1, len(data))

	fields := parseFieldDescriptors(t, data)
	total := 1
	for _, f := range fields {
		assert.LessOrEqual(t, int(f.length), charMax)
		total += int(f.length)
	}
	assert.Equal(t, recordLen, total)

	joined := strings.Join(warnings, "\n")
	assert.Contains(t, joined, "shrunk from 254 to")
	assert.Contains(t, joined, "1000-byte record limit")

	// The record still holds each column's cut value, not garbage.
	rec0 := data[dbase2HeaderLen : dbase2HeaderLen+recordLen]
	off := 1
	for _, f := range fields {
		value := strings.TrimRight(string(rec0[off:off+int(f.length)]), " ")
		if f.name == "PATH" || f.name == "NAME" || f.name == "DESCR" || f.name == "CONTENT" {
			decoded, err := charmap.KOI8R.NewDecoder().Bytes([]byte(value))
			require.NoError(t, err)
			assert.Equal(t, strings.Repeat("щ", int(f.length)), string(decoded), "field %s", f.name)
		}
		off += int(f.length)
	}
}

// TestExport_EmptyTableKeepsOnlyNonCharacterColumns documents the corner the
// omission rule reaches when a table has no rows at all.
func TestExport_EmptyTableKeepsOnlyNonCharacterColumns(t *testing.T) {
	zipData, _, err := Export(ExportInput{})
	require.NoError(t, err)
	files := unzip(t, zipData)

	assert.Equal(t, []string{"ID"}, fieldNames(parseFieldDescriptors(t, files["TAGS.DBF"])))
	assert.Equal(t, []string{"ENTRYID"}, fieldNames(parseFieldDescriptors(t, files["ENTRREQ.DBF"])))
}

type fieldDesc struct {
	name   string
	typ    byte
	length byte
}

func parseFieldDescriptors(t *testing.T, data []byte) []fieldDesc {
	t.Helper()
	var fields []fieldDesc
	off := 8
	for {
		require.Less(t, off, dbase2HeaderLen)
		if data[off] == 0x0D {
			break
		}
		name := strings.TrimRight(string(data[off:off+11]), "\x00")
		fields = append(fields, fieldDesc{
			name:   name,
			typ:    data[off+11],
			length: data[off+12],
		})
		off += 16
	}
	return fields
}

func fieldNames(fields []fieldDesc) []string {
	out := make([]string, 0, len(fields))
	for _, f := range fields {
		out = append(out, f.name)
	}
	return out
}

func fieldByName(t *testing.T, fields []fieldDesc, name string) fieldDesc {
	t.Helper()
	for _, f := range fields {
		if f.name == name {
			return f
		}
	}
	t.Fatalf("field %s not found in %v", name, fieldNames(fields))
	return fieldDesc{}
}

func unzip(t *testing.T, data []byte) map[string][]byte {
	t.Helper()
	zr, err := zip.NewReader(bytes.NewReader(data), int64(len(data)))
	require.NoError(t, err)

	out := make(map[string][]byte, len(zr.File))
	for _, f := range zr.File {
		rc, err := f.Open()
		require.NoError(t, err)
		b, err := io.ReadAll(rc)
		require.NoError(t, err)
		require.NoError(t, rc.Close())
		out[f.Name] = b
	}
	return out
}

func keys(m map[string][]byte) []string {
	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	return out
}
