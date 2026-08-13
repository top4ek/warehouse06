package rebus

import (
	"encoding/binary"
	"fmt"
)

// dBASE II (version byte 0x02) table writer.
//
// The layout below is taken from the classic dBASE II specification and
// verified byte for byte against testdata/sample_dbase2.dbf, a file the
// target REBUS installation accepts (see TestWriteTable_MatchesSampleFile):
//
//	offset 0      1 byte    version, 0x02
//	offset 1      2 bytes   record count, little endian
//	offset 3      3 bytes   date of last update, month / day / year
//	offset 6      2 bytes   record length, little endian (delete flag included)
//	offset 8    512 bytes   up to 32 field descriptors, 16 bytes each,
//	                        followed by a 0x0D terminator and zero padding
//	offset 521            first record
//
// Each field descriptor is:
//
//	offset 0     11 bytes   name, ASCII, NUL padded
//	offset 11     1 byte    type: 'C', 'N' or 'L' - dBASE II has no date or
//	                        memo type
//	offset 12     1 byte    length
//	offset 13     2 bytes   field data address, little endian
//	offset 15     1 byte    decimal count
//
// A record is the delete flag (0x20 live, 0x2A deleted) followed by the
// fields, each padded to its full declared length. The file ends with a
// 0x1A EOF marker.
//
// This is a different format from dBASE III PLUS (0x03), not a variant of
// it: III has a variable-length header, 32-byte field descriptors with
// length and address swapped, and a 32-bit record count. Writing III output
// is what made the previous export unreadable in REBUS.
const (
	dbase2Version      = 0x02
	dbase2HeaderLen    = 521
	dbase2FieldDescLen = 16
	dbase2MaxFields    = 32
	dbase2MaxNameLen   = 10
	dbase2MaxCharLen   = 254
	dbase2MaxRecords   = 65535

	// dbase2MaxRecordLen is dBASE II's practical record-size limit. The
	// header stores the record length in 16 bits, but a 1980s reader is not
	// expected to handle more than this - tableDef shrinks its widest
	// Character columns to stay under it.
	dbase2MaxRecordLen = 1000

	recordLiveFlag = 0x20
	eofMarker      = 0x1A

	// fieldAddrBase is where the sample file's first field data address
	// starts; every following address is the previous one plus the previous
	// field's length. These are dBASE II's own in-memory record-buffer
	// pointers and carry no meaning to a reader, but the sample is the only
	// input REBUS is known to accept, so its scheme is reproduced exactly.
	fieldAddrBase = 0x70B9
)

type fieldType byte

const (
	typeChar    fieldType = 'C'
	typeNumeric fieldType = 'N'
	typeLogical fieldType = 'L'
)

type field struct {
	name   string
	typ    fieldType
	length uint8
}

// recordLength is the on-disk size of one record: the delete flag plus every
// field at its full declared width.
func recordLength(fields []field) int {
	total := 1
	for _, f := range fields {
		total += int(f.length)
	}
	return total
}

