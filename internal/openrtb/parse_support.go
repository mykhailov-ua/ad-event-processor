package openrtb

import (
	"unicode/utf8"
	_ "unsafe"
)

const (
	MaxWSkip               = 256
	MaxJSONTotalWSkip      = 4096
	MaxJSONStringScanBytes = 65536
	MaxJSONStringEscapes   = 16384
	MaxJSONKeyPairs        = 10000
	OrtbScanMaxBytes       = 262144
	OrtbMaxQuoteChecks     = 65536
)

func ParseDecimalMicro(b []byte) int64 {
	return parseDecimalMicro(b)
}

func parseDecimalMicro(b []byte) int64 {
	n := len(b)
	if n == 0 {
		return 0
	}
	var val int64
	var dec int64
	var decDigits int
	hasDec := false
parseLoop:
	for i := range n {
		c := b[i]
		switch {
		case c >= '0' && c <= '9':
			if !hasDec {
				val = val*10 + int64(c-'0')
			} else if decDigits < 6 {
				dec = dec*10 + int64(c-'0')
				decDigits++
			}
		case c == '.':
			hasDec = true
		default:
			break parseLoop
		}
	}
	for decDigits < 6 {
		dec *= 10
		decDigits++
	}
	if hasDec {
		return val*1_000_000 + dec
	}
	return val * 1_000_000
}

type jsonScanBudget struct {
	wsLeft    int
	strLeft   int
	escLeft   int
	pairsLeft int
}

func newJSONScanBudget() jsonScanBudget {
	return jsonScanBudget{
		wsLeft:    MaxJSONTotalWSkip,
		strLeft:   MaxJSONStringScanBytes,
		escLeft:   MaxJSONStringEscapes,
		pairsLeft: MaxJSONKeyPairs,
	}
}

func (b *jsonScanBudget) consumeWS(n int) bool {
	if b == nil {
		return true
	}
	b.wsLeft -= n
	return b.wsLeft >= 0
}

func (b *jsonScanBudget) consumeStrByte() bool {
	if b == nil {
		return true
	}
	b.strLeft--
	return b.strLeft >= 0
}

func (b *jsonScanBudget) consumeEscape() bool {
	if b == nil {
		return true
	}
	b.escLeft--
	return b.escLeft >= 0
}

var hexLookup [256]byte

func init() {
	for i := range hexLookup {
		hexLookup[i] = 0xff
	}
	for i := byte('0'); i <= '9'; i++ {
		hexLookup[i] = i - '0'
	}
	for i := byte('a'); i <= 'f'; i++ {
		hexLookup[i] = i - 'a' + 10
	}
	for i := byte('A'); i <= 'F'; i++ {
		hexLookup[i] = i - 'A' + 10
	}
}

func decodeJSONU16(hex []byte) (uint16, bool) {
	if len(hex) != 4 {
		return 0, false
	}
	var v uint16
	for i := range 4 {
		h := hexLookup[hex[i]]
		if h == 0xff {
			return 0, false
		}
		v = (v << 4) | uint16(h)
	}
	return v, true
}

func utf8ValidBytes(b []byte) bool {
	return utf8.Valid(b)
}

func scanJSONStringEnd(data []byte, i, n int, b *jsonScanBudget) (int, bool) {
	if i >= n || data[i] != '"' {
		return i, false
	}
	contentStart := i + 1
	i++
	for i < n {
		c := data[i]
		if c == '"' {
			if !utf8ValidBytes(data[contentStart:i]) {
				return i, false
			}
			return i + 1, true
		}
		if !b.consumeStrByte() {
			return i, false
		}
		if c == '\\' {
			if !b.consumeEscape() {
				return i, false
			}
			i++
			if i >= n {
				return i, false
			}
			switch data[i] {
			case '"', '\\', '/', 'b', 'f', 'n', 'r', 't':
				i++
			case 'u':
				i++
				if i+4 > n {
					return i, false
				}
				cp, ok := decodeJSONU16(data[i : i+4])
				if !ok {
					return i, false
				}
				switch {
				case cp >= 0xD800 && cp <= 0xDBFF:
					if i+10 > n || data[i+4] != '\\' || data[i+5] != 'u' {
						return i, false
					}
					cp2, ok := decodeJSONU16(data[i+6 : i+10])
					if !ok || cp2 < 0xDC00 || cp2 > 0xDFFF {
						return i, false
					}
					i += 10
				case cp >= 0xDC00 && cp <= 0xDFFF:
					return i, false
				default:
					i += 4
				}
			default:
				return i, false
			}
			continue
		}
		if c < 0x20 {
			return i, false
		}
		i++
	}
	return i, false
}

func hashUserIDBytes(userID []byte) uint64 {
	const (
		offset64 = 14695981039346656037
		prime64  = 1099511628211
	)
	h := uint64(offset64)
	for i := range userID {
		h ^= uint64(userID[i])
		h *= prime64
	}
	return h
}

//go:linkname monoNano runtime.nanotime
func monoNano() int64
