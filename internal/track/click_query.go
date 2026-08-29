package track

import (
	"ad-event-processor/internal/filter"

	"github.com/google/uuid"
)

func loadU32LE(b []byte) uint32 {
	_ = b[3]
	return uint32(b[0]) | uint32(b[1])<<8 | uint32(b[2])<<16 | uint32(b[3])<<24
}

func loadU64LE(b []byte) uint64 {
	_ = b[7]
	return uint64(b[0]) | uint64(b[1])<<8 | uint64(b[2])<<16 | uint64(b[3])<<24 |
		uint64(b[4])<<32 | uint64(b[5])<<40 | uint64(b[6])<<48 | uint64(b[7])<<56
}

func bytesEqualASCII(b []byte, s string) bool {
	if len(b) != len(s) {
		return false
	}
	for i := range len(b) {
		if b[i] != s[i] {
			return false
		}
	}
	return true
}

var clickHexLookup [256]byte

func init() {
	for i := range clickHexLookup {
		clickHexLookup[i] = 0xff
	}
	for i := byte('0'); i <= '9'; i++ {
		clickHexLookup[i] = i - '0'
	}
	for i := byte('a'); i <= 'f'; i++ {
		clickHexLookup[i] = i - 'a' + 10
	}
	for i := byte('A'); i <= 'F'; i++ {
		clickHexLookup[i] = i - 'A' + 10
	}
}

const (
	u32Type      uint32 = 0x65707974
	u32User      uint32 = 0x72657375
	u64ClickID   uint64 = 0x64695f6b63696c63
	u64Campaign  uint64 = 0x6e676961706d6163
	u64Placement uint64 = 0x6e65636d65636170
)

const (
	clickPathPrefix      = "/click"
	clickDefaultType     = "click"
	redirectHdrPrefix    = "HTTP/1.1 302 Found\r\nLocation: "
	redirectHdrSuffix    = "\r\nReferrer-Policy: no-referrer\r\nCache-Control: no-store\r\nContent-Length: 0\r\nConnection: keep-alive\r\n\r\n"
	maxClickQueryValue   = 2048
	maxRedirectLocation  = 4096
	clickQueryScratchCap = 512
	redirectWireMinCap   = 512
)

type ClickQueryKeyID uint8

const (
	clickKeyUnknown ClickQueryKeyID = iota
	clickKeyCampaignID
	clickKeyClickID
	clickKeyUserID
	clickKeyType
	clickKeyPlacementID
	clickKeySub1
	clickKeySub2
	clickKeySub3
	clickKeySub4
	clickKeySub5
	clickKeySubGeneric
	clickKeyFBCLID
	clickKeyGCLID
	clickKeyTTCLID
	clickKeyCost
	clickKeyCPC
	clickKeyBid
	clickKeyDMR
	clickKeyExpires
	clickKeySig
	clickKeySmoke
)

const (
	u32Sub1 uint32 = 0x31627573
	u32Sub2 uint32 = 0x32627573
	u32Sub3 uint32 = 0x33627573
	u32Sub4 uint32 = 0x34627573
	u32Sub5 uint32 = 0x35627573
)

type RedirectMacroID uint8

const (
	redirectMacroNone RedirectMacroID = iota
	redirectMacroClickID
	redirectMacroUserID
	redirectMacroSub1
	redirectMacroSub2
	redirectMacroSub3
	redirectMacroSub4
	redirectMacroSub5
)

const (
	redirectMacroClickLen = 10
	redirectMacroUserLen  = 9
	redirectMacroSubLen   = 6
)

type ClickQueryParsed struct {
	CampaignID              uuid.UUID
	EventType               string
	UserID                  string
	ClickID                 string
	PlacementID             string
	Subs                    SubIDSlots
	FBCLID                  string
	GCLID                   string
	TTCLID                  string
	IngressCost             []byte
	IngressCPC              []byte
	IngressBid              []byte
	Passthrough             []byte
	DMR                     bool
	LinkExpires             int64
	LinkSig                 string
	IPv6RotationShadow      bool
	IPv4RotationShadow      bool
	AttestationLightMissing bool
	ReviewTrafficMatched    bool
	Smoke                   bool
	OK                      bool
}

func (p *ClickQueryParsed) Reset() {
	p.CampaignID = uuid.Nil
	p.EventType = ""
	p.UserID = ""
	p.ClickID = ""
	p.PlacementID = ""
	p.Subs.Reset()
	p.FBCLID = ""
	p.GCLID = ""
	p.TTCLID = ""
	p.IngressCost = nil
	p.IngressCPC = nil
	p.IngressBid = nil
	p.Passthrough = nil
	p.DMR = false
	p.LinkExpires = 0
	p.LinkSig = ""
	p.IPv6RotationShadow = false
	p.IPv4RotationShadow = false
	p.AttestationLightMissing = false
	p.ReviewTrafficMatched = false
	p.Smoke = false
	p.OK = false
}

