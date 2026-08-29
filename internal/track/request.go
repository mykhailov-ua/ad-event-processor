package track

import (
	"ad-event-processor/internal/domain"

	"github.com/google/uuid"
)

type Request struct {
	CampaignID             uuid.UUID
	UserID                 string
	Type                   string
	ClickID                string
	PlacementID            string
	Payload                []byte
	Subs                   SubIDSlots
	FBCLID                 string
	GCLID                  string
	TTCLID                 string
	MSCLKID                string
	TBLCI                  string
	OBClickID              string
	EventID                string
	TxID                   string
	JSONSerializationFlags uint8
	TelemetrySet           uint8
	TelemetryEvents        []domain.BehaviorTelemetryEvent
}

func (v *Request) ResetForParse() {
	v.CampaignID = uuid.Nil
	v.UserID = ""
	v.Type = ""
	v.ClickID = ""
	v.PlacementID = ""
	v.Subs.Reset()
	v.FBCLID = ""
	v.GCLID = ""
	v.TTCLID = ""
	v.MSCLKID = ""
	v.TBLCI = ""
	v.OBClickID = ""
	v.EventID = ""
	v.TxID = ""
	v.JSONSerializationFlags = 0
	v.TelemetrySet = 0
	v.TelemetryEvents = v.TelemetryEvents[:0]
}

func (v *Request) ResetPayload() {
	v.Payload = nil
}
