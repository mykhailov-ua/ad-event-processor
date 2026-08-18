package ingestion

import (
	"net/http"

	"github.com/bidshard/ad-event-processor/internal/domain"
	"github.com/google/uuid"
	"github.com/panjf2000/gnet/v2"
)

var (
	dmrHTTPPrefix = []byte("HTTP/1.1 200 OK\r\nContent-Type: text/html; charset=utf-8\r\nCache-Control: no-store\r\nConnection: keep-alive\r\nContent-Length: ")
	dmrHTTPMiddle = []byte("\r\n\r\n")
	dmrHTMLPrefix = []byte("<!DOCTYPE html><html><head><meta http-equiv=\"refresh\" content=\"0;url=")
	dmrHTMLMid    = []byte("\"><script>window.location.replace(\"")
	dmrHTMLSuffix = []byte("\")</script></head><body></body></html>")
)

func BuildDmrResponse(dst []byte, url []byte) []byte {
	htmlEscapedLen := dmrHTMLEscapeLen(url)
	jsEscapedLen := dmrJsEscapeLen(url)

	bodyLen := len(dmrHTMLPrefix) + htmlEscapedLen + len(dmrHTMLMid) + jsEscapedLen + len(dmrHTMLSuffix)

	var lengthBuf [16]byte
	lengthStr := dmrFormatInt(lengthBuf[:], int64(bodyLen))

	totalLen := len(dmrHTTPPrefix) + len(lengthStr) + len(dmrHTTPMiddle) + bodyLen

	dst = dmrGrow(dst, totalLen)
	originalLen := len(dst)
	dst = dst[:originalLen+totalLen]
	writeBuf := dst[originalLen:]

	offset := copy(writeBuf, dmrHTTPPrefix)
	offset += copy(writeBuf[offset:], lengthStr)
	offset += copy(writeBuf[offset:], dmrHTTPMiddle)

	offset += copy(writeBuf[offset:], dmrHTMLPrefix)
	offset += dmrWriteHTMLEscaped(writeBuf[offset:], url)
	offset += copy(writeBuf[offset:], dmrHTMLMid)
	offset += dmrWriteJsEscaped(writeBuf[offset:], url)
	copy(writeBuf[offset:], dmrHTMLSuffix)

	return dst
}

func dmrHTMLUnicodeLineSepLen(url []byte, i, n int) (int, int) {
	if i+2 >= n || url[i] != 0xe2 || url[i+1] != 0x80 {
		return 0, 0
	}
	switch url[i+2] {
	case 0xa8:
		return 7, 3
	case 0xa9:
		return 7, 3
	default:
		return 0, 0
	}
}

func dmrHTMLControlEntityLen(c byte) int {
	if c < 10 {
		return 4
	}
	return 5
}

func dmrHTMLEscapeLen(url []byte) int {
	n := len(url)
	if n == 0 {
		return 0
	}
	length := 0
	for i := 0; i < n; i++ {
		if extra, skip := dmrHTMLUnicodeLineSepLen(url, i, n); skip > 0 {
			length += extra
			i += skip - 1
			continue
		}
		c := url[i]
		switch c {
		case '"':
			length += 6
		case '&':
			length += 5
		case '\'':
			length += 5
		case '<':
			length += 4
		case '>':
			length += 4
		default:
			if c < 0x20 {
				length += dmrHTMLControlEntityLen(c)
			} else {
				length++
			}
		}
	}
	return length
}

func dmrWriteHTMLEscaped(dst, url []byte) int {
	n := len(url)
	if n == 0 {
		return 0
	}
	_ = dst[dmrHTMLEscapeLen(url)-1]
	_ = url[n-1]

	offset := 0
	for i := 0; i < n; i++ {
		if extra, skip := dmrHTMLUnicodeLineSepLen(url, i, n); skip > 0 {
			if url[i+2] == 0xa8 {
				copy(dst[offset:], "&#8232;")
			} else {
				copy(dst[offset:], "&#8233;")
			}
			offset += extra
			i += skip - 1
			continue
		}
		c := url[i]
		switch c {
		case '"':
			copy(dst[offset:], "&quot;")
			offset += 6
		case '&':
			copy(dst[offset:], "&amp;")
			offset += 5
		case '\'':
			copy(dst[offset:], "&#39;")
			offset += 5
		case '<':
			copy(dst[offset:], "&lt;")
			offset += 4
		case '>':
			copy(dst[offset:], "&gt;")
			offset += 4
		default:
			if c < 0x20 {
				if c < 10 {
					dst[offset] = '&'
					dst[offset+1] = '#'
					dst[offset+2] = '0' + c
					dst[offset+3] = ';'
					offset += 4
				} else {
					dst[offset] = '&'
					dst[offset+1] = '#'
					dst[offset+2] = '0' + c/10
					dst[offset+3] = '0' + c%10
					dst[offset+4] = ';'
					offset += 5
				}
			} else {
				dst[offset] = c
				offset++
			}
		}
	}
	return offset
}

func dmrJsUnicodeLineSepLen(url []byte, i, n int) (int, int) {
	if i+2 >= n || url[i] != 0xe2 || url[i+1] != 0x80 {
		return 0, 0
	}
	switch url[i+2] {
	case 0xa8, 0xa9:
		return 6, 3
	default:
		return 0, 0
	}
}

func dmrJsEscapeLen(url []byte) int {
	n := len(url)
	if n == 0 {
		return 0
	}
	length := 0
	for i := 0; i < n; i++ {
		if extra, skip := dmrJsUnicodeLineSepLen(url, i, n); skip > 0 {
			length += extra
			i += skip - 1
			continue
		}
		c := url[i]
		switch c {
		case '\\', '"', '\'':
			length += 2
		case '\n', '\r':
			length += 2
		case '/':
			length += 2
		case '<', '>':
			length += 4
		default:
			length++
		}
	}
	return length
}

