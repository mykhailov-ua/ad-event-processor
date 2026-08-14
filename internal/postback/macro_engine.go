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
	TokenMacroSubBase   TokenKind = 32
	TokenMacroSubIDBase TokenKind = 62
)

const maxSubMacroSlots = 30

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

func parseSubDigits(s string) (int, bool) {
	switch len(s) {
	case 1:
		if s[0] < '1' || s[0] > '9' {
			return 0, false
		}
		return int(s[0] - '0'), true
	case 2:
		if s[0] < '1' || s[0] > '3' || s[1] < '0' || s[1] > '9' {
			return 0, false
		}
		idx := int(s[0]-'0')*10 + int(s[1]-'0')
		if idx < 10 || idx > maxSubMacroSlots {
			return 0, false
		}
		return idx, true
	default:
		return 0, false
	}
}

func parseSubMacroName(name string, subidStyle bool) (int, bool) {
	n := len(name)
	if subidStyle {
		if n < 6 || !eqFoldASCII(name[:5], "subid") {
			return 0, false
		}
		return parseSubDigits(name[5:])
	}
	if n < 4 || !eqFoldASCII(name[:3], "sub") {
		return 0, false
	}
	return parseSubDigits(name[3:])
}

func parseMacroKind(name string) (TokenKind, bool) {
	if idx, ok := parseSubMacroName(name, false); ok {
		return TokenMacroSubBase + TokenKind(idx-1), true
	}
	if idx, ok := parseSubMacroName(name, true); ok {
		return TokenMacroSubIDBase + TokenKind(idx-1), true
	}
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
			return TokenMacroSubIDBase, true
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
	SubIDs    [maxSubMacroSlots]string
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
	for i := range n {
		kind := TokenKind(kinds[i])
		switch kind {
		case TokenStatic:
			dst = appendStringInline(dst, statics[i])
		case TokenMacroClickID:
			dst = appendStringInline(dst, ctx.ClickID)
		case TokenMacroPayout:
			dst = appendStringInline(dst, ctx.Payout)
		case TokenMacroTxID:
			dst = appendStringInline(dst, ctx.TxID)
		case TokenMacroSubID1:
			dst = appendStringInline(dst, ctx.SubIDs[0])
		case TokenMacroParam10:
			dst = appendStringInline(dst, ctx.SubIDs[9])
		case TokenMacroEventType:
			dst = appendStringInline(dst, ctx.EventType)
		default:
			if kind >= TokenMacroSubBase && kind < TokenMacroSubBase+maxSubMacroSlots {
				dst = appendStringInline(dst, ctx.SubIDs[kind-TokenMacroSubBase])
			} else if kind >= TokenMacroSubIDBase && kind < TokenMacroSubIDBase+maxSubMacroSlots {
				dst = appendStringInline(dst, ctx.SubIDs[kind-TokenMacroSubIDBase])
			}
		}
	}
	return dst
}

func (mt *MacroTemplate) RenderStack(ctx *EventContext, scratch *[MaxRenderedURLLen]byte) []byte {
	return mt.RenderAppend(scratch[:0], ctx)
}

func PopulateEventContextSubs(evt *EventContext, payload *PostbackPayload) {
	if payload == nil {
		return
	}
	evt.SubIDs = payload.SubIDs()
}
