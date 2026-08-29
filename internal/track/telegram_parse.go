package track

import (
	"bytes"
	"encoding/hex"
	"unsafe"

	"ad-event-processor/internal/filter"

	"github.com/google/uuid"
)

type TelegramBidRequest struct {
	IP          []byte
	UA          []byte
	PublisherID []byte
	TelegramID  []byte
	WidgetID    []byte
	BidFloor    float64
	Premium     bool
	Motivated   bool
	Width       int32
	Height      int32
	Production  bool
}

func ParseASCIIInt(b []byte) int {
	res := 0
	for i := range b {
		c := b[i]
		if c >= '0' && c <= '9' {
			res = res*10 + int(c-'0')
		}
	}
	return res
}

func ParseASCIIFloat(b []byte) float64 {
	res := 0.0
	dec := -1.0
	for i := range b {
		c := b[i]
		if c >= '0' && c <= '9' {
			if dec < 0 {
				res = res*10.0 + float64(c-'0')
			} else {
				res += float64(c-'0') * dec
				dec *= 0.1
			}
		} else if c == '.' {
			dec = 0.1
		}
	}
	return res
}

func ParseTelegramBidRequest(body []byte, out *TelegramBidRequest) bool {
	*out = TelegramBidRequest{}
	if len(body) == 0 {
		return false
	}
	for i := 0; i < len(body); {
		if body[i] != '"' {
			i++
			continue
		}
		i++
		keyStart := i
		for i < len(body) && body[i] != '"' {
			i++
		}
		if i >= len(body) {
			break
		}
		key := body[keyStart:i]
		i++
		for i < len(body) && body[i] != ':' {
			i++
		}
		if i >= len(body) {
			break
		}
		i++
		for i < len(body) && (body[i] == ' ' || body[i] == '\t' || body[i] == '\r' || body[i] == '\n') {
			i++
		}
		if i >= len(body) {
			break
		}
		if body[i] == '"' {
			i++
			valStart := i
			for i < len(body) && body[i] != '"' {
				i++
			}
			val := body[valStart:i]
			i++
			switch {
			case bytes.Equal(key, []byte("ip")):
				out.IP = val
			case bytes.Equal(key, []byte("user_agent")), bytes.Equal(key, []byte("ua")):
				out.UA = val
			case bytes.Equal(key, []byte("publisher_id")):
				out.PublisherID = val
			case bytes.Equal(key, []byte("telegram_id")):
				out.TelegramID = val
			case bytes.Equal(key, []byte("widget_id")):
				out.WidgetID = val
			}
		} else {
			valStart := i
			for i < len(body) && body[i] != ',' && body[i] != '}' && body[i] != ']' && body[i] != ' ' && body[i] != '\n' && body[i] != '\r' && body[i] != '\t' {
				i++
			}
			val := body[valStart:i]
			switch {
			case bytes.Equal(key, []byte("premium")):
				out.Premium = bytes.Equal(val, []byte("true"))
			case bytes.Equal(key, []byte("motivated")):
				out.Motivated = bytes.Equal(val, []byte("true"))
			case bytes.Equal(key, []byte("production")):
				out.Production = bytes.Equal(val, []byte("true"))
			case bytes.Equal(key, []byte("bid_floor")):
				out.BidFloor = ParseASCIIFloat(val)
			case bytes.Equal(key, []byte("width")):
				out.Width = int32(ParseASCIIInt(val))
			case bytes.Equal(key, []byte("height")):
				out.Height = int32(ParseASCIIInt(val))
			}
		}
	}
	return len(out.PublisherID) > 0
}

func AppendUintStr(dst []byte, v uint64) []byte {
	if v == 0 {
		return append(dst, '0')
	}
	var tmp [20]byte
	i := len(tmp)
	for v > 0 {
		i--
		tmp[i] = byte('0' + v%10)
		v /= 10
	}
	return append(dst, tmp[i:]...)
}