func MatchClickQueryKey(key []byte) ClickQueryKeyID {
	switch len(key) {
	case 3:
		if key[0] == 'd' && key[1] == 'm' && key[2] == 'r' {
			return clickKeyDMR
		}
		if key[0] == 'c' && key[1] == 'p' && key[2] == 'c' {
			return clickKeyCPC
		}
		if key[0] == 'b' && key[1] == 'i' && key[2] == 'd' {
			return clickKeyBid
		}
	case 4:
		if loadU32LE(key) == 0x74736f63 {
			return clickKeyCost
		}
		if key[0] == '_' && key[1] == 's' && key[2] == 'i' && key[3] == 'g' {
			return clickKeySig
		}
		switch loadU32LE(key) {
		case u32Type:
			return clickKeyType
		case u32Sub1:
			return clickKeySub1
		case u32Sub2:
			return clickKeySub2
		case u32Sub3:
			return clickKeySub3
		case u32Sub4:
			return clickKeySub4
		case u32Sub5:
			return clickKeySub5
		}
		if idx, OK := SubKeyIndex(key); OK && idx >= 6 {
			return clickKeySubGeneric
		}
	case 5:
		if key[0] == 's' && key[1] == 'm' && key[2] == 'o' && key[3] == 'k' && key[4] == 'e' {
			return clickKeySmoke
		}
		if loadU32LE(key) == 0x696c6367 && key[4] == 'd' {
			return clickKeyGCLID
		}
		if idx, OK := SubKeyIndex(key); OK && idx >= 10 {
			return clickKeySubGeneric
		}
	case 6:
		switch loadU32LE(key) {
		case 0x6c636266:
			if key[4] == 'i' && key[5] == 'd' {
				return clickKeyFBCLID
			}
		case 0x6c637474:
			if key[4] == 'i' && key[5] == 'd' {
				return clickKeyTTCLID
			}
		}
	case 7:
		if key[0] == 'e' && key[1] == 'x' && key[2] == 'p' && key[3] == 'i' &&
			key[4] == 'r' && key[5] == 'e' && key[6] == 's' {
			return clickKeyExpires
		}
		if loadU32LE(key) == u32User && key[4] == '_' && key[5] == 'i' && key[6] == 'd' {
			return clickKeyUserID
		}
	case 8:
		if loadU64LE(key) == u64ClickID {
			return clickKeyClickID
		}
	case 11:
		if loadU64LE(key) == u64Campaign && key[8] == '_' && key[9] == 'i' && key[10] == 'd' {
			return clickKeyCampaignID
		}
	case 12:
		if loadU64LE(key) == u64Placement && key[8] == '_' && key[9] == 'i' && key[10] == 'd' {
			return clickKeyPlacementID
		}
	}
	return clickKeyUnknown
}

func SplitClickPathQuery(path []byte) (base, query []byte, OK bool) {
	if len(path) < len(clickPathPrefix) {
		return nil, nil, false
	}
	if !bytesEqualASCII(path[:len(clickPathPrefix)], clickPathPrefix) {
		return nil, nil, false
	}
	if len(path) == len(clickPathPrefix) {
		return path[:len(clickPathPrefix)], nil, true
	}
	switch path[len(clickPathPrefix)] {
	case '?':
		return path[:len(clickPathPrefix)], path[len(clickPathPrefix)+1:], true
	case '/':
		return nil, nil, false
	default:
		return nil, nil, false
	}
}

func QueryNeedsPctDecode(src []byte) bool {
	for i := range len(src) {
		switch src[i] {
		case '%', '+':
			return true
		}
	}
	return false
}

func AppendPctDecoded(dst, src []byte) []byte {
	n := len(src)
	if n == 0 {
		return dst
	}
	if !QueryNeedsPctDecode(src) {
		return append(dst, src...)
	}
	_ = src[n-1]
	for i := range n {
		c := src[i]
		if c == '%' {
			if i+2 >= n {
				dst = append(dst, c)
				continue
			}
			hi := clickHexLookup[src[i+1]]
			lo := clickHexLookup[src[i+2]]
			if hi == 0xff || lo == 0xff {
				dst = append(dst, c)
				continue
			}
			dst = append(dst, (hi<<4)|lo)
			i += 2
			continue
		}
		if c == '+' {
			dst = append(dst, ' ')
			continue
		}
		dst = append(dst, c)
	}
	return dst
}

