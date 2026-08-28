package ingestion

import (
	"github.com/google/uuid"

	"ad-event-processor/internal/domain"
)

type TrackRequest struct {
	CampaignID             uuid.UUID
	UserID                 string
	Type                   string
	ClickID                string
	PlacementID            string
	Payload                []byte
	subs                   SubIDSlots
	fbclid                 string
	gclid                  string
	ttclid                 string
	msclkid                string
	tblci                  string
	obClickID              string
	eventID                string
	txID                   string
	ortbSlot               *openRTBScratchSlot
	JSONSerializationFlags uint8
	TelemetrySet           uint8
	TelemetryEvents        []domain.BehaviorTelemetryEvent
}

func (v *TrackRequest) Reset() {
	v.resetForParse()
	v.Payload = nil
	if v.ortbSlot != nil {
		releaseOpenRTBScratchSlot(v.ortbSlot)
		v.ortbSlot = nil
	}
}

func (v *TrackRequest) resetForParse() {
	v.CampaignID = uuid.Nil
	v.UserID = ""
	v.Type = ""
	v.ClickID = ""
	v.PlacementID = ""
	v.subs.reset()
	v.fbclid = ""
	v.gclid = ""
	v.ttclid = ""
	v.msclkid = ""
	v.tblci = ""
	v.obClickID = ""
	v.eventID = ""
	v.txID = ""
	v.JSONSerializationFlags = 0
	v.TelemetrySet = 0
	v.TelemetryEvents = v.TelemetryEvents[:0]
}

func (v *TrackRequest) UnmarshalJSON(data []byte) error {
	return ParseTrackRequestJSON(v, data)
}

func appendJSONString(dst []byte, s []byte) []byte {
	dst = append(dst, '"')
	for _, b := range s {
		switch b {
		case '"':
			dst = append(dst, '\\', '"')
		case '\\':
			dst = append(dst, '\\', '\\')
		case '\n':
			dst = append(dst, '\\', 'n')
		case '\r':
			dst = append(dst, '\\', 'r')
		case '\t':
			dst = append(dst, '\\', 't')
		default:
			dst = append(dst, b)
		}
	}
	dst = append(dst, '"')
	return dst
}

func marshalExtra(dst []byte, keys, values [][]byte) []byte {
	dst = dst[:0]
	dst = append(dst, '{')
	for i, key := range keys {
		if i > 0 {
			dst = append(dst, ',')
		}
		dst = appendJSONString(dst, key)
		dst = append(dst, ':')
		if i < len(values) {
			dst = appendJSONString(dst, values[i])
		} else {
			dst = append(dst, '"', '"')
		}
	}
	dst = append(dst, '}')
	return dst
}