func AppendFloatStr(dst []byte, f float64) []byte {
	if f == 0.0 {
		return append(dst, "0.0"...)
	}
	whole := int64(f)
	dst = AppendUintStr(dst, uint64(whole))
	dst = append(dst, '.')
	frac := int64((f - float64(whole)) * 1000000)
	if frac < 0 {
		frac = -frac
	}
	var tmp [6]byte
	for i := 5; i >= 0; i-- {
		tmp[i] = byte('0' + frac%10)
		frac /= 10
	}
	end := 6
	for end > 1 && tmp[end-1] == '0' {
		end--
	}
	return append(dst, tmp[:end]...)
}

func AppendUUIDStr(dst []byte, u uuid.UUID) []byte {
	var buf [36]byte
	hex.Encode(buf[0:8], u[0:4])
	buf[8] = '-'
	hex.Encode(buf[9:13], u[4:6])
	buf[13] = '-'
	hex.Encode(buf[14:18], u[6:8])
	buf[18] = '-'
	hex.Encode(buf[19:23], u[8:10])
	buf[23] = '-'
	hex.Encode(buf[24:36], u[10:16])
	return append(dst, buf[:]...)
}

const (
	telegramPathClick      = "/tg/click"
	telegramPathImpression = "/tg/impression"
)

var (
	RespTelegram204 = []byte("HTTP/1.1 204 No Content\r\nContent-Length: 0\r\nConnection: keep-alive\r\n\r\n")
	RespTelegram400 = []byte("HTTP/1.1 400 Bad Request\r\nContent-Length: 15\r\nContent-Type: text/plain\r\nConnection: keep-alive\r\n\r\nInvalid Request")
	RespTelegram404 = []byte("HTTP/1.1 404 Not Found\r\nContent-Length: 9\r\nContent-Type: text/plain\r\nConnection: keep-alive\r\n\r\nNot Found")

	telegramBridgePayloadPrefix = []byte(`{"bridge_token":"`)
	telegramBridgePayloadSuffix = []byte(`"}`)
)

type TelegramQueryParsed struct {
	CampaignID  uuid.UUID
	ClickID     uuid.UUID
	ClickIDStr  string
	BridgeToken string
	PlacementID string
	Premium     bool
	Motivated   bool
	Subs        [5]string
	DMR         bool
	Passthrough []byte
	OK          bool
}

type TelegramKeyID uint8

const (
	telegramKeyUnknown TelegramKeyID = iota
	telegramKeyCampaignID
	telegramKeyClickID
	telegramKeyBridgeToken
	telegramKeyPlacementID
	telegramKeyPremium
	telegramKeyMotivated
	telegramKeySub1
	telegramKeySub2
	telegramKeySub3
	telegramKeySub4
	telegramKeySub5
	telegramKeyDMR
	telegramKeyForbidden
)

const (
	telegramRedirectMacroClickLen  = 10
	telegramRedirectMacroBridgeLen = 14
	telegramRedirectMacroSubLen    = 6
)

var (
	telegramMacroClick  = [telegramRedirectMacroClickLen]byte{'{', 'c', 'l', 'i', 'c', 'k', '_', 'i', 'd', '}'}
	telegramMacroBridge = [telegramRedirectMacroBridgeLen]byte{'{', 'b', 'r', 'i', 'd', 'g', 'e', '_', 't', 'o', 'k', 'e', 'n', '}'}
)

