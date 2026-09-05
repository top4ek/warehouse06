package rebus

import (
	"golang.org/x/text/encoding/charmap"
)

// koi8Translit maps runes KOI8-R has no byte for onto a KOI8-R stand-in.
// Without it the catalog's ordinary Russian typography (guillemets, em
// dashes, the numero sign) would all degrade to '?'.
//
// Every replacement MUST be exactly one byte. koi8Converter.Encode's
// one-byte-per-rune guarantee is what lets tables.go truncate by rune count
// against a byte-sized Character field, so a multi-character replacement
// (say '…' -> "...") would silently overflow those fields. That is why '…'
// collapses to a single '.' rather than an ellipsis spelled out.
var koi8Translit = map[rune]byte{
	'«': '"', '»': '"', '„': '"', '“': '"', '”': '"', '‟': '"', '″': '"',
	'‘': '\'', '’': '\'', '‚': '\'', '′': '\'', '‹': '\'', '›': '\'',
	'—': '-', '–': '-', '‒': '-', '−': '-', '‑': '-',
	'…': '.',
	'№': 'N',
	'×': 'x',
	' ': ' ', // thin space
	' ': ' ', // narrow no-break space
	// U+00A0 no-break space is deliberately absent: KOI8-R has its own
	// byte for it (0x9A), so EncodeRune maps it before this table is
	// consulted.
}

// koi8Converter transcodes between UTF-8 and KOI8-R. Control characters are
// replaced with spaces: dBASE II records carry no field separators, so a
// newline inside a fixed-width Character field is only noise for a text-mode
// reader, and 0x1A specifically is the DBF end-of-file marker - one of those
// inside a record would look like end-of-file and truncate the table. (That
// also rules out golang.org/x/text/encoding's own unmappable-rune
// substitution, ASCIISub, which is 0x1A.) Runes with no KOI8-R byte go
// through koi8Translit first and are substituted with '?' only if that
// misses too.
// Every rune, mapped or substituted, encodes to exactly one byte, so encoded
// length always equals input rune count - callers can truncate by rune count
// before encoding without needing to encode first.
// The dBASE II header has no code-page byte at all, so which single-byte
// Cyrillic encoding the output is in depends entirely on what the reading
// application assumes.
type koi8Converter struct {
	onUnmappable func(rune)
}

func (c koi8Converter) Decode(in []byte) ([]byte, error) {
	return charmap.KOI8R.NewDecoder().Bytes(in)
}

func (c koi8Converter) Encode(in []byte) ([]byte, error) {
	out := make([]byte, 0, len(in))
	for _, r := range string(in) {
		switch {
		case r < 0x20 || r == 0x7F:
			out = append(out, ' ')
		case r < 0x80:
			out = append(out, byte(r))
		default:
			if b, ok := charmap.KOI8R.EncodeRune(r); ok {
				out = append(out, b)
				continue
			}
			if b, ok := koi8Translit[r]; ok {
				out = append(out, b)
				continue
			}
			out = append(out, '?')
			if c.onUnmappable != nil {
				c.onUnmappable(r)
			}
		}
	}
	return out, nil
}
