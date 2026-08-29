package ingest

import (
	"ad-event-processor/internal/domain"
	"ad-event-processor/internal/ingest/parser"

	"github.com/google/uuid"
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
	v.subs.Reset()
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

func ParseTrackRequestJSON(v *TrackRequest, data []byte) error {
	return parseTrackRequestJSON(v, data)
}

func ParseTrackRequestJSONOpt(v *TrackRequest, data []byte) error {
	return parseTrackRequestJSON(v, data)
}

func parseTrackRequestJSON(v *TrackRequest, data []byte) error {
	v.resetForParse()
	if len(data) == 0 {
		return parser.ErrMalformed
	}
	_ = data[len(data)-1]

	n := len(data)
	bud := newJSONScanBudget()
	i, ok := skipJSONWSBudget(data, 0, n, &bud)
	if !ok || i >= n || data[i] != '{' {
		return parser.ErrMalformed
	}
	i++

	for i < n {
		i, ok = skipJSONWSBudget(data, i, n, &bud)
		if !ok || i >= n {
			return parser.ErrMalformed
		}
		if data[i] == '}' {
			return nil
		}
		if data[i] != '"' {
			return parser.ErrMalformed
		}
		i++

		keyStart := i
		for i < n && data[i] != '"' {
			if data[i] == '\\' {
				return parser.ErrMalformed
			}
			i++
		}
		if i >= n {
			return parser.ErrMalformed
		}
		keyEnd := i
		i++
		if !jsonTrackKeyOK(data[keyStart:keyEnd]) {
			return parser.ErrMalformed
		}

		i, ok = skipJSONWSBudget(data, i, n, &bud)
		if !ok || i >= n || data[i] != ':' {
			return parser.ErrMalformed
		}
		i++

		i, ok = skipJSONWSBudget(data, i, n, &bud)
		if !ok || i >= n {
			return parser.ErrMalformed
		}

		kid := parser.MatchTrackKey(data[keyStart:keyEnd])
		keyBytes := data[keyStart:keyEnd]
		switch kid {
		case parser.KeyType, parser.KeyUserID, parser.KeyClickID, parser.KeyPlacementID,
			parser.KeyFBCLID, parser.KeyGCLID, parser.KeyTTCLID, parser.KeyMSCLKID,
			parser.KeyTBLCI, parser.KeyOBClickID, parser.KeyEventID, parser.KeyTxID:
			if data[i] != '"' {
				return parser.ErrMalformed
			}
			valStart := i + 1
			end, ok := scanJSONStringEnd(data, i, n, &bud)
			if !ok {
				return parser.ErrMalformed
			}
			valBytes := data[valStart : end-1]
			i = end
			assignTrackStringField(v, kid, valBytes)
		case parser.KeyCampaignID:
			if data[i] != '"' {
				return parser.ErrMalformed
			}
			valStart := i + 1
			end, ok := scanJSONLiteralStringEnd(data, i, n, &bud)
			if !ok {
				return parser.ErrMalformed
			}
			if !ParseUUID(data[valStart:end-1], &v.CampaignID) {
				return parser.ErrMalformed
			}
			i = end
		case parser.KeyPayload:
			valStart := i
			valEnd, err := skipJSONValueBudget(data, i, &bud)
			if err != nil {
				return err
			}
			v.Payload = data[valStart:valEnd]
			i = valEnd
		default:
			if idx, ok := subKeyIndex(keyBytes); ok {
				if data[i] != '"' {
					return parser.ErrMalformed
				}
				valStart := i + 1
				end, ok := scanJSONStringEnd(data, i, n, &bud)
				if !ok {
					return parser.ErrMalformed
				}
				v.subs[idx-1] = unsafeString(data[valStart : end-1])
				i = end
			} else if matchTelemetryKey(keyBytes) {
				end, events, ok := parseTrackTelemetryValue(data, i, n, &bud, v.TelemetryEvents)
				if !ok {
					return parser.ErrMalformed
				}
				v.TelemetryEvents = events
				v.TelemetrySet = 1
				i = end
			} else {
				valEnd, err := skipJSONValueBudget(data, i, &bud)
				if err != nil {
					return err
				}
				i = valEnd
			}
		}

		if !bud.consumeKeyPair() {
			return parser.ErrMalformed
		}

		i, ok = skipJSONWSBudget(data, i, n, &bud)
		if !ok || i >= n {
			return parser.ErrMalformed
		}
		switch data[i] {
		case ',':
			i++
		case '}':
			return nil
		default:
			return parser.ErrMalformed
		}
	}

	return parser.ErrMalformed
}

func assignTrackStringField(v *TrackRequest, kid parser.KeyID, valBytes []byte) {
	switch kid {
	case parser.KeyType:
		v.Type = unsafeString(valBytes)
	case parser.KeyUserID:
		v.UserID = unsafeString(valBytes)
	case parser.KeyClickID:
		v.ClickID = unsafeString(valBytes)
	case parser.KeyPlacementID:
		v.PlacementID = unsafeString(valBytes)
	case parser.KeyFBCLID:
		v.fbclid = unsafeString(valBytes)
	case parser.KeyGCLID:
		v.gclid = unsafeString(valBytes)
	case parser.KeyTTCLID:
		v.ttclid = unsafeString(valBytes)
	case parser.KeyMSCLKID:
		v.msclkid = unsafeString(valBytes)
	case parser.KeyTBLCI:
		v.tblci = unsafeString(valBytes)
	case parser.KeyOBClickID:
		v.obClickID = unsafeString(valBytes)
	case parser.KeyEventID:
		v.eventID = unsafeString(valBytes)
	case parser.KeyTxID:
		v.txID = unsafeString(valBytes)
	}
}
