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

type BehaviorTelemetryEvent struct {
	T  string
	TS int64
	X  int
	Y  int
	Z  int
}

type Event struct {
	ClickID                string
	CampaignID             uuid.UUID
	UserID                 string
	Type                   string
	PlacementID            string
	Payload                []byte
	IP                     string
	UA                     string
	TLSHash                string
	TLSJA3                 string
	TLSJA4                 string
	SecCHUA                string
	SecCHUAPlatform        string
	TLSALPN                string
	AcceptLang             string
	SecFetchPresent        uint8
	SecFetchSite           uint8
	SecFetchMode           uint8
	SecFetchDest           uint8
	SecCHUAMobile          uint8
	IngressH2              uint8
	HTTP1HeaderOrder       [16]uint8
	HTTP1HeaderOrderCount  uint8
	AcceptEncodingFlags    uint8
	AcceptEncodingSet      uint8
	H2WireFlags            uint8
	H2SettingsCRC          uint32
	H2EnablePush           uint8
	H2InitialWindow        uint32
	H2WindowUpdateInc      uint32
	H2PseudoOrder          uint16
	H2PseudoOrderCount     uint8
	FraudReason            string
	FraudScore             uint32
	LayerDesyncCount       uint8
	SilentRejectEvent      bool
	ReviewRoutedEvent      bool
	ShadowEvent            bool
	SmokeEvent             bool
	CreatedAt              time.Time
	StringBuffer           []byte
	Scratch                unsafe.Pointer
	FilterDeadlineMono     int64
	FilterWorkerIdx        int8
	FilterCampResolved     bool
	FilterCamp             *Campaign
	IngestGeoResolved      bool
	IngestAnonymous        bool
	GeoHash                uint32
	GeoCountry             string
	ClearingPriceMicro     int64
	IngressCostMicro       int64
	ClickIDBuf             [36]byte
	UserPIIHash            [16]byte
	HasUserPIIHash         bool
	TCPMSS                 uint16
	TCPMSSSet              uint8
	TCPTTL                 uint8
	TCPTTLSet              uint8
	TCPWindow              uint16
	TCPWindowSet           uint8
	TCPSig                 uint32
	TCPSigSet              uint8
	RTTSynMS               uint16
	TTFBAppMS              uint16
	RTTSplitDeltaMS        uint16
	ConnTimingSet          uint8
	JSONSerializationFlags uint8
	TelemetrySet           uint8
	TelemetryEvents        []BehaviorTelemetryEvent
	MobileTouchCount       uint8
	MobileGyroSamples      uint8
	MobileGyroVariance     uint16
	MobileGyroFlat         uint8
	MobileBiometricSet     uint8
	MobileBiometricMobile  uint8
}

func (e *Event) Reset() {
	e.ClickID = ""
	e.CampaignID = uuid.Nil
	e.UserID = ""
	e.Type = ""
	e.PlacementID = ""
	if cap(e.Payload) > 4096 {
		e.Payload = make([]byte, 0, 1024)
	} else {
		e.Payload = e.Payload[:0]
	}
	e.IP = ""
	e.UA = ""
	e.TLSHash = ""
	e.TLSJA3 = ""
	e.TLSJA4 = ""
	e.SecCHUA = ""
	e.SecCHUAPlatform = ""
	e.TLSALPN = ""
	e.AcceptLang = ""
	e.SecFetchPresent = 0
	e.SecFetchSite = 0
	e.SecFetchMode = 0
	e.SecFetchDest = 0
	e.SecCHUAMobile = 0
	e.IngressH2 = 0
	e.HTTP1HeaderOrderCount = 0
	e.AcceptEncodingFlags = 0
	e.AcceptEncodingSet = 0
	e.H2WireFlags = 0
	e.H2SettingsCRC = 0
	e.H2EnablePush = 0
	e.H2InitialWindow = 0
	e.H2WindowUpdateInc = 0
	e.H2PseudoOrder = 0
	e.H2PseudoOrderCount = 0
	e.FraudReason = ""
	e.FraudScore = 0
	e.LayerDesyncCount = 0
	e.SilentRejectEvent = false
	e.ReviewRoutedEvent = false
	e.ShadowEvent = false
	e.SmokeEvent = false
	e.CreatedAt = time.Time{}
	e.Scratch = nil
	e.FilterDeadlineMono = 0
	e.FilterWorkerIdx = -1
	e.FilterCampResolved = false
	e.FilterCamp = nil
	e.IngestGeoResolved = false
	e.IngestAnonymous = false
	e.GeoHash = 0
	e.GeoCountry = ""
	e.ClearingPriceMicro = 0
	e.IngressCostMicro = 0
	e.HasUserPIIHash = false
	e.TCPMSS = 0
	e.TCPMSSSet = 0
	e.TCPTTL = 0
	e.TCPTTLSet = 0
	e.TCPWindow = 0
	e.TCPWindowSet = 0
	e.TCPSig = 0
	e.TCPSigSet = 0
	e.RTTSynMS = 0
	e.TTFBAppMS = 0
	e.RTTSplitDeltaMS = 0
	e.ConnTimingSet = 0
	e.JSONSerializationFlags = 0
	e.TelemetrySet = 0
	if cap(e.TelemetryEvents) > 64 {
		e.TelemetryEvents = make([]BehaviorTelemetryEvent, 0, 8)
	} else {
		e.TelemetryEvents = e.TelemetryEvents[:0]
	}
	e.MobileTouchCount = 0
	e.MobileGyroSamples = 0
	e.MobileGyroVariance = 0
	e.MobileGyroFlat = 0
	e.MobileBiometricSet = 0
	e.MobileBiometricMobile = 0
	if cap(e.StringBuffer) > 2048 {
		e.StringBuffer = make([]byte, 0, 256)
	} else {
		e.StringBuffer = e.StringBuffer[:0]
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
