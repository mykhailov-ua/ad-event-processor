package ingestion

import (
	"sync"
	"unsafe"

	"ad-event-processor/internal/ingestion/pb"
	"ad-event-processor/pkg/money"
)

const MicroUnitFactor = money.MicroUnit

func SliceToMap(slice []string) map[string]struct{} {
	if slice == nil {
		return nil
	}
	m := make(map[string]struct{}, len(slice))
	for _, s := range slice {
		m[s] = struct{}{}
	}
	return m
}

func UnsafeString(b []byte) string {
	if len(b) == 0 {
		return ""
	}
	return unsafe.String(unsafe.SliceData(b), len(b))
}

func UnsafeBytes(s string) []byte {
	if s == "" {
		return nil
	}
	return unsafe.Slice(unsafe.StringData(s), len(s))
}

type ByteSliceValue struct {
	b []byte
}

func (v *ByteSliceValue) MarshalBinary() ([]byte, error) {
	return v.b, nil
}

var byteSliceValuePool = sync.Pool{
	New: func() any {
		return new(ByteSliceValue)
	},
}

func DeepResetAdStreamEvent(m *pb.AdStreamEvent) {
	if m == nil {
		return
	}
	m.ClickId = m.ClickId[:0]
	m.CampaignId = m.CampaignId[:0]
	m.EventType = m.EventType[:0]
	m.Payload = m.Payload[:0]
	m.Ip = m.Ip[:0]
	m.Ua = m.Ua[:0]
	m.FraudReason = m.FraudReason[:0]
	m.CreatedAtUnix = 0
	m.FraudScore = 0
	m.SilentRejectEvent = false
	m.ReviewRoutedEvent = false
}

func ClearAdStreamEvent(m *pb.AdStreamEvent) {
	if m == nil {
		return
	}
	m.ClickId = nil
	m.CampaignId = nil
	m.EventType = nil
	m.Payload = nil
	m.Ip = nil
	m.Ua = nil
	m.FraudReason = nil
	m.CreatedAtUnix = 0
	m.FraudScore = 0
	m.SilentRejectEvent = false
	m.ReviewRoutedEvent = false
}

func DeepResetAdDLQEvent(m *pb.AdDLQEvent) {
	if m == nil {
		return
	}
	if m.OriginalEvent != nil {
		DeepResetAdStreamEvent(m.OriginalEvent)
	}
	m.Error = m.Error[:0]
	m.OriginalId = m.OriginalId[:0]
	m.WorkerId = m.WorkerId[:0]
	m.FailedAtUnix = 0
	m.RetryCount = 0
}