func ParseClickQuery(path []byte, scratch []byte, out *ClickQueryParsed) []byte {
	out.Reset()
	_, query, OK := SplitClickPathQuery(path)
	if !OK || len(query) == 0 {
		return scratch
	}
	_ = query[len(query)-1]

	scratch = scratch[:0]
	firstPassthrough := true

	for start := 0; start < len(query); {
		end := start
		for end < len(query) && query[end] != '&' {
			end++
		}
		seg := query[start:end]
		start = end
		if start < len(query) && query[start] == '&' {
			start++
		}
		if len(seg) == 0 {
			continue
		}
		eq := -1
		for i := range seg {
			if seg[i] == '=' {
				eq = i
				break
			}
		}
		var key, val []byte
		if eq < 0 {
			key = seg
		} else {
			key = seg[:eq]
			val = seg[eq+1:]
		}
		if len(val) > maxClickQueryValue {
			return scratch
		}

		kid := MatchClickQueryKey(key)
		if kid == clickKeyUnknown {
			if !firstPassthrough {
				scratch = append(scratch, '&')
			}
			firstPassthrough = false
			scratch = append(scratch, seg...)
			continue
		}

		valStart := len(scratch)
		scratch = AppendPctDecoded(scratch, val)
		decoded := scratch[valStart:]

		switch kid {
		case clickKeyCampaignID:
			if !filter.ParseUUID(decoded, &out.CampaignID) {
				return scratch
			}
		case clickKeyType:
			out.EventType = filter.UnsafeString(decoded)
		case clickKeyUserID:
			out.UserID = filter.UnsafeString(decoded)
		case clickKeyClickID:
			out.ClickID = filter.UnsafeString(decoded)
		case clickKeyPlacementID:
			out.PlacementID = filter.UnsafeString(decoded)
		case clickKeySub1:
			out.Subs[0] = filter.UnsafeString(decoded)
		case clickKeySub2:
			out.Subs[1] = filter.UnsafeString(decoded)
		case clickKeySub3:
			out.Subs[2] = filter.UnsafeString(decoded)
		case clickKeySub4:
			out.Subs[3] = filter.UnsafeString(decoded)
		case clickKeySub5:
			out.Subs[4] = filter.UnsafeString(decoded)
		case clickKeySubGeneric:
			if idx, OK := SubKeyIndex(key); OK {
				out.Subs[idx-1] = filter.UnsafeString(decoded)
			}
		case clickKeyFBCLID:
			out.FBCLID = filter.UnsafeString(decoded)
		case clickKeyGCLID:
			out.GCLID = filter.UnsafeString(decoded)
		case clickKeyTTCLID:
			out.TTCLID = filter.UnsafeString(decoded)
		case clickKeyCost:
			out.IngressCost = decoded
		case clickKeyCPC:
			out.IngressCPC = decoded
		case clickKeyBid:
			out.IngressBid = decoded
		case clickKeyDMR:
			out.DMR = ParseDmrQueryFlag(decoded)
		case clickKeyExpires:
			if exp, OK := ParseLinkExpires(decoded); OK {
				out.LinkExpires = exp
			}
		case clickKeySig:
			out.LinkSig = filter.UnsafeString(decoded)
		case clickKeySmoke:
			out.Smoke = ParseSmokeQueryFlag(decoded)
		}
	}

	if out.CampaignID == uuid.Nil {
		return scratch
	}
	if out.EventType == "" {
		out.EventType = clickDefaultType
	}
	out.Passthrough = scratch
	out.OK = true
	return scratch
}

func ParseSmokeQueryFlag(decoded []byte) bool {
	return ParseDmrQueryFlag(decoded)
}

func RedirectBaseValid(base []byte) bool {
	n := len(base)
	if n < 8 {
		return false
	}
	_ = base[n-1]
	if base[0] != 'h' || base[1] != 't' || base[2] != 't' || base[3] != 'p' {
		return false
	}
	if base[4] == ':' && base[5] == '/' && base[6] == '/' {
		return true
	}
	if n >= 9 && base[4] == 's' && base[5] == ':' && base[6] == '/' && base[7] == '/' {
		return true
	}
	return false
}

