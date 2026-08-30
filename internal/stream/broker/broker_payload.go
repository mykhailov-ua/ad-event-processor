package broker

import (
	"encoding/binary"
	"time"

	"ad-event-processor/internal/domain"
	"ad-event-processor/internal/filter"
	"ad-event-processor/internal/ingest/pb"
	"ad-event-processor/internal/stream/codec"

	"github.com/google/uuid"
)

// ParseBrokerPayloadStream walks a broker mmap fetch blob: repeated uvarint(len) || vtproto message.
// Used by BrokerStreamConsumer after pkg/broker/client.Fetch; tolerates legacy single-message blobs.
//
// Verify: go test ./internal/ingest/ -short -run TestFault_BrokerLiveConsumer_CorruptPayload -count=1
func ParseBrokerPayloadStream(data []byte, fn func(evt *domain.Event)) error {
	if len(data) == 0 {
		return nil
	}
	offset := 0
	parsedAny := false

	for offset < len(data) {
		length, n := binary.Uvarint(data[offset:])
		if n <= 0 || offset+n+int(length) > len(data) {
			// Truncated tail or non-framed blob: try whole buffer as one AdStreamEvent (older producers).
			if !parsedAny {
				evt, err := ParseBrokerPayload(data)
				if err != nil {
					return err
				}
				fn(evt)
				return nil
			}
			break
		}
		msgBytes := data[offset+n : offset+n+int(length)]
		evt, err := ParseBrokerPayload(msgBytes)
		if err != nil {
			// First frame corrupt: fall back to raw parse once; partial batch stops at first bad frame.
			if !parsedAny && offset == 0 {
				if rawEvt, rawErr := ParseBrokerPayload(data); rawErr == nil {
					fn(rawEvt)
					return nil
				}
			}
			if !parsedAny {
				return err
			}
			// Mid-batch corrupt frame: return nil so consumer skips this WAL offset and keeps prior events.
			break
		}
		fn(evt)
		parsedAny = true
		offset += n + int(length)
	}
	return nil
}

// ParseBrokerPayload decodes one broker record into domain.Event. Tries AdStreamEvent VT first
// (tracker BrokerProducer path), then AdLogRecord (legacy / fraud batch). Returns pooled event on success.
func ParseBrokerPayload(data []byte) (*domain.Event, error) {
	evt := domain.EventPool.Get().(*domain.Event)
	evt.Reset()

	pbEvt := codec.StreamEventPool.Get().(*pb.AdStreamEvent)
	codec.DeepResetAdStreamEvent(pbEvt)
	if err := pbEvt.UnmarshalVT(data); err == nil {
		fillEventFromStreamProto(pbEvt, evt)
		codec.DeepResetAdStreamEvent(pbEvt)
		codec.StreamEventPool.Put(pbEvt)
		return evt, nil
	}
	codec.DeepResetAdStreamEvent(pbEvt)
	codec.StreamEventPool.Put(pbEvt)

	rec := codec.AdLogRecordPool.Get().(*pb.AdLogRecord)
	rec.Reset()
	if err := rec.UnmarshalVT(data); err == nil {
		fillEventFromLogRecord(rec, evt)
		campIDSaved := rec.CampaignId
		rec.Reset()
		if cap(campIDSaved) >= 16 {
			rec.CampaignId = campIDSaved[:0]
		}
		codec.AdLogRecordPool.Put(rec)
		return evt, nil
	}
	campIDSaved := rec.CampaignId
	rec.Reset()
	if cap(campIDSaved) >= 16 {
		rec.CampaignId = campIDSaved[:0]
	}
	codec.AdLogRecordPool.Put(rec)
	domain.EventPool.Put(evt)
	return nil, ErrBrokerPayloadUnrecognized
}