func MatchTelegramQueryKey(key []byte) TelegramKeyID {
	kl := len(key)
	if kl == 0 {
		return telegramKeyUnknown
	}
	_ = key[kl-1]
	switch kl {
	case 3:
		if key[0] == 'd' && key[1] == 'm' && key[2] == 'r' {
			return telegramKeyDMR
		}
	case 4:
		if key[0] == 'h' && key[1] == 'a' && key[2] == 's' && key[3] == 'h' {
			return telegramKeyForbidden
		}
		if key[0] == 'u' && key[1] == 's' && key[2] == 'e' && key[3] == 'r' {
			return telegramKeyForbidden
		}
		if key[0] == 's' && key[1] == 'u' && key[2] == 'b' {
			switch key[3] {
			case '1':
				return telegramKeySub1
			case '2':
				return telegramKeySub2
			case '3':
				return telegramKeySub3
			case '4':
				return telegramKeySub4
			case '5':
				return telegramKeySub5
			}
		}
	case 7:
		if key[0] == 'p' && key[1] == 'r' && key[2] == 'e' && key[3] == 'm' && key[4] == 'i' && key[5] == 'u' && key[6] == 'm' {
			return telegramKeyPremium
		}
	case 8:
		if key[0] == 'c' && key[1] == 'l' && key[2] == 'i' && key[3] == 'c' && key[4] == 'k' && key[5] == '_' && key[6] == 'i' && key[7] == 'd' {
			return telegramKeyClickID
		}
		if key[0] == 'i' && key[1] == 'n' && key[2] == 'i' && key[3] == 't' && key[4] == 'D' && key[5] == 'a' && key[6] == 't' && key[7] == 'a' {
			return telegramKeyForbidden
		}
	case 9:
		if key[0] == 'a' && key[1] == 'u' && key[2] == 't' && key[3] == 'h' && key[4] == '_' && key[5] == 'd' && key[6] == 'a' && key[7] == 't' && key[8] == 'e' {
			return telegramKeyForbidden
		}
		if key[0] == 'm' && key[1] == 'o' && key[2] == 't' && key[3] == 'i' && key[4] == 'v' && key[5] == 'a' && key[6] == 't' && key[7] == 'e' && key[8] == 'd' {
			return telegramKeyMotivated
		}
		if key[0] == 's' && key[1] == 'i' && key[2] == 'g' && key[3] == 'n' && key[4] == 'a' && key[5] == 't' && key[6] == 'u' && key[7] == 'r' && key[8] == 'e' {
			return telegramKeyForbidden
		}
	case 11:
		if key[0] == 'c' && key[1] == 'a' && key[2] == 'm' && key[3] == 'p' && key[4] == 'a' && key[5] == 'i' && key[6] == 'g' && key[7] == 'n' && key[8] == '_' && key[9] == 'i' && key[10] == 'd' {
			return telegramKeyCampaignID
		}
	case 12:
		if key[0] == 'b' && key[1] == 'r' && key[2] == 'i' && key[3] == 'd' && key[4] == 'g' && key[5] == 'e' && key[6] == '_' && key[7] == 't' && key[8] == 'o' && key[9] == 'k' && key[10] == 'e' && key[11] == 'n' {
			return telegramKeyBridgeToken
		}
		if key[0] == 'p' && key[1] == 'l' && key[2] == 'a' && key[3] == 'c' && key[4] == 'e' && key[5] == 'm' && key[6] == 'e' && key[7] == 'n' && key[8] == 't' && key[9] == '_' && key[10] == 'i' && key[11] == 'd' {
			return telegramKeyPlacementID
		}
	}
	return telegramKeyUnknown
}

func TelegramQueryFlagTrue(decoded []byte) bool {
	if len(decoded) == 0 {
		return false
	}
	if decoded[0] == '1' {
		return true
	}
	return len(decoded) == 4 && decoded[0] == 't' && decoded[1] == 'r' && decoded[2] == 'u' && decoded[3] == 'e'
}

func MarshalTelegramBridgePayload(dst []byte, token string) []byte {
	dst = dst[:0]
	dst = append(dst, telegramBridgePayloadPrefix...)
	dst = append(dst, token...)
	dst = append(dst, telegramBridgePayloadSuffix...)
	return dst
}

