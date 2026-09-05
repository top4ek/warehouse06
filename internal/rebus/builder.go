package rebus

import (
	"fmt"
)

// charMax is the widest Character column dBASE II allows. Since column widths
// are derived from the data, this is the only cap applied before the record
// budget: text longer than this cannot be represented at all and is truncated
// with a warning.
const charMax = dbase2MaxCharLen

// colDef is a column whose width is not yet known - it is computed from the
// values collected in tableDef.rows.
type colDef struct {
	name string
	typ  fieldType
}

// tableDef collects one table's values before any column is sized: rows[i][j]
// holds row i's value for cols[j], already rendered to the string form that
// goes into the record (see num, logical and date in tables.go). dBASE II
// records are fixed-length, so every row pays each column's declared width in
// full - collecting the data first lets each column be sized to the longest
// value actually present, and lets a Character column with no values anywhere
// be dropped from the schema entirely.
//
// Character values must already be truncated to charMax by the caller (which
// is also where a truncation warning belongs, since only the caller knows the
// row's identity).
type tableDef struct {
	name string
	cols []colDef
	rows [][]string
}

func (t *tableDef) addRow(values ...string) {
	t.rows = append(t.rows, values)
}

// build sizes every column from the collected data, shrinks the schema to fit
// dBASE II's record limit and writes the table. Character columns with no
// value in any row are omitted entirely. conv transcodes Character values to
// KOI8-R; Numeric and Logical values are ASCII already.
func (t *tableDef) build(conv koi8Converter, warn func(string)) (map[string][]byte, error) {
	kept := make([]int, 0, len(t.cols))
	fields := make([]field, 0, len(t.cols))
	for i, c := range t.cols {
		length, keep := t.width(i, c.typ)
		if !keep {
			warn(fmt.Sprintf("%s column %s omitted: no row has a value", t.name, c.name))
			continue
		}
		kept = append(kept, i)
		fields = append(fields, field{name: c.name, typ: c.typ, length: length})
	}
	if err := t.shrinkToRecordLimit(fields, warn); err != nil {
		return nil, err
	}

	rows := make([][]string, 0, len(t.rows))
	for i, values := range t.rows {
		row := make([]string, 0, len(kept))
		for pos, srcIdx := range kept {
			value := values[srcIdx]
			if fields[pos].typ == typeChar {
				encoded, err := conv.Encode([]byte(value))
				if err != nil {
					return nil, fmt.Errorf("encode %s.%s on row %d: %w", t.name, t.cols[srcIdx].name, i, err)
				}
				value = string(encoded)
			}
			row = append(row, value)
		}
		rows = append(rows, row)
	}

	data, err := writeTable(fields, rows)
	if err != nil {
		return nil, fmt.Errorf("write %s: %w", t.name, err)
	}
	return map[string][]byte{t.name: data}, nil
}

// width returns the length to declare for column idx, and whether the column
// should exist at all. Numeric columns get at least one character so an empty
// table still has a valid schema; Logical is fixed at one character by the
// format.
func (t *tableDef) width(idx int, typ fieldType) (uint8, bool) {
	if typ == typeLogical {
		return 1, true
	}
	longest := 0
	for _, values := range t.rows {
		// Every rune koi8Converter.Encode produces is exactly one byte, so a
		// rune count is the encoded byte length.
		if n := len([]rune(values[idx])); n > longest {
			longest = n
		}
	}
	if typ == typeChar {
		if longest == 0 {
			return 0, false
		}
		return uint8(longest), true
	}
	if longest == 0 {
		longest = 1
	}
	return uint8(longest), true
}

// shrinkToRecordLimit narrows Character columns in place until the record fits
// dbase2MaxRecordLen, always taking from the widest column first so the cut is
// spread over the columns that can afford it. Values are cut to the final
// width when the record is written.
func (t *tableDef) shrinkToRecordLimit(fields []field, warn func(string)) error {
	original := make([]uint8, len(fields))
	for i, f := range fields {
		original[i] = f.length
	}

	for total := recordLength(fields); total > dbase2MaxRecordLen; total = recordLength(fields) {
		widest, second := -1, 1
		for i, f := range fields {
			if f.typ != typeChar {
				continue
			}
			if widest == -1 || f.length > fields[widest].length {
				widest = i
			}
		}
		if widest == -1 || fields[widest].length <= 1 {
			return fmt.Errorf("%s: record length %d cannot be shrunk below the dBASE II limit of %d", t.name, total, dbase2MaxRecordLen)
		}
		for i, f := range fields {
			if i != widest && f.typ == typeChar && int(f.length) > second {
				second = int(f.length)
			}
		}

		// Cut to whichever is larger: the width that just fits, or the
		// next-widest column - past that point the next round takes from
		// that column instead.
		target := int(fields[widest].length) - (total - dbase2MaxRecordLen)
		if target < second {
			target = second
		}
		// Ties between equally wide columns still have to make progress.
		if target >= int(fields[widest].length) {
			target = int(fields[widest].length) - 1
		}
		if target < 1 {
			target = 1
		}
		fields[widest].length = uint8(target)
	}

	for i, f := range fields {
		if f.length < original[i] {
			warn(fmt.Sprintf("%s column %s shrunk from %d to %d characters to fit the dBASE II %d-byte record limit",
				t.name, f.name, original[i], f.length, dbase2MaxRecordLen))
		}
	}
	return nil
}
