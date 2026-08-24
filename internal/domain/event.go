package domain

import (
	"context"
	"sync"
	"time"
	"unsafe"

	"github.com/google/uuid"
)

type contextKey string

const DeduplicationTokenKey contextKey = "dedup_token"

type Event struct {
	ClickID            string
	CampaignID         uuid.UUID
	UserID             string
	Type               string
	PlacementID        string
	Payload            []byte
	IP                 string
	UA                 string
	TLSHash            string
	TLSJA3             string
	TLSJA4             string
	SecCHUA            string
	AcceptLang         string
	FraudReason        string
	FraudScore         uint32
	SilentRejectEvent  bool
	ShadowEvent        bool
	CreatedAt          time.Time
	StringBuffer       []byte
	Scratch            unsafe.Pointer
	FilterDeadlineMono int64
	FilterWorkerIdx    int8
	IngestGeoResolved  bool
	GeoHash            uint32
	GeoCountry         string
	ClearingPriceMicro int64
	ClickIDBuf         [36]byte
	UserPIIHash        [16]byte
	HasUserPIIHash     bool
	TCPMSS             uint8
	TCPMSSSet          uint8
	TCPTTL             uint8
	TCPTTLSet          uint8
	TCPWindow          uint16
	TCPWindowSet       uint8
}

func (event *Event) Reset() {
	event.ClickID = ""
	event.CampaignID = uuid.Nil
	event.UserID = ""
	event.Type = ""
	event.PlacementID = ""
	if cap(event.Payload) > 4096 {
		event.Payload = make([]byte, 0, 1024)
	} else {
		event.Payload = event.Payload[:0]
	}
	event.IP = ""
	event.UA = ""
	event.TLSHash = ""
	event.TLSJA3 = ""
	event.TLSJA4 = ""
	event.SecCHUA = ""
	event.AcceptLang = ""
	event.FraudReason = ""
	event.FraudScore = 0
	event.SilentRejectEvent = false
	event.ShadowEvent = false
	event.CreatedAt = time.Time{}
	event.Scratch = nil
	event.FilterDeadlineMono = 0
	event.FilterWorkerIdx = -1
	event.IngestGeoResolved = false
	event.GeoHash = 0
	event.GeoCountry = ""
	event.ClearingPriceMicro = 0
	event.HasUserPIIHash = false
	event.TCPMSS = 0
	event.TCPMSSSet = 0
	event.TCPTTL = 0
	event.TCPTTLSet = 0
	event.TCPWindow = 0
	event.TCPWindowSet = 0
	if cap(event.StringBuffer) > 2048 {
		event.StringBuffer = make([]byte, 0, 256)
	} else {
		event.StringBuffer = event.StringBuffer[:0]
	}
}

var EventPool = sync.Pool{
	New: func() any {
		return &Event{
			Payload:      make([]byte, 0, 1024),
			StringBuffer: make([]byte, 0, 256),
		}
	},
}

type EventStore interface {
	StoreBatch(ctx context.Context, events []*Event) error
	Close() error
}
