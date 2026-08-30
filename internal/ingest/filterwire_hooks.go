package ingest

import (
	"net/http"

	"ad-event-processor/internal/domain"
	"ad-event-processor/internal/filter"
	"ad-event-processor/internal/metrics"
	"ad-event-processor/internal/telemetry"
)

type filterRejectSpec struct {
	status      int
	body        string
	gnetResp    []byte
	metricLabel string
}

var filterRejectSpecs = [...]filterRejectSpec{
	filter.FilterRejectEmergencyBreaker:   {http.StatusServiceUnavailable, "service temporarily unavailable", respEmergencyBreaker, "emergency_breaker"},
	filter.FilterRejectRateLimit:          {http.StatusTooManyRequests, "rate limit exceeded", respRateLimit, "rate_limit"},
	filter.FilterRejectDuplicate:          {http.StatusConflict, "duplicate event", respDuplicate, "duplicate"},
	filter.FilterRejectBudget:             {http.StatusPaymentRequired, "budget exhausted", respBudget, "budget"},
	filter.FilterRejectPacing:             {http.StatusTooManyRequests, "pacing limit reached", respPacing, "pacing"},
	filter.FilterRejectFreq:               {http.StatusForbidden, "frequency limit reached", respFreq, "freq"},
	filter.FilterRejectGeo:                {http.StatusForbidden, "geo-targeting blocked", respGeo, "geo"},
	filter.FilterRejectSchedule:           {http.StatusForbidden, "outside delivery schedule", respSchedule, "schedule"},
	filter.FilterRejectCampaignNotFound:   {http.StatusNotFound, "campaign not found", respCampaignNotFound, "campaign_not_found"},
	filter.FilterRejectBidFloor:           {http.StatusPaymentRequired, "bid floor not met", respBidFloorNotMet, "bid_floor"},
	filter.FilterRejectTimeout:            {http.StatusGatewayTimeout, "filter timeout", respFilterTimeout, "filter_timeout"},
	filter.FilterRejectFraud:              {http.StatusAccepted, "", nil, "fraud"},
	filter.FilterRejectConsent:            {http.StatusNoContent, "", respConsentDenied, "consent_denied"},
	filter.FilterRejectInfra:              {http.StatusServiceUnavailable, "service unavailable", respInfraUnavailable, "infra_unavailable"},
	filter.FilterRejectLicenseExpired:     {http.StatusForbidden, "license expired", respLicenseExpired, "license_expired"},
	filter.FilterRejectDailyQuotaExceeded: {http.StatusTooManyRequests, "daily quota exceeded", respDailyQuotaExceeded, "daily_quota_exceeded"},
	filter.FilterRejectPlacementBlocked:   {http.StatusForbidden, "placement blocked", respPlacementBlocked, "placement_blocked"},
	filter.FilterRejectSegmentExcluded:    {http.StatusForbidden, "segment excluded", respSegmentExcluded, "segment_excluded"},
	filter.FilterRejectSegmentNotIncluded: {http.StatusForbidden, "segment not included", respSegmentNotIncluded, "segment_not_included"},
	filter.FilterRejectRegistryStale:      {http.StatusServiceUnavailable, "registry_stale", respRegistryStale, "registry_stale"},
	filter.FilterRejectShardUnavailable:   {http.StatusServiceUnavailable, "shard_unavailable", respShardUnavailable, "shard_unavailable"},
	filter.FilterRejectProducerOverload:   {http.StatusServiceUnavailable, "producer overloaded", respProducerOverload, "producer_overload"},
	filter.FilterRejectFraudBlocked:       {http.StatusForbidden, "fraud blocked", respFraudBlocked, "fraud_blocked"},
}

func recordHTTPFilterReject(kind filter.FilterRejectKind, evt *domain.Event) {
	metrics.FilterBlockedTotal.WithLabelValues(filterRejectSpecs[kind].metricLabel).Inc()
	telemetry.RecordRejected()
	filter.RecordFilterRejectCountrySample(kind, evt, nil, 0)
}

func (h *AdsPacketHandler) recordTrackReject(ctx *ConnContext, evt *domain.Event, kind filter.FilterRejectKind) {
	h.trackMetrics.recordFilterReject(kind)
	if ctx == nil || evt == nil {
		return
	}
	filter.RecordFilterRejectDimensions(h.logger, &h.auditLogSeq, h.auditLogSampleMask, ctx.ShardID, evt, kind)
}
