package ingestion

import (
	"sync/atomic"

	"ad-event-processor/internal/domain"
	"ad-event-processor/internal/metrics"
	"ad-event-processor/pkg/logger"
)

const filterRejectSampleEventType = "filter_reject"

var filterRejectCountrySampleSeq atomic.Uint64

func normalizeRejectCountry(country string) string {
	if len(country) != 2 {
		return ""
	}
	b0, b1 := country[0], country[1]
	if b0 < 'A' || b0 > 'Z' || b1 < 'A' || b1 > 'Z' {
		return ""
	}
	return country
}

func truncateRejectPlacement(placement string) string {
	if len(placement) <= 64 {
		return placement
	}
	return placement[:64]
}

func appendRejectSamplePayload(dst []byte, kind, placement, country string) []byte {
	dst = append(dst, `{"k":"`...)
	dst = append(dst, kind...)
	dst = append(dst, `","p":"`...)
	dst = append(dst, placement...)
	dst = append(dst, `","c":"`...)
	dst = append(dst, country...)
	dst = append(dst, `"}`...)
	return dst
}

func recordFilterRejectCountrySample(kind filterRejectKind, evt *domain.Event, seq *atomic.Uint64, sampleMask uint64) {
	if evt == nil {
		return
	}
	country := normalizeRejectCountry(evt.GeoCountry)
	if country == "" {
		return
	}
	counter := seq
	mask := sampleMask
	if counter == nil {
		counter = &filterRejectCountrySampleSeq
		mask = auditLogSampleMaskDefault
	}
	if !shouldSampleHistogram(counter.Add(1), mask) {
		return
	}
	reason := filterRejectSpecs[kind].metricLabel
	metrics.FilterRejectCountryTotal.WithLabelValues(reason, country).Inc()
}

func writeFilterRejectSample(
	l *logger.Logger,
	seq *atomic.Uint64,
	sampleMask uint64,
	shardID int,
	evt *domain.Event,
	kind filterRejectKind,
) {
	if l == nil || evt == nil || seq == nil {
		return
	}
	if !shouldSampleHistogram(seq.Add(1), sampleMask) {
		return
	}
	placement := truncateRejectPlacement(evt.PlacementID)
	country := normalizeRejectCountry(evt.GeoCountry)
	if placement == "" && country == "" {
		return
	}

	sample := domain.EventPool.Get().(*domain.Event)
	sample.Reset()
	sample.ClickID = evt.ClickID
	sample.CampaignID = evt.CampaignID
	sample.Type = filterRejectSampleEventType
	sample.PlacementID = placement
	sample.GeoCountry = country
	sample.Payload = appendRejectSamplePayload(sample.Payload[:0], filterRejectSpecs[kind].metricLabel, placement, country)
	writeAuditLog(l, seq, 0, shardID, sample)
	domain.EventPool.Put(sample)
}

func recordFilterRejectDimensions(
	l *logger.Logger,
	seq *atomic.Uint64,
	sampleMask uint64,
	shardID int,
	evt *domain.Event,
	kind filterRejectKind,
) {
	recordFilterRejectCountrySample(kind, evt, seq, sampleMask)
	writeFilterRejectSample(l, seq, sampleMask, shardID, evt, kind)
}