func DispatchRedirectMacro(base []byte, i int) (RedirectMacroID, int) {
	n := len(base)
	if i >= n || base[i] != '{' || i+1 >= n {
		return redirectMacroNone, i
	}
	switch base[i+1] {
	case 'c':
		if i+redirectMacroClickLen <= n &&
			base[i+2] == 'l' && base[i+3] == 'i' && base[i+4] == 'c' && base[i+5] == 'k' &&
			base[i+6] == '_' && base[i+7] == 'i' && base[i+8] == 'd' && base[i+9] == '}' {
			return redirectMacroClickID, i + redirectMacroClickLen
		}
	case 'u':
		if i+redirectMacroUserLen <= n &&
			base[i+2] == 's' && base[i+3] == 'e' && base[i+4] == 'r' &&
			base[i+5] == '_' && base[i+6] == 'i' && base[i+7] == 'd' && base[i+8] == '}' {
			return redirectMacroUserID, i + redirectMacroUserLen
		}
	case 's':
		if i+redirectMacroSubLen <= n && base[i+2] == 'u' && base[i+3] == 'b' && base[i+5] == '}' {
			digit := base[i+4]
			if digit >= '1' && digit <= '9' {
				return RedirectMacroID(redirectMacroSub1 + RedirectMacroID(digit-'1')), i + redirectMacroSubLen
			}
		}
		if i+redirectMacroSubLen+1 <= n && base[i+2] == 'u' && base[i+3] == 'b' {
			d1, d2 := base[i+4], base[i+5]
			if d1 >= '1' && d1 <= '3' && d2 >= '0' && d2 <= '9' {
				idx := int(d1-'0')*10 + int(d2-'0')
				if idx >= 10 && idx <= MaxSubIDs && base[i+6] == '}' {
					return RedirectMacroID(redirectMacroSub1 + RedirectMacroID(idx-1)), i + redirectMacroSubLen + 1
				}
			}
		}
	}
	return redirectMacroNone, i
}

const redirectMacroHex = "0123456789ABCDEF"

func RedirectMacroByteUnreserved(c byte) bool {
	switch {
	case c >= 'a' && c <= 'z', c >= 'A' && c <= 'Z', c >= '0' && c <= '9':
		return true
	case c == '-', c == '_', c == '.', c == '~':
		return true
	default:
		return false
	}
}

func AppendRedirectMacroEscaped(dst, src []byte) []byte {
	n := len(src)
	if n == 0 {
		return dst
	}
	_ = src[n-1]
	for i := range n {
		c := src[i]
		if RedirectMacroByteUnreserved(c) {
			dst = append(dst, c)
			continue
		}
		dst = append(dst, '%', redirectMacroHex[c>>4], redirectMacroHex[c&0x0f])
	}
	return dst
}

func ExpandRedirectMacros(dst, base []byte, ClickID, UserID string, Subs SubIDSlots) []byte {
	clickB := filter.UnsafeBytes(ClickID)
	userB := filter.UnsafeBytes(UserID)
	subB := [MaxSubIDs][]byte{}
	for i := range MaxSubIDs {
		subB[i] = filter.UnsafeBytes(Subs[i])
	}

	n := len(base)
	if n == 0 {
		return dst
	}
	_ = base[n-1]

	i := 0
	for i < n {
		if base[i] != '{' {
			dst = append(dst, base[i])
			i++
			continue
		}
		mid, end := DispatchRedirectMacro(base, i)
		switch mid {
		case redirectMacroClickID:
			dst = AppendRedirectMacroEscaped(dst, clickB)
			i = end
		case redirectMacroUserID:
			dst = AppendRedirectMacroEscaped(dst, userB)
			i = end
		default:
			if mid >= redirectMacroSub1 && mid < redirectMacroSub1+RedirectMacroID(MaxSubIDs) {
				dst = AppendRedirectMacroEscaped(dst, subB[mid-redirectMacroSub1])
				i = end
				continue
			}
			dst = append(dst, base[i])
			i++
		}
	}
	return dst
}

func BuildRedirectLocation(dst, base []byte, ClickID, UserID string, Subs SubIDSlots, Passthrough []byte) ([]byte, bool) {
	if !RedirectBaseValid(base) {
		return dst, false
	}
	dst = dst[:0]
	dst = ExpandRedirectMacros(dst, base, ClickID, UserID, Subs)
	if len(dst) > maxRedirectLocation {
		return dst, false
	}
	if len(Passthrough) == 0 {
		return dst, true
	}
	sep := byte('?')
	for i := range len(dst) {
		if dst[i] == '?' {
			sep = '&'
			break
		}
	}
	if len(dst)+1+len(Passthrough) > maxRedirectLocation {
		return dst, false
	}
	dst = append(dst, sep)
	dst = append(dst, Passthrough...)
	return dst, true
}
