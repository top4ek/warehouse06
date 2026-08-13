package rebus

import (
	"fmt"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestWriteTable_MatchesSampleFile rebuilds testdata/sample_dbase2.dbf - a
// table the target REBUS installation reads - from its own schema and data,
// and requires the result to be byte-identical. This is the reference check
// on the whole layout: header fields, the 521-byte header, the 16-byte field
// descriptors (field data addresses included), record padding and the EOF
// marker.
//
// The sample is 1024 bytes because it was taken off CP/M media, which
// allocates in blocks; everything past the EOF marker is zero fill and not
// part of the format, so only the meaningful prefix is compared.
func TestWriteTable_MatchesSampleFile(t *testing.T) {
	want, err := os.ReadFile("testdata/sample_dbase2.dbf")
	require.NoError(t, err)

	got, err := writeTable(
		[]field{
			{"NAME", typeChar, 20},
			{"AGE", typeNumeric, 3},
			{"CITY", typeChar, 15},
		},
		[][]string{
			{"VASYA", "13", "MUHOSRANSK"},
			{"KOLYA", "54", "MSK"},
			{"TANYA", "33", "UFA"},
		},
	)
	require.NoError(t, err)

	require.LessOrEqual(t, len(got), len(want))
	assert.Equal(t, want[:len(got)], got)
	assert.Equal(t, make([]byte, len(want)-len(got)), want[len(got):], "sample tail must be pure CP/M block padding")
}

func TestWriteTable_RejectsInvalidSchemas(t *testing.T) {
	tests := []struct {
		name    string
		fields  []field
		rows    [][]string
		wantErr string
	}{
		{
			name:    "underscore in field name",
			fields:  []field{{"ENTRY_ID", typeNumeric, 4}},
			wantErr: "not a valid dBASE II identifier",
		},
		{
			name:    "lowercase field name",
			fields:  []field{{"entryid", typeNumeric, 4}},
			wantErr: "not a valid dBASE II identifier",
		},
		{
			name:    "field name longer than 10 characters",
			fields:  []field{{"ENTRYCREATED", typeChar, 8}},
			wantErr: "must be 1 to 10 characters",
		},
		{
			name:    "more than 32 fields",
			fields:  manyFields(33),
			wantErr: "exceed the dBASE II maximum of 32",
		},
		{
			name:    "character field wider than 254",
			fields:  []field{{"PATH", typeChar, 255}},
			wantErr: "exceeds the dBASE II character maximum",
		},
		{
			name:    "record longer than 1000 bytes",
			fields:  []field{{"A", typeChar, 254}, {"B", typeChar, 254}, {"C", typeChar, 254}, {"D", typeChar, 254}},
			wantErr: "exceeds the dBASE II maximum of 1000",
		},
		{
			name:    "numeric value wider than its field",
			fields:  []field{{"ID", typeNumeric, 2}},
			rows:    [][]string{{"1234"}},
			wantErr: "does not fit in 2 characters",
		},
		{
			name:    "logical value that is not T or F",
			fields:  []field{{"ISIMAGE", typeLogical, 1}},
			rows:    [][]string{{"yes"}},
			wantErr: "logical value must be T or F",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			_, err := writeTable(tc.fields, tc.rows)
			require.Error(t, err)
			assert.Contains(t, err.Error(), tc.wantErr)
		})
	}
}

// TestWriteTable_PadsAndAlignsValues pins the per-type record layout: text
// left aligned and cut to the field width, numbers right aligned, logicals a
// single T/F.
func TestWriteTable_PadsAndAlignsValues(t *testing.T) {
	data, err := writeTable(
		[]field{
			{"NAME", typeChar, 4},
			{"ID", typeNumeric, 3},
			{"ISIMAGE", typeLogical, 1},
		},
		[][]string{{"overlong", "7", "T"}, {"ab", "123", "F"}},
	)
	require.NoError(t, err)

	const recordLen = 1 + 4 + 3 + 1
	require.Len(t, data, dbase2HeaderLen+2*recordLen+1)
	assert.Equal(t, byte(eofMarker), data[len(data)-1])

	rec0 := data[dbase2HeaderLen : dbase2HeaderLen+recordLen]
	assert.Equal(t, byte(recordLiveFlag), rec0[0])
	assert.Equal(t, "over", string(rec0[1:5]))
	assert.Equal(t, "  7", string(rec0[5:8]))
	assert.Equal(t, "T", string(rec0[8:9]))

	rec1 := data[dbase2HeaderLen+recordLen : dbase2HeaderLen+2*recordLen]
	assert.Equal(t, "ab  ", string(rec1[1:5]))
	assert.Equal(t, "123", string(rec1[5:8]))
	assert.Equal(t, "F", string(rec1[8:9]))
}

// TestWriteTable_FullFieldSetTerminates covers the corner where all 32 field
// descriptors are used: the 0x0D terminator lands on the header's last byte.
func TestWriteTable_FullFieldSetTerminates(t *testing.T) {
	data, err := writeTable(manyFields(32), nil)
	require.NoError(t, err)

	require.Len(t, data, dbase2HeaderLen+1)
	assert.Equal(t, byte(0x0D), data[dbase2HeaderLen-1])
	assert.Equal(t, byte(eofMarker), data[dbase2HeaderLen])
}

func manyFields(n int) []field {
	fields := make([]field, 0, n)
	for i := 0; i < n; i++ {
		fields = append(fields, field{
			name:   fmt.Sprintf("F%c%c", 'A'+i/26, 'A'+i%26),
			typ:    typeChar,
			length: 1,
		})
	}
	return fields
}
