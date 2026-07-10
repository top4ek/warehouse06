package repository

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestFormatFTSQuery(t *testing.T) {
	tests := []struct {
		name string
		in   string
		want string
	}{
		{name: "single word", in: "exolon", want: "\"exolon\"*"},
		{name: "multiple words", in: "vec tor", want: "\"vec\"* AND \"tor\"*"},
		{name: "whitespace only", in: "  ", want: ""},
		{name: "quoted term", in: `foo "bar" baz`, want: "\"foo\"* AND \"bar\"* AND \"baz\"*"},
		{name: "escaped double quote", in: `say "hi"`, want: "\"say\"* AND \"hi\"*"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, FormatFTSQuery(tt.in))
		})
	}
}
