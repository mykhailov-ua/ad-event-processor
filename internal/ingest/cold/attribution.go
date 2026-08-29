package cold

import (
	"ad-event-processor/internal/ingest/parser"
	"ad-event-processor/internal/track"

	"github.com/google/uuid"
)

func AppendAttributionPayload(dst, payload []byte, subs track.SubIDSlots, fbclid, gclid, ttclid, msclkid, tblci, obClickID, eventID, txID string) []byte {
	dst = dst[:0]
	switch {
	case len(payload) > 0 && payload[0] == '{':
		if len(payload) > 1 && payload[len(payload)-1] == '}' {
			dst = append(dst, payload[:len(payload)-1]...)
		} else {
			dst = append(dst, payload...)
		}
	case len(payload) > 0:
		dst = append(dst, '{')
		dst = appendJSONKeyString(dst, "payload", payload)
	default:
		dst = append(dst, '{')
	}

	for i := range track.MaxSubIDs {
		if subs[i] == "" {
			continue
		}
		if len(dst) > 1 {
			dst = append(dst, ',')
		}
		dst = appendSubKey(dst, i+1)
		dst = append(dst, ':')
		dst = parser.AppendJSONString(dst, unsafeBytes(subs[i]))
	}
	if fbclid != "" {
		if len(dst) > 1 {
			dst = append(dst, ',')
		}
		dst = append(dst, `"fbclid":`...)
		dst = parser.AppendJSONString(dst, unsafeBytes(fbclid))
	}
	if gclid != "" {
		if len(dst) > 1 {
			dst = append(dst, ',')
		}
		dst = append(dst, `"gclid":`...)
		dst = parser.AppendJSONString(dst, unsafeBytes(gclid))
	}
	if ttclid != "" {
		if len(dst) > 1 {
			dst = append(dst, ',')
		}
		dst = append(dst, `"ttclid":`...)
		dst = parser.AppendJSONString(dst, unsafeBytes(ttclid))
	}
	if msclkid != "" {
		if len(dst) > 1 {
			dst = append(dst, ',')
		}
		dst = append(dst, `"msclkid":`...)
		dst = parser.AppendJSONString(dst, unsafeBytes(msclkid))
	}
	if tblci != "" {
		if len(dst) > 1 {
			dst = append(dst, ',')
		}
		dst = append(dst, `"tblci":`...)
		dst = parser.AppendJSONString(dst, unsafeBytes(tblci))
	}
	if obClickID != "" {
		if len(dst) > 1 {
			dst = append(dst, ',')
		}
		dst = append(dst, `"ob_click_id":`...)
		dst = parser.AppendJSONString(dst, unsafeBytes(obClickID))
	}
	if eventID != "" {
		if len(dst) > 1 {
			dst = append(dst, ',')
		}
		dst = append(dst, `"event_id":`...)
		dst = parser.AppendJSONString(dst, unsafeBytes(eventID))
	}
	if txID != "" {
		if len(dst) > 1 {
			dst = append(dst, ',')
		}
		dst = append(dst, `"tx_id":`...)
		dst = parser.AppendJSONString(dst, unsafeBytes(txID))
	}
	if len(dst) == 1 {
		return dst[:0]
	}
	dst = append(dst, '}')
	return dst
}

func appendSubKey(dst []byte, idx int) []byte {
	dst = append(dst, '"', 's', 'u', 'b')
	if idx < 10 {
		return append(dst, byte('0'+idx), '"')
	}
	return append(dst, byte('0'+idx/10), byte('0'+idx%10), '"')
}

func AppendAttributionPassthrough(dst []byte, fbclid, gclid, ttclid string) []byte {
	if fbclid != "" {
		if len(dst) > 0 {
			dst = append(dst, '&')
		}
		dst = append(dst, "fbclid="...)
		dst = append(dst, fbclid...)
	}
	if gclid != "" {
		if len(dst) > 0 {
			dst = append(dst, '&')
		}
		dst = append(dst, "gclid="...)
		dst = append(dst, gclid...)
	}
	if ttclid != "" {
		if len(dst) > 0 {
			dst = append(dst, '&')
		}
		dst = append(dst, "ttclid="...)
		dst = append(dst, ttclid...)
	}
	return dst
}

func AppendFlowAttribution(dst []byte, landerID, offerID uuid.UUID) []byte {
	if landerID == uuid.Nil && offerID == uuid.Nil {
		return dst
	}
	switch {
	case len(dst) == 0:
		dst = append(dst, '{')
	case dst[0] != '{':
		dst = append(dst, '{')
	case len(dst) > 1 && dst[len(dst)-1] == '}':
		dst = dst[:len(dst)-1]
	default:
		dst = append(dst, '{')
	}
	if landerID != uuid.Nil {
		if len(dst) > 1 {
			dst = append(dst, ',')
		}
		dst = append(dst, `"lander_id":"`...)
		dst = append(dst, landerID.String()...)
		dst = append(dst, '"')
	}
	if offerID != uuid.Nil {
		if len(dst) > 1 {
			dst = append(dst, ',')
		}
		dst = append(dst, `"offer_id":"`...)
		dst = append(dst, offerID.String()...)
		dst = append(dst, '"')
	}
	if len(dst) == 1 {
		return dst[:0]
	}
	dst = append(dst, '}')
	return dst
}

func appendJSONKeyString(dst []byte, key string, val []byte) []byte {
	dst = append(dst, '"')
	dst = append(dst, key...)
	dst = append(dst, '"', ':')
	return parser.AppendJSONString(dst, val)
}
