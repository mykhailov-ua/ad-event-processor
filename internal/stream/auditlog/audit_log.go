package auditlog

import (
	"sync/atomic"
	"time"

	"ad-event-processor/internal/domain"
	"ad-event-processor/internal/filter"
	"ad-event-processor/internal/ingest/pb"
	"ad-event-processor/internal/metrics"
	"ad-event-processor/internal/stream/codec"
	"ad-event-processor/pkg/logger"

	"github.com/google/uuid"
)

const SampleMaskDefault = 127

func SampleMaskFromConfig(cfgVal int) uint64 {
	return filter.HistogramSampleMaskFromConfig(cfgVal)
}

func auditLogPriority(eventType string) uint8 {
	switch eventType {
	case "click", "conversion", "filter_reject":
		return 1
	default:
		return 0
	}
}

func Write(
	l *logger.Logger,
	seq *atomic.Uint64,
	sampleMask uint64,
	shardID int,
	evt *domain.Event,
) {
	if l == nil || evt == nil {
		return
	}
	priority := auditLogPriority(evt.Type)
	if priority == 0 {
		if !filter.ShouldSampleHistogram(seq.Add(1), sampleMask) {
			return
		}
	}

	createdAt := evt.CreatedAt
	if createdAt.IsZero() {
		createdAt = filter.CachedTimeUTC()
	}

	pbEvt := codec.StreamEventPool.Get().(*pb.AdStreamEvent)
	pbEvt.ClickId = codec.UnsafeBytes(evt.ClickID)
	pbEvt.CampaignId = evt.CampaignID[:]
	pbEvt.EventType = codec.UnsafeBytes(evt.Type)
	pbEvt.Payload = evt.Payload
	pbEvt.Ip = codec.UnsafeBytes(evt.IP)
	pbEvt.Ua = codec.UnsafeBytes(evt.UA)
	pbEvt.UserId = codec.UnsafeBytes(evt.UserID)
	pbEvt.CreatedAtUnix = createdAt.Unix()
	pbEvt.FraudScore = evt.FraudScore
	pbEvt.FraudReason = codec.UnsafeBytes(evt.FraudReason)
	pbEvt.SilentRejectEvent = evt.SilentRejectEvent
	pbEvt.ReviewRoutedEvent = evt.ReviewRoutedEvent

	size := pbEvt.SizeVT()
	bufPtr := codec.LogBufPool.Get().(*[]byte)
	buf := *bufPtr
	if cap(buf) < size {
		buf = make([]byte, size)
	} else {
		buf = buf[:size]
	}

	n, err := pbEvt.MarshalToSizedBufferVT(buf)
	if err == nil {
		if !l.WriteToShard(shardID, priority, buf[:n]) {
			metrics.HandlerLogDropTotal.Inc()
		}
	}
	*bufPtr = buf
	codec.LogBufPool.Put(bufPtr)

	codec.ClearAdStreamEvent(pbEvt)
	codec.StreamEventPool.Put(pbEvt)
}

func EventFromFields(ts int64, campaignID uuid.UUID, clickID, eventType string) *domain.Event {
	evt := domain.EventPool.Get().(*domain.Event)
	evt.Reset()
	evt.ClickID = clickID
	evt.CampaignID = campaignID
	evt.Type = eventType
	if ts > 0 {
		evt.CreatedAt = time.Unix(ts, 0)
	}
	return evt
}
