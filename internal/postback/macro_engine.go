package postback

import (
	"unsafe"
)

const MaxRenderedURLLen = 2048

const maxInlineTokens = 48

type TokenKind uint8

const (
	TokenStatic TokenKind = iota
	TokenMacroClickID
	TokenMacroPayout
	TokenMacroTxID
	TokenMacroSubID1
	TokenMacroParam10
	TokenMacroEventType
)

type MacroTemplate struct {
	kinds      [maxInlineTokens]uint8
	staticVals [maxInlineTokens]string
	length     uint8
	slab       []byte
}

func ParseTemplate(tpl string) *MacroTemplate {
	mt := &MacroTemplate{}
	mt.slab = make([]byte, len(tpl))
	copy(mt.slab, tpl)
	owned := unsafe.String(&mt.slab[0], len(mt.slab))

	lastIdx := 0
	for lastIdx < len(owned) {
		rest := owned[lastIdx:]
		start := indexByte(rest, '{')
		if start < 0 {
			if lastIdx < len(owned) {
				mt.pushToken(TokenStatic, owned[lastIdx:])
			}
			break
		}
		startIdx := lastIdx + start
		end := indexByte(owned[startIdx:], '}')
		if end < 0 {
			mt.pushToken(TokenStatic, owned[lastIdx:])
			break
		}
		endIdx := startIdx + end

		if startIdx > lastIdx {
			mt.pushToken(TokenStatic, owned[lastIdx:startIdx])
		}

		macro := owned[startIdx+1 : endIdx]
		if kind, ok := parseMacroKind(macro); ok {
			mt.pushToken(kind, "")
		} else {
			mt.pushToken(TokenStatic, owned[startIdx:endIdx+1])
		}
		lastIdx = endIdx + 1
	}
	return mt
}

func (mt *MacroTemplate) pushToken(kind TokenKind, static string) {
	i := int(mt.length)
	if i >= maxInlineTokens {
		return
	}
	mt.kinds[i] = uint8(kind)
	if kind == TokenStatic {
		mt.staticVals[i] = static
	}
	mt.length++
}

func indexByte(s string, c byte) int {
	for i := 0; i < len(s); i++ {
		if s[i] == c {
			return i
		}
	}
	return -1
}

func parseMacroKind(name string) (TokenKind, bool) {
	switch len(name) {
	case 5:
		if eqFoldASCII(name, "tx_id") {
			return TokenMacroTxID, true
		}
	case 6:
		if eqFoldASCII(name, "payout") {
			return TokenMacroPayout, true
		}
		if eqFoldASCII(name, "subid1") {
			return TokenMacroSubID1, true
		}
	case 7:
		if eqFoldASCII(name, "param10") {
			return TokenMacroParam10, true
		}
	case 8:
		if eqFoldASCII(name, "click_id") {
			return TokenMacroClickID, true
		}
	case 10:
		if eqFoldASCII(name, "event_type") {
			return TokenMacroEventType, true
		}
	}
	return 0, false
}

func eqFoldASCII(s, lit string) bool {
	if len(s) != len(lit) {
		return false
	}
	for i := 0; i < len(s); i++ {
		a := s[i]
		if a >= 'A' && a <= 'Z' {
			a += 'a' - 'A'
		}
		b := lit[i]
		if a != b {
			return false
		}
	}
	return true
}

type EventContext struct {
	ClickID   string
	Payout    string
	TxID      string
	SubID1    string
	Param10   string
	EventType string
}

func appendStringInline(dst []byte, s string) []byte {
	n := len(s)
	if n == 0 {
		return dst
	}
	old := len(dst)
	end := old + n
	if cap(dst) < end {
		return append(dst, s...)
	}
	dst = dst[:end]
	copy(dst[old:], s)
	return dst
}

func (mt *MacroTemplate) RenderAppend(dst []byte, ctx *EventContext) []byte {
	n := int(mt.length)
	kinds := mt.kinds[:n]
	statics := mt.staticVals[:n]
	for i := 0; i < n; i++ {
		switch TokenKind(kinds[i]) {
		case TokenStatic:
			dst = appendStringInline(dst, statics[i])
		case TokenMacroClickID:
			dst = appendStringInline(dst, ctx.ClickID)
		case TokenMacroPayout:
			dst = appendStringInline(dst, ctx.Payout)
		case TokenMacroTxID:
			dst = appendStringInline(dst, ctx.TxID)
		case TokenMacroSubID1:
			dst = appendStringInline(dst, ctx.SubID1)
		case TokenMacroParam10:
			dst = appendStringInline(dst, ctx.Param10)
		case TokenMacroEventType:
			dst = appendStringInline(dst, ctx.EventType)
		}
	}
	return dst
}

func (mt *MacroTemplate) RenderStack(ctx *EventContext, scratch *[MaxRenderedURLLen]byte) []byte {
	return mt.RenderAppend(scratch[:0], ctx)
}
