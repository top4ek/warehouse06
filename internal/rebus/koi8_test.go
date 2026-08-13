package rebus

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"golang.org/x/text/encoding/charmap"
)

func TestKoi8Converter_RoundTrip(t *testing.T) {
	conv := koi8Converter{}
	input := "Вектор-06Ц: игра для 8-битного компьютера, ASCII too"

	encoded, err := conv.Encode([]byte(input))
	require.NoError(t, err)

	// Decode independently via the stdlib charmap, not our own Decode, so the
	// test doesn't just check Encode/Decode agree with each other.
	decoded, err := charmap.KOI8R.NewDecoder().Bytes(encoded)
	require.NoError(t, err)
	assert.Equal(t, input, string(decoded))

	// Our own Decode should agree too.
	viaOwnDecode, err := conv.Decode(encoded)
	require.NoError(t, err)
	assert.Equal(t, input, string(viaOwnDecode))
}

func TestKoi8Converter_UnmappableRune(t *testing.T) {
	var unmappable []rune
	conv := koi8Converter{onUnmappable: func(r rune) { unmappable = append(unmappable, r) }}

	// U+1F600 (an emoji) has no KOI8-R mapping and no transliteration.
	encoded, err := conv.Encode([]byte("ok\U0001F600ok"))
	require.NoError(t, err)

	assert.Equal(t, "ok?ok", string(encoded))
	assert.Equal(t, []rune{'\U0001F600'}, unmappable)
}

// TestKoi8Converter_Transliterates covers the typography that dominates the
// real catalog: none of it has a KOI8-R byte, and all of it must survive as a
// readable stand-in rather than as '?'.
func TestKoi8Converter_Transliterates(t *testing.T) {
	var unmappable []rune
	conv := koi8Converter{onUnmappable: func(r rune) { unmappable = append(unmappable, r) }}

	input := "«Вектор-06Ц» — №1… ’x’ y"
	encoded, err := conv.Encode([]byte(input))
	require.NoError(t, err)
	assert.Empty(t, unmappable, "transliterated runes must not be reported unmappable")
	assert.Len(t, encoded, len([]rune(input)), "one byte per rune must hold for transliterations too")

	decoded, err := charmap.KOI8R.NewDecoder().Bytes(encoded)
	require.NoError(t, err)
	assert.Equal(t, `"Вектор-06Ц" - N1. 'x' y`, string(decoded))
}

func TestKoi8Converter_NeutralizesControlCharacters(t *testing.T) {
	conv := koi8Converter{}

	// U+001A must never survive transcoding: it is the DBF end-of-file
	// marker, so an embedded one could look like end-of-file to a reader.
	// The rest of the C0 range (markdown text is full of newlines) is only
	// noise inside a fixed-width Character field.
	encoded, err := conv.Encode([]byte("before\x1Aafter"))
	require.NoError(t, err)
	assert.NotContains(t, encoded, byte(0x1A))
	assert.Equal(t, "before after", string(encoded))

	encoded, err = conv.Encode([]byte("line\r\nnext\ttab\x7F"))
	require.NoError(t, err)
	assert.Equal(t, "line  next tab ", string(encoded))
}

func TestKoi8Converter_EncodedLengthEqualsRuneCount(t *testing.T) {
	conv := koi8Converter{}
	input := "Смесь ASCII и Кириллицы \U0001F600"

	encoded, err := conv.Encode([]byte(input))
	require.NoError(t, err)

	assert.Len(t, encoded, len([]rune(input)))
}
