package ingestion

import (
	"bytes"
	"encoding/hex"
	"net/http"

	"espx/internal/telemetry"

	"github.com/google/uuid"
	"github.com/panjf2000/gnet/v2"
)

type tgBidRequest struct {
	ip          []byte
	ua          []byte
	publisherID []byte
	telegramID  []byte
	widgetID    []byte
	bidFloor    float64
	premium     bool
	motivated   bool
	width       int32
	height      int32
	production  bool
}

func parseAsciiInt(b []byte) int {
	res := 0
	for i := 0; i < len(b); i++ {
		c := b[i]
		if c >= '0' && c <= '9' {
			res = res*10 + int(c-'0')
		}
	}
	return res
}

func parseAsciiFloat(b []byte) float64 {
	res := 0.0
	dec := -1.0
	for i := 0; i < len(b); i++ {
		c := b[i]
		if c >= '0' && c <= '9' {
			if dec < 0 {
				res = res*10.0 + float64(c-'0')
			} else {
				res = res + float64(c-'0')*dec
				dec *= 0.1
			}
		} else if c == '.' {
			dec = 0.1
		}
	}
	return res
}

func parseTgBidRequest(body []byte, out *tgBidRequest) bool {
	*out = tgBidRequest{}
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
			if bytes.Equal(key, []byte("ip")) {
				out.ip = val
			} else if bytes.Equal(key, []byte("user_agent")) || bytes.Equal(key, []byte("ua")) {
				out.ua = val
			} else if bytes.Equal(key, []byte("publisher_id")) {
				out.publisherID = val
			} else if bytes.Equal(key, []byte("telegram_id")) {
				out.telegramID = val
			} else if bytes.Equal(key, []byte("widget_id")) {
				out.widgetID = val
			}
		} else {
			valStart := i
			for i < len(body) && body[i] != ',' && body[i] != '}' && body[i] != ']' && body[i] != ' ' && body[i] != '\n' && body[i] != '\r' && body[i] != '\t' {
				i++
			}
			val := body[valStart:i]
			if bytes.Equal(key, []byte("premium")) {
				out.premium = bytes.Equal(val, []byte("true"))
			} else if bytes.Equal(key, []byte("motivated")) {
				out.motivated = bytes.Equal(val, []byte("true"))
			} else if bytes.Equal(key, []byte("production")) {
				out.production = bytes.Equal(val, []byte("true"))
			} else if bytes.Equal(key, []byte("bid_floor")) {
				out.bidFloor = parseAsciiFloat(val)
			} else if bytes.Equal(key, []byte("width")) {
				out.width = int32(parseAsciiInt(val))
			} else if bytes.Equal(key, []byte("height")) {
				out.height = int32(parseAsciiInt(val))
			}
		}
	}
	return len(out.publisherID) > 0
}

func appendUintStr(dst []byte, v uint64) []byte {
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

func appendFloatStr(dst []byte, f float64) []byte {
	if f == 0.0 {
		return append(dst, "0.0"...)
	}
	whole := int64(f)
	dst = appendUintStr(dst, uint64(whole))
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

func appendUUIDStr(dst []byte, u uuid.UUID) []byte {
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

func (h *AdsPacketHandler) reactTgBid(req parsedHTTPRequest, c gnet.Conn, ctx *connContext) gnet.Action {
	startMono := monotonicNano()
	telemetry.RecordTrack()

	var parsedReq tgBidRequest
	if !parseTgBidRequest(req.Body, &parsedReq) {
		h.write(c, respTg400, ctx)
		h.recordMetrics(startMono, http.StatusBadRequest)
		return gnet.None
	}

	clientIP := unsafeString(parsedReq.ip)
	evt := &ctx.evt
	evt.Reset()
	evt.IP = clientIP
	ensureIngestGeo(h.trackProc.ingestGeo, evt)

	targeting := RtbTargetingInput{
		PublisherFloorMicro: int64(parsedReq.bidFloor * 1000000),
		GeoHash:             evt.GeoHash,
	}

	res, reason := h.trackProc.rtbCatalog.RunAuction(evt, targeting)
	if !reason.OK() {
		h.write(c, respTg204, ctx)
		h.recordMetrics(startMono, http.StatusNoContent)
		return gnet.None
	}

	uid, ok := h.trackProc.rtbCatalog.UUIDForWinner(res.CampaignID)
	if !ok {
		h.write(c, respTg204, ctx)
		h.recordMetrics(startMono, http.StatusNoContent)
		return gnet.None
	}

	clickID := NewFastUUID()

	body := ctx.extraBuf[:0]
	body = append(body, `[{"creative_id":`...)
	body = appendUintStr(body, uint64(res.CreativeID))
	body = append(body, `,"campaign_id":"`...)
	body = appendUUIDStr(body, uid)
	body = append(body, `","price":`...)
	body = appendFloatStr(body, float64(res.Price)/1000000.0)
	body = append(body, `,"link":"`...)
	body = append(body, `http://track.local/tg/click?campaign_id=`...)
	body = appendUUIDStr(body, uid)
	body = append(body, `&click_id=`...)
	body = appendUUIDStr(body, clickID)
	if len(parsedReq.widgetID) > 0 {
		body = append(body, `&widget_id=`...)
		body = append(body, parsedReq.widgetID...)
	}
	body = append(body, `","width":`...)
	body = appendUintStr(body, uint64(parsedReq.width))
	body = append(body, `,"height":`...)
	body = appendUintStr(body, uint64(parsedReq.height))
	body = append(body, `}]`...)

	wire := ctx.bufSlice[:0]
	wire = append(wire, "HTTP/1.1 200 OK\r\nContent-Type: application/json\r\nContent-Length: "...)
	wire = appendUintStr(wire, uint64(len(body)))
	wire = append(wire, "\r\nConnection: keep-alive\r\n\r\n"...)
	wire = append(wire, body...)

	ctx.extraBuf = body
	ctx.bufSlice = wire

	h.write(c, wire, ctx)
	h.recordMetrics(startMono, http.StatusOK)
	return gnet.None
}
