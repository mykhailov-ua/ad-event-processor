package track

import (
	"ad-event-processor/internal/domain"
	"ad-event-processor/internal/filter"

	"github.com/google/uuid"
)

type WireIngress struct {
	ClientIP    []byte
	UserAgent   []byte
	TLSHash     []byte
	TLSJA3      []byte
	TLSJA4      []byte
	SecCHUA     []byte
	AcceptLang  []byte
	ProtoH2     bool
	WireMeta    func(evt *domain.Event)
	FillIngress func(evt *domain.Event, protoH2 bool)
}

func FillTelegramEventFromParsed(
	evt *domain.Event,
	eventType string,
	parsed *TelegramQueryParsed,
	req WireIngress,
) {
	evt.Reset()
	evt.ClickID = parsed.ClickIDStr
	evt.CampaignID = parsed.CampaignID
	evt.Type = eventType
	evt.PlacementID = parsed.PlacementID
	if len(req.ClientIP) > 0 {
		evt.IP = filter.UnsafeString(req.ClientIP)
	}
	if len(req.UserAgent) > 0 {
		evt.UA = filter.UnsafeString(req.UserAgent)
	}
	evt.TLSHash = filter.UnsafeString(req.TLSHash)
	evt.TLSJA3 = filter.UnsafeString(req.TLSJA3)
	evt.TLSJA4 = filter.UnsafeString(req.TLSJA4)
	evt.SecCHUA = filter.UnsafeString(req.SecCHUA)
	evt.AcceptLang = filter.UnsafeString(req.AcceptLang)
	if req.FillIngress != nil {
		req.FillIngress(evt, req.ProtoH2)
	}
	if req.WireMeta != nil {
		req.WireMeta(evt)
	}
	evt.Payload = MarshalTelegramBridgePayload(evt.Payload, parsed.BridgeToken)
}

type TelegramBidSeat struct {
	CreativeID uint64
	CampaignID uuid.UUID
	PriceMicro int64
}

func BuildTelegramBidJSON(
	dst []byte,
	seat TelegramBidSeat,
	baseURL string,
	clickID uuid.UUID,
	width, height int32,
	widgetID []byte,
) []byte {
	dst = append(dst, `[{"creative_id":`...)
	dst = AppendUintStr(dst, seat.CreativeID)
	dst = append(dst, `,"campaign_id":"`...)
	dst = AppendUUIDStr(dst, seat.CampaignID)
	dst = append(dst, `","price":`...)
	dst = AppendFloatStr(dst, float64(seat.PriceMicro)/1000000.0)
	dst = append(dst, `,"link":"`...)
	dst = AppendTelegramClickLink(dst, baseURL, seat.CampaignID, clickID, widgetID)
	dst = append(dst, `","width":`...)
	dst = AppendUintStr(dst, uint64(width))
	dst = append(dst, `,"height":`...)
	dst = AppendUintStr(dst, uint64(height))
	dst = append(dst, `}]`...)
	return dst
}

func BuildTelegramBidWire(dst []byte, body []byte) []byte {
	dst = append(dst, "HTTP/1.1 200 OK\r\nContent-Type: application/json\r\nContent-Length: "...)
	dst = AppendUintStr(dst, uint64(len(body)))
	dst = append(dst, "\r\nConnection: keep-alive\r\n\r\n"...)
	dst = append(dst, body...)
	return dst
}
