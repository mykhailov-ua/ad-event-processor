package ingest

import (
	"net/http"
	"strconv"
	"unicode/utf8"

	"ad-event-processor/internal/domain"

	"github.com/google/uuid"
	"github.com/panjf2000/gnet/v2"
)

const (
	dmrHTMLPrefix = "HTTP/1.1 200 OK\r\nContent-Type: text/html; charset=utf-8\r\n" +
		"Cache-Control: no-store\r\nConnection: keep-alive\r\nContent-Length: "
	dmrBodyPrefix = `<!DOCTYPE html><html lang="en"><head><meta charset="utf-8">` +
		`<meta http-equiv="refresh" content="0;url=`
	dmrBodyMid    = `"></head><body><script>window.location.replace("`
	dmrBodySuffix = `")</script></body></html>`
)

func dmrResponseWireLen(url []byte) int {
	urlExpand := len(url) * 7
	bodyLen := len(dmrBodyPrefix) + urlExpand + len(dmrBodyMid) + urlExpand + len(dmrBodySuffix)
	hdrLen := len(dmrHTMLPrefix) + 6 + len("\r\n\r\n")
	return hdrLen + bodyLen
}

func BuildDmrResponse(dst []byte, url []byte) []byte {
	var bodyBuf [8192]byte
	body := buildDmrBody(bodyBuf[:0], url)

	dst = append(dst, dmrHTMLPrefix...)
	dst = strconv.AppendInt(dst, int64(len(body)), 10)
	dst = append(dst, "\r\n\r\n"...)
	dst = append(dst, body...)
	return dst
}

func buildDmrBody(dst []byte, url []byte) []byte {
	dst = append(dst, dmrBodyPrefix...)
	dst = dmrHTMLAttrEscape(dst, url)
	dst = append(dst, dmrBodyMid...)
	dst = dmrJSStringEscape(dst, url)
	dst = append(dst, dmrBodySuffix...)
	return dst
}

func dmrHTMLAttrEscape(dst, src []byte) []byte {
	i := 0
	for i < len(src) {
		r, size := utf8.DecodeRune(src[i:])
		switch {
		case r == '&':
			dst = append(dst, "&amp;"...)
		case r == '"':
			dst = append(dst, "&quot;"...)
		case r == '\'':
			dst = append(dst, "&#39;"...)
		case r == '<':
			dst = append(dst, "&lt;"...)
		case r == '>':
			dst = append(dst, "&gt;"...)
		case r == '\r':
			dst = append(dst, "&#13;"...)
		case r == '\n':
			dst = append(dst, "&#10;"...)

		case r == 0x2028:
			dst = append(dst, "&#8232;"...)
		case r == 0x2029:
			dst = append(dst, "&#8233;"...)
		default:
			dst = append(dst, src[i:i+size]...)
		}
		i += size
	}
	return dst
}

func dmrJSStringEscape(dst, src []byte) []byte {
	i := 0
	for i < len(src) {
		r, size := utf8.DecodeRune(src[i:])
		switch {
		case r == '"':
			dst = append(dst, `\"`...)
		case r == '\'':
			dst = append(dst, `\'`...)
		case r == '\\':
			dst = append(dst, `\\`...)
		case r == '\r':
			dst = append(dst, `\r`...)
		case r == '\n':
			dst = append(dst, `\n`...)

		case r == '/':
			dst = append(dst, `\/`...)

		case r == 0x2028:
			dst = append(dst, `\u2028`...)
		case r == 0x2029:
			dst = append(dst, `\u2029`...)

		case r < 0x20:
			dst = append(dst, '\\', 'x')
			dst = append(dst, hexNibble(byte(r)>>4), hexNibble(byte(r)&0x0f))
		default:
			dst = append(dst, src[i:i+size]...)
		}
		i += size
	}
	return dst
}

func hexNibble(v byte) byte {
	if v < 10 {
		return '0' + v
	}
	return 'a' + v - 10
}

func (h *AdsPacketHandler) writeGnetClickDmrRedirect(ctx *ConnContext, c gnet.Conn, startMono int64, url []byte) {
	need := dmrResponseWireLen(url)
	buf := ctx.BufSlice
	if cap(buf) < need {
		buf = make([]byte, 0, need)
		ctx.BufSlice = buf
	}
	ctx.BufSlice = BuildDmrResponse(buf[:0], url)
	h.write(c, ctx.BufSlice, ctx)
	h.recordMetrics(startMono, http.StatusOK)
}

func clickDmrEnabled(dmr bool, camp *domain.Campaign) bool {
	if dmr {
		return true
	}
	return camp != nil && camp.DmrEnabled
}

func (h *AdsPacketHandler) clickDmrActive(campaignID uuid.UUID, dmr bool) bool {
	if dmr {
		return true
	}
	if h.registry == nil {
		return false
	}
	camp, ok := h.registry.GetCampaign(campaignID)
	if !ok || camp == nil {
		return false
	}
	return camp.DmrEnabled
}

func (h *AdsPacketHandler) writeGnetClickLandingRedirect(ctx *ConnContext, c gnet.Conn, startMono int64, loc []byte, dmrActive bool) {
	if dmrActive {
		h.writeGnetClickDmrRedirect(ctx, c, startMono, loc)
		return
	}
	h.writeGnetClickRedirect(ctx, c, startMono, loc)
}
