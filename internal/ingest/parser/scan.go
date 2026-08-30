package parser

import (
	"errors"
	"sync/atomic"
	"unicode/utf8"

	"ad-event-processor/internal/config"
)

var ErrMalformed = errors.New("malformed json")

const (
	MaxJSONDepth     = 16
	OrtbMaxJSONDepth = 32

	MaxWSkip = 256

	OrtbScanMaxBytes = 262144

	OrtbMaxQuoteChecks = 65536

	MaxJSONKeyPairs = 10000
)

var strictUTF8Enabled atomic.Bool

func init() {
	strictUTF8Enabled.Store(true)
}

func ConfigureSecurity(cfg *config.Config) {
	if cfg == nil {
		strictUTF8Enabled.Store(true)
		return
	}
	strictUTF8Enabled.Store(cfg.JSONStrictUTF8)
}

func StrictUTF8Atomic() *atomic.Bool {
	return &strictUTF8Enabled
}

func strictUTF8() bool {
	return strictUTF8Enabled.Load()
}

const (
	MaxJSONTotalWSkip = 4096

	MaxJSONStringScanBytes = 65536

	MaxJSONStringEscapes = 16384
)

// ScanBudget is a multi-counter bomb budget for hand-rolled JSON scan (not a single byte cap).
// Separate ws/str/esc/pairs limits defeat whitespace-only and escape-loop DoS without rejecting valid /track.
type ScanBudget struct {
	wsLeft    int
	strLeft   int
	escLeft   int
	pairsLeft int
}

func NewScanBudget() ScanBudget {
	return ScanBudget{
		wsLeft:    MaxJSONTotalWSkip,
		strLeft:   MaxJSONStringScanBytes,
		escLeft:   MaxJSONStringEscapes,
		pairsLeft: MaxJSONKeyPairs,
	}
}

func (b *ScanBudget) ConsumeWS(n int) bool {
	if b == nil {
		return true
	}
	b.wsLeft -= n
	return b.wsLeft >= 0
}

func (b *ScanBudget) ConsumeStrByte() bool {
	if b == nil {
		return true
	}
	b.strLeft--
	return b.strLeft >= 0
}

func (b *ScanBudget) ConsumeEscape() bool {
	if b == nil {
		return true
	}
	b.escLeft--
	return b.escLeft >= 0
}

func (b *ScanBudget) ConsumeKeyPair() bool {
	if b == nil {
		return true
	}
	b.pairsLeft--
	return b.pairsLeft >= 0
}

func SkipWSBudget(data []byte, i, n int, b *ScanBudget) (int, bool) {
	skipped := 0
	for i < n && whitespaceTable[data[i]] != 0 {
		if skipped >= MaxWSkip {
			return i, false
		}
		skipped++
		i++
	}
	if !b.ConsumeWS(skipped) {
		return i, false
	}
	return i, true
}

func TrackKeyOK(key []byte) bool {
	if len(key) == 0 {
		return false
	}
	for _, c := range key {
		if c > 0x7f {
			return false
		}
	}
	return true
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

func ScanStringEnd(data []byte, i, n int, b *ScanBudget) (int, bool) {
	if i >= n || data[i] != '"' {
		return i, false
	}
	contentStart := i + 1
	i++
	for i < n {
		c := data[i]
		if c == '"' {
			if strictUTF8() && !utf8ValidBytes(data[contentStart:i]) {
				return i, false
			}
			return i + 1, true
		}
		if !b.ConsumeStrByte() {
			return i, false
		}
		if c == '\\' {
			if !b.ConsumeEscape() {
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

func ScanLiteralStringEnd(data []byte, i, n int, b *ScanBudget) (int, bool) {
	if i >= n || data[i] != '"' {
		return i, false
	}
	contentStart := i + 1
	i++
	for i < n {
		c := data[i]
		if c == '"' {
			if strictUTF8() && !utf8ValidBytes(data[contentStart:i]) {
				return i, false
			}
			return i + 1, true
		}
		if c == '\\' || c < 0x20 {
			return i, false
		}
		if !b.ConsumeStrByte() {
			return i, false
		}
		i++
	}
	return i, false
}

func utf8ValidBytes(b []byte) bool {
	return utf8.Valid(b)
}

func SkipValueBudget(data []byte, start int, bud *ScanBudget) (int, error) {
	return SkipValueBudgetDepth(data, start, bud, MaxJSONDepth)
}

// SkipValueBudgetDepth walks one JSON value; maxDepth is 16 (/track) or 32 (OpenRTB) per caller.
// Each nested { or [ consumes str budget and depth; exceeding either returns ErrMalformed.
func SkipValueBudgetDepth(data []byte, start int, bud *ScanBudget, maxDepth int) (int, error) {
	n := len(data)
	if start >= n {
		return start, ErrMalformed
	}
	_ = data[n-1]

	i := start
	tok := data[i]
	switch tok {
	case '"':
		end, ok := ScanStringEnd(data, i, n, bud)
		if !ok {
			return i, ErrMalformed
		}
		return end, nil
	case '{', '[':
		depth := 1
		if !bud.ConsumeStrByte() {
			return i, ErrMalformed
		}
		i++
		for i < n && depth > 0 {
			switch data[i] {
			case '"':
				end, ok := ScanStringEnd(data, i, n, bud)
				if !ok {
					return i, ErrMalformed
				}
				i = end
			case '{', '[':
				if !bud.ConsumeStrByte() {
					return i, ErrMalformed
				}
				depth++
				if depth > maxDepth {
					return i, ErrMalformed
				}
				i++
			case '}', ']':
				if !bud.ConsumeStrByte() {
					return i, ErrMalformed
				}
				depth--
				i++
			default:
				if !bud.ConsumeStrByte() {
					return i, ErrMalformed
				}
				i++
			}
		}
		if depth > 0 {
			return i, ErrMalformed
		}
		return i, nil
	case 't', 'f', 'n':
		for i < n && !isDelimiter(data[i]) {
			if !bud.ConsumeStrByte() {
				return i, ErrMalformed
			}
			i++
		}
		return i, nil
	default:
		for i < n && !isDelimiter(data[i]) {
			if !bud.ConsumeStrByte() {
				return i, ErrMalformed
			}
			i++
		}
		return i, nil
	}
}