func ValidateBridgeToken(b []byte) bool {
	bn := len(b)
	if bn == 0 || bn > 64 {
		return false
	}
	_ = b[bn-1]
	for i := range bn {
		c := b[i]
		if (c < 'a' || c > 'z') && (c < 'A' || c > 'Z') && (c < '0' || c > '9') && c != '_' && c != '-' {
			return false
		}
	}
	return true
}

func ParseTelegramQuery(path []byte, scratch []byte, out *TelegramQueryParsed) []byte {
	out.CampaignID = uuid.Nil
	out.ClickID = uuid.Nil
	out.ClickIDStr = ""
	out.BridgeToken = ""
	out.PlacementID = ""
	out.Premium = false
	out.Motivated = false
	out.Subs = [5]string{}
	out.DMR = false
	out.Passthrough = nil
	out.OK = false

	var query []byte
	qIdx := -1
	pn := len(path)
	if pn > 0 {
		_ = path[pn-1]
	}
	for i := range pn {
		if path[i] == '?' {
			qIdx = i
			break
		}
	}
	if qIdx >= 0 {
		query = path[qIdx+1:]
	}
	qn := len(query)
	if qn == 0 {
		return scratch
	}
	_ = query[qn-1]

	scratch = scratch[:0]
	firstPassthrough := true

	for start := 0; start < qn; {
		end := start
		for end < qn && query[end] != '&' {
			end++
		}
		seg := query[start:end]
		start = end
		if start < qn && query[start] == '&' {
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

		kid := MatchTelegramQueryKey(key)
		if kid == telegramKeyForbidden {
			return scratch
		}
		if kid == telegramKeyUnknown {
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
		case telegramKeyCampaignID:
			if !filter.ParseUUID(decoded, &out.CampaignID) {
				return scratch
			}
		case telegramKeyClickID:
			if !filter.ParseUUID(decoded, &out.ClickID) {
				return scratch
			}
			out.ClickIDStr = filter.UnsafeString(decoded)
		case telegramKeyBridgeToken:
			if !ValidateBridgeToken(decoded) {
				return scratch
			}
			out.BridgeToken = filter.UnsafeString(decoded)
		case telegramKeyPlacementID:
			out.PlacementID = filter.UnsafeString(decoded)
		case telegramKeyPremium:
			out.Premium = TelegramQueryFlagTrue(decoded)
		case telegramKeyMotivated:
			out.Motivated = TelegramQueryFlagTrue(decoded)
		case telegramKeySub1, telegramKeySub2, telegramKeySub3, telegramKeySub4, telegramKeySub5:
			out.Subs[kid-telegramKeySub1] = filter.UnsafeString(decoded)
		case telegramKeyDMR:
			out.DMR = ParseDmrQueryFlag(decoded)
		}
	}

	if out.CampaignID == uuid.Nil || out.ClickID == uuid.Nil {
		return scratch
	}
	out.Passthrough = scratch
	out.OK = true
	return scratch
}

type TelegramRedirectMacroID uint8

const (
	telegramRedirectMacroNone TelegramRedirectMacroID = iota
	telegramRedirectMacroClickID
	telegramRedirectMacroBridgeToken
	telegramRedirectMacroSub1
	telegramRedirectMacroSub2
	telegramRedirectMacroSub3
	telegramRedirectMacroSub4
	telegramRedirectMacroSub5
)

func DispatchTelegramRedirectMacro(base []byte, i int) (TelegramRedirectMacroID, int) {
	n := len(base)
	if n == 0 || i >= n || base[i] != '{' || i+1 >= n {
		return telegramRedirectMacroNone, i
	}
	_ = base[n-1]
	switch base[i+1] {
	case 'c':
		if i+telegramRedirectMacroClickLen <= n {
			_ = base[i+telegramRedirectMacroClickLen-1]
			if *(*[telegramRedirectMacroClickLen]byte)(unsafe.Pointer(&base[i])) == telegramMacroClick {
				return telegramRedirectMacroClickID, i + telegramRedirectMacroClickLen
			}
		}
	case 'b':
		if i+telegramRedirectMacroBridgeLen <= n {
			_ = base[i+telegramRedirectMacroBridgeLen-1]
			if *(*[telegramRedirectMacroBridgeLen]byte)(unsafe.Pointer(&base[i])) == telegramMacroBridge {
				return telegramRedirectMacroBridgeToken, i + telegramRedirectMacroBridgeLen
			}
		}
	case 's':
		if i+telegramRedirectMacroSubLen <= n {
			_ = base[i+telegramRedirectMacroSubLen-1]
			digit := base[i+4]
			if base[i+2] == 'u' && base[i+3] == 'b' && base[i+5] == '}' && digit >= '1' && digit <= '5' {
				want := [telegramRedirectMacroSubLen]byte{'{', 's', 'u', 'b', digit, '}'}
				if *(*[telegramRedirectMacroSubLen]byte)(unsafe.Pointer(&base[i])) == want {
					return telegramRedirectMacroSub1 + TelegramRedirectMacroID(digit-'1'), i + telegramRedirectMacroSubLen
				}
			}
		}
	}
	return telegramRedirectMacroNone, i
}

func ExpandTelegramRedirectMacros(dst, base []byte, ClickID, BridgeToken string, Subs [5]string) []byte {
	clickB := filter.UnsafeBytes(ClickID)
	bridgeB := filter.UnsafeBytes(BridgeToken)
	subB := [5][]byte{
		filter.UnsafeBytes(Subs[0]),
		filter.UnsafeBytes(Subs[1]),
		filter.UnsafeBytes(Subs[2]),
		filter.UnsafeBytes(Subs[3]),
		filter.UnsafeBytes(Subs[4]),
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
		mid, end := DispatchTelegramRedirectMacro(base, i)
		switch mid {
		case telegramRedirectMacroClickID:
			dst = AppendRedirectMacroEscaped(dst, clickB)
			i = end
		case telegramRedirectMacroBridgeToken:
			dst = AppendRedirectMacroEscaped(dst, bridgeB)
			i = end
		case telegramRedirectMacroSub1:
			dst = AppendRedirectMacroEscaped(dst, subB[0])
			i = end
		case telegramRedirectMacroSub2:
			dst = AppendRedirectMacroEscaped(dst, subB[1])
			i = end
		case telegramRedirectMacroSub3:
			dst = AppendRedirectMacroEscaped(dst, subB[2])
			i = end
		case telegramRedirectMacroSub4:
			dst = AppendRedirectMacroEscaped(dst, subB[3])
			i = end
		case telegramRedirectMacroSub5:
			dst = AppendRedirectMacroEscaped(dst, subB[4])
			i = end
		default:
			dst = append(dst, base[i])
			i++
		}
	}
	return dst
}

func BuildTelegramRedirectLocation(dst, base []byte, ClickID, BridgeToken string, Subs [5]string, Passthrough []byte) ([]byte, bool) {
	if !RedirectBaseValid(base) {
		return dst, false
	}
	dst = dst[:0]
	dst = ExpandTelegramRedirectMacros(dst, base, ClickID, BridgeToken, Subs)
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

func AppendTelegramClickLink(dst []byte, baseURL string, CampaignID, ClickID uuid.UUID, WidgetID []byte) []byte {
	dst = append(dst, baseURL...)
	sep := byte('?')
	for i := range len(dst) {
		if dst[i] == '?' {
			sep = '&'
			break
		}
	}
	dst = append(dst, sep)
	dst = append(dst, "campaign_id="...)
	dst = AppendUUIDStr(dst, CampaignID)
	dst = append(dst, "&click_id="...)
	dst = AppendUUIDStr(dst, ClickID)
	if len(WidgetID) > 0 {
		dst = append(dst, "&widget_id="...)
		dst = append(dst, WidgetID...)
	}
	return dst
}