// writeTable serializes one dBASE II table. rows[i][j] is row i's value for
// fields[j], already transcoded to KOI8-R (so one byte per character) and
// unpadded: Character values are left aligned and cut to the field width,
// Numeric values right aligned, Logical values must be "T" or "F".
func writeTable(fields []field, rows [][]string) ([]byte, error) {
	if err := validateFields(fields); err != nil {
		return nil, err
	}
	if len(rows) > dbase2MaxRecords {
		return nil, fmt.Errorf("%d records exceed the dBASE II maximum of %d", len(rows), dbase2MaxRecords)
	}
	recordLen := recordLength(fields)
	if recordLen > dbase2MaxRecordLen {
		return nil, fmt.Errorf("record length %d exceeds the dBASE II maximum of %d", recordLen, dbase2MaxRecordLen)
	}

	out := make([]byte, dbase2HeaderLen, dbase2HeaderLen+len(rows)*recordLen+1)
	out[0] = dbase2Version
	binary.LittleEndian.PutUint16(out[1:3], uint16(len(rows)))
	// Bytes 3-5 (month, day, year) are left at zero, as in the sample file.
	// A real date would also make every export non-deterministic: sync
	// rebuilds this archive on every run and byte-identical output for
	// unchanged data is worth more than a "last updated" stamp REBUS shows
	// nowhere.
	binary.LittleEndian.PutUint16(out[6:8], uint16(recordLen))

	addr := fieldAddrBase
	for i, f := range fields {
		desc := out[8+i*dbase2FieldDescLen:]
		copy(desc[:11], f.name) // remainder stays NUL, as the format requires
		desc[11] = byte(f.typ)
		desc[12] = f.length
		binary.LittleEndian.PutUint16(desc[13:15], uint16(addr))
		addr += int(f.length)
	}
	out[8+len(fields)*dbase2FieldDescLen] = 0x0D

	for i, row := range rows {
		if len(row) != len(fields) {
			return nil, fmt.Errorf("row %d has %d values, want %d", i, len(row), len(fields))
		}
		out = append(out, recordLiveFlag)
		for j, f := range fields {
			cell, err := padValue(f, row[j])
			if err != nil {
				return nil, fmt.Errorf("row %d field %s: %w", i, f.name, err)
			}
			out = append(out, cell...)
		}
	}
	return append(out, eofMarker), nil
}

// padValue renders one value at exactly f.length bytes.
func padValue(f field, value string) ([]byte, error) {
	cell := make([]byte, f.length)
	for i := range cell {
		cell[i] = ' '
	}
	switch f.typ {
	case typeChar:
		// Longer values are cut rather than rejected: tableDef may have
		// narrowed the column below its data to fit the record limit.
		copy(cell, value)
	case typeNumeric:
		if len(value) > int(f.length) {
			return nil, fmt.Errorf("numeric value %q does not fit in %d characters", value, f.length)
		}
		copy(cell[int(f.length)-len(value):], value)
	case typeLogical:
		if value != "T" && value != "F" {
			return nil, fmt.Errorf("logical value must be T or F, got %q", value)
		}
		cell[0] = value[0]
	default:
		return nil, fmt.Errorf("unsupported field type %q", byte(f.typ))
	}
	return cell, nil
}

func validateFields(fields []field) error {
	if len(fields) == 0 {
		return fmt.Errorf("table has no fields")
	}
	if len(fields) > dbase2MaxFields {
		return fmt.Errorf("%d fields exceed the dBASE II maximum of %d", len(fields), dbase2MaxFields)
	}
	for _, f := range fields {
		if err := validateFieldName(f.name); err != nil {
			return err
		}
		if f.length == 0 {
			return fmt.Errorf("field %s has zero length", f.name)
		}
		switch f.typ {
		case typeChar:
			if int(f.length) > dbase2MaxCharLen {
				return fmt.Errorf("field %s length %d exceeds the dBASE II character maximum of %d", f.name, f.length, dbase2MaxCharLen)
			}
		case typeNumeric:
		case typeLogical:
			if f.length != 1 {
				return fmt.Errorf("logical field %s must have length 1, got %d", f.name, f.length)
			}
		default:
			return fmt.Errorf("field %s has unsupported type %q", f.name, byte(f.typ))
		}
	}
	return nil
}

// validateFieldName enforces dBASE II identifier rules: at most 10 uppercase
// ASCII letters or digits, starting with a letter. Notably the underscore is
// a dBASE III addition and is rejected here.
func validateFieldName(name string) error {
	if name == "" || len(name) > dbase2MaxNameLen {
		return fmt.Errorf("field name %q must be 1 to %d characters", name, dbase2MaxNameLen)
	}
	for i := 0; i < len(name); i++ {
		c := name[i]
		switch {
		case c >= 'A' && c <= 'Z':
		case c >= '0' && c <= '9' && i > 0:
		default:
			return fmt.Errorf("field name %q is not a valid dBASE II identifier", name)
		}
	}
	return nil
}