// fillEventFromStreamProto maps hot-path vtproto fields into domain.Event with one StringBuffer arena.
// Strings alias StringBuffer until consumer returns evt to EventPool (cold path only).
func fillEventFromStreamProto(pbEvt *pb.AdStreamEvent, evt *domain.Event) {
	totalLen := len(pbEvt.ClickId) + len(pbEvt.EventType) + len(pbEvt.Ip) + len(pbEvt.Ua) + len(pbEvt.FraudReason)
	if cap(evt.StringBuffer) < totalLen {
		evt.StringBuffer = make([]byte, 0, totalLen+128)
	} else {
		evt.StringBuffer = evt.StringBuffer[:0]
	}

	evt.StringBuffer = append(evt.StringBuffer, pbEvt.ClickId...)
	evt.ClickID = filter.UnsafeString(evt.StringBuffer[len(evt.StringBuffer)-len(pbEvt.ClickId):])

	evt.StringBuffer = append(evt.StringBuffer, pbEvt.EventType...)
	evt.Type = filter.UnsafeString(evt.StringBuffer[len(evt.StringBuffer)-len(pbEvt.EventType):])

	evt.StringBuffer = append(evt.StringBuffer, pbEvt.Ip...)
	evt.IP = filter.UnsafeString(evt.StringBuffer[len(evt.StringBuffer)-len(pbEvt.Ip):])

	evt.StringBuffer = append(evt.StringBuffer, pbEvt.Ua...)
	evt.UA = filter.UnsafeString(evt.StringBuffer[len(evt.StringBuffer)-len(pbEvt.Ua):])

	if len(pbEvt.UserId) > 0 {
		evt.StringBuffer = append(evt.StringBuffer, pbEvt.UserId...)
		evt.UserID = filter.UnsafeString(evt.StringBuffer[len(evt.StringBuffer)-len(pbEvt.UserId):])
	}

	_ = filter.ParseUUID(pbEvt.CampaignId, &evt.CampaignID)
	evt.Payload = append(evt.Payload[:0], pbEvt.Payload...)
	if len(pbEvt.FraudReason) > 0 {
		evt.StringBuffer = append(evt.StringBuffer, pbEvt.FraudReason...)
		evt.FraudReason = filter.UnsafeString(evt.StringBuffer[len(evt.StringBuffer)-len(pbEvt.FraudReason):])
	}
	evt.FraudScore = pbEvt.FraudScore
	evt.LayerDesyncCount = uint8(pbEvt.LayerDesyncCount)
	evt.SilentRejectEvent = pbEvt.SilentRejectEvent
	if pbEvt.CreatedAtUnix > 0 {
		evt.CreatedAt = time.Unix(pbEvt.CreatedAtUnix, 0)
	}
}

// fillEventFromLogRecord handles compact AdLogRecord wire (click_id + event_type + campaign_id bytes).
func fillEventFromLogRecord(rec *pb.AdLogRecord, evt *domain.Event) {
	if cap(evt.StringBuffer) < len(rec.ClickId)+len(rec.EventType) {
		evt.StringBuffer = make([]byte, 0, len(rec.ClickId)+len(rec.EventType)+64)
	} else {
		evt.StringBuffer = evt.StringBuffer[:0]
	}
	evt.StringBuffer = append(evt.StringBuffer, rec.ClickId...)
	evt.ClickID = filter.UnsafeString(evt.StringBuffer[len(evt.StringBuffer)-len(rec.ClickId):])
	evt.StringBuffer = append(evt.StringBuffer, rec.EventType...)
	evt.Type = filter.UnsafeString(evt.StringBuffer[len(evt.StringBuffer)-len(rec.EventType):])
	if len(rec.CampaignId) >= 16 {
		_ = filter.ParseUUID(rec.CampaignId[:16], &evt.CampaignID)
	} else {
		evt.CampaignID = uuid.Nil
	}
	if rec.TimestampUnix > 0 {
		evt.CreatedAt = time.Unix(rec.TimestampUnix, 0)
	}
}
