// Package rebus generates a KOI8-R-encoded, dBASE II-compatible export of the
// catalog, targeting the "РЕБУС" DBMS historically run on Robotron PC-1715
// hardware.
//
// dBASE II (version byte 0x02), not dBASE III PLUS: a sample table REBUS
// accepts is kept at testdata/sample_dbase2.dbf and the writer in dbase2.go
// is verified byte for byte against it. The two formats share almost nothing
// beyond the .DBF extension - see dbase2.go for the layout.
package rebus

import (
	"archive/zip"
	"bytes"
	"fmt"
	"sort"

	"warehouse06/internal/domain"
)

type ExportInput struct {
	Entries []*domain.Entry
	Authors []*domain.Author
	Tags    []*domain.Tag
}

// Export builds the full set of REBUS-compatible tables from input and
// returns them bundled as a zip archive, plus any non-fatal warnings
// (truncated fields, omitted columns, unmappable KOI8-R characters). It is
// pure: all table construction happens against in-memory buffers, never real
// files.
//
// Column widths are not fixed: every Character and Numeric column is sized to
// the longest value actually present, and a Character column with no value in
// any row is left out of the table entirely (see tableDef). The schema
// therefore varies between exports as the catalog changes.
func Export(input ExportInput) ([]byte, []string, error) {
	var warnings []string
	warn := func(msg string) { warnings = append(warnings, msg) }
	conv := koi8Converter{onUnmappable: func(r rune) {
		warn(fmt.Sprintf("unmappable character %q replaced with '?' during KOI8-R transcoding", r))
	}}

	steps := []struct {
		name string
		fn   func() (map[string][]byte, error)
	}{
		{"ENTRIES.DBF", func() (map[string][]byte, error) { return buildEntries(conv, warn, input.Entries) }},
		{"AUTHORS.DBF", func() (map[string][]byte, error) { return buildAuthors(conv, warn, input.Authors) }},
		{"TAGS.DBF", func() (map[string][]byte, error) { return buildTags(conv, warn, input.Tags) }},
		{"FILES.DBF", func() (map[string][]byte, error) { return buildFiles(conv, warn, input.Entries) }},
		{"ENTRAUTH.DBF", func() (map[string][]byte, error) { return buildEntrAuth(conv, warn, input.Entries) }},
		{"ENTRTAGS.DBF", func() (map[string][]byte, error) { return buildEntrTags(conv, warn, input.Entries) }},
		{"ENTRREQ.DBF", func() (map[string][]byte, error) { return buildEntrReq(conv, warn, input.Entries) }},
	}

	files := make(map[string][]byte)
	for _, s := range steps {
		built, err := s.fn()
		if err != nil {
			return nil, nil, fmt.Errorf("rebus: build %s: %w", s.name, err)
		}
		for name, data := range built {
			files[name] = data
		}
	}

	zipData, err := zipFiles(files)
	if err != nil {
		return nil, nil, fmt.Errorf("rebus: zip export: %w", err)
	}
	return zipData, warnings, nil
}

func zipFiles(files map[string][]byte) ([]byte, error) {
	names := make([]string, 0, len(files))
	for name := range files {
		names = append(names, name)
	}
	sort.Strings(names) // deterministic archive ordering

	var buf bytes.Buffer
	zw := zip.NewWriter(&buf)
	for _, name := range names {
		fw, err := zw.Create(name)
		if err != nil {
			return nil, fmt.Errorf("add %s to zip: %w", name, err)
		}
		if _, err := fw.Write(files[name]); err != nil {
			return nil, fmt.Errorf("write %s to zip: %w", name, err)
		}
	}
	if err := zw.Close(); err != nil {
		return nil, fmt.Errorf("close zip: %w", err)
	}
	return buf.Bytes(), nil
}