func dmrWriteJsEscaped(dst, url []byte) int {
	n := len(url)
	if n == 0 {
		return 0
	}
	_ = dst[dmrJsEscapeLen(url)-1]
	_ = url[n-1]

	offset := 0
	for i := 0; i < n; i++ {
		if extra, skip := dmrJsUnicodeLineSepLen(url, i, n); skip > 0 {
			if url[i+2] == 0xa8 {
				copy(dst[offset:], `\u2028`)
			} else {
				copy(dst[offset:], `\u2029`)
			}
			offset += extra
			i += skip - 1
			continue
		}
		c := url[i]
		switch c {
		case '\\':
			dst[offset] = '\\'
			dst[offset+1] = '\\'
			offset += 2
		case '"':
			dst[offset] = '\\'
			dst[offset+1] = '"'
			offset += 2
		case '\'':
			dst[offset] = '\\'
			dst[offset+1] = '\''
			offset += 2
		case '/':
			dst[offset] = '\\'
			dst[offset+1] = '/'
			offset += 2
		case '\n':
			dst[offset] = '\\'
			dst[offset+1] = 'n'
			offset += 2
		case '\r':
			dst[offset] = '\\'
			dst[offset+1] = 'r'
			offset += 2
		case '<':
			dst[offset] = '\\'
			dst[offset+1] = 'x'
			dst[offset+2] = '3'
			dst[offset+3] = 'c'
			offset += 4
		case '>':
			dst[offset] = '\\'
			dst[offset+1] = 'x'
			dst[offset+2] = '3'
			dst[offset+3] = 'e'
			offset += 4
		default:
			dst[offset] = c
			offset++
		}
	}
	return offset
}

func dmrFormatInt(buf []byte, v int64) []byte {
	if v == 0 {
		buf[0] = '0'
		return buf[:1]
	}
	i := len(buf)
	for v > 0 {
		i--
		buf[i] = byte('0' + (v % 10))
		v /= 10
	}
	return buf[i:]
}

func dmrGrow(buf []byte, extra int) []byte {
	n := len(buf)
	if cap(buf)-n >= extra {
		return buf
	}
	newCap := cap(buf) * 2
	if newCap < n+extra {
		newCap = n + extra
	}
	newBuf := make([]byte, n, newCap)
	copy(newBuf, buf)
	return newBuf
}

func dmrResponseWireLen(url []byte) int {
	htmlEscapedLen := dmrHTMLEscapeLen(url)
	jsEscapedLen := dmrJsEscapeLen(url)
	bodyLen := len(dmrHTMLPrefix) + htmlEscapedLen + len(dmrHTMLMid) + jsEscapedLen + len(dmrHTMLSuffix)

	var lengthBuf [16]byte
	lengthStr := dmrFormatInt(lengthBuf[:], int64(bodyLen))

	return len(dmrHTTPPrefix) + len(lengthStr) + len(dmrHTTPMiddle) + bodyLen
}

func parseDmrQueryFlag(decoded []byte) bool {
	n := len(decoded)
	if n == 1 && decoded[0] == '1' {
		return true
	}
	if n != 4 {
		return false
	}
	return (decoded[0] == 't' || decoded[0] == 'T') &&
		(decoded[1] == 'r' || decoded[1] == 'R') &&
		(decoded[2] == 'u' || decoded[2] == 'U') &&
		(decoded[3] == 'e' || decoded[3] == 'E')
}

func clickDmrEnabled(queryFlag bool, camp *domain.Campaign) bool {
	if queryFlag {
		return true
	}
	return camp != nil && camp.DmrEnabled
}

func (h *AdsPacketHandler) clickDmrActive(campaignID uuid.UUID, queryFlag bool) bool {
	var camp *domain.Campaign
	if h != nil && h.registry != nil {
		if c, ok := h.registry.GetCampaign(campaignID); ok {
			camp = c
		}
	}
	return clickDmrEnabled(queryFlag, camp)
}

func (h *AdsPacketHandler) writeGnetClickLandingRedirect(ctx *connContext, c gnet.Conn, startMono int64, location []byte, dmr bool) {
	if dmr {
		h.writeGnetClickDmrRedirect(ctx, c, startMono, location)
		return
	}
	h.writeGnetClickRedirect(ctx, c, startMono, location)
}

func (h *AdsPacketHandler) writeGnetClickDmrRedirect(ctx *connContext, c gnet.Conn, startMono int64, location []byte) {
	need := dmrResponseWireLen(location)
	buf := ctx.bufSlice
	if cap(buf) < need || dmrLocationUsesBuf(buf, location) {
		newCap := need
		if newCap < 4096 {
			newCap = 4096
		}
		buf = make([]byte, 0, newCap)
	} else {
		buf = buf[:0]
	}
	buf = BuildDmrResponse(buf, location)
	ctx.bufSlice = buf
	h.write(c, buf, ctx)
	h.recordMetrics(startMono, http.StatusOK)
}

func dmrLocationUsesBuf(buf, location []byte) bool {
	if cap(buf) == 0 || len(location) == 0 {
		return false
	}
	if len(buf) > 0 && &buf[0] == &location[0] {
		return true
	}
	if len(buf) == 0 {
		backing := buf[:cap(buf)]
		if len(backing) >= len(location) {
			return &backing[0] == &location[0]
		}
	}
	return false
}
