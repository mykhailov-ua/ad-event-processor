package ingest

import (
	"ad-event-processor/internal/filter"
	"ad-event-processor/internal/metrics"

	"github.com/prometheus/client_golang/prometheus"
)

type PreboundTrackMetrics struct {
	throughputProto prometheus.Counter
	throughputJSON  prometheus.Counter

	decisionAccepted         prometheus.Counter
	decisionEmergencyBreaker prometheus.Counter
	decisionRateLimited      prometheus.Counter
	decisionDuplicate        prometheus.Counter
	decisionBudgetExhausted  prometheus.Counter
	decisionPacingLimit      prometheus.Counter
	decisionFrequencyCapped  prometheus.Counter
	decisionGeoBlocked       prometheus.Counter
	decisionScheduleBlocked  prometheus.Counter
	decisionCampaignNotFound prometheus.Counter
	decisionBidFloor         prometheus.Counter
	decisionFilterTimeout    prometheus.Counter
	decisionFraud            prometheus.Counter
	decisionConsentDenied    prometheus.Counter
	decisionInfraUnavailable prometheus.Counter
	decisionRegistryStale    prometheus.Counter
	decisionShardUnavailable prometheus.Counter
	decisionProducerOverload prometheus.Counter

	blockedEmergencyBreaker prometheus.Counter
	blockedRateLimit        prometheus.Counter
	blockedDuplicate        prometheus.Counter
	blockedBudget           prometheus.Counter
	blockedPacing           prometheus.Counter
	blockedFreq             prometheus.Counter
	blockedGeo              prometheus.Counter
	blockedSchedule         prometheus.Counter
	blockedCampaignNotFound prometheus.Counter
	blockedBidFloor         prometheus.Counter
	blockedFilterTimeout    prometheus.Counter
	blockedFraud            prometheus.Counter
	blockedConsent          prometheus.Counter
	blockedInfra            prometheus.Counter
	blockedRegistryStale    prometheus.Counter
	blockedShardUnavailable prometheus.Counter
	blockedProducerOverload prometheus.Counter
}

func NewPreboundTrackMetrics() PreboundTrackMetrics {
	return PreboundTrackMetrics{
		throughputProto: metrics.FilterThroughput.WithLabelValues("protobuf"),
		throughputJSON:  metrics.FilterThroughput.WithLabelValues("json"),

		decisionAccepted:         metrics.FilterDecisions.WithLabelValues("accepted"),
		decisionEmergencyBreaker: metrics.FilterDecisions.WithLabelValues("emergency_breaker"),
		decisionRateLimited:      metrics.FilterDecisions.WithLabelValues("rate_limited"),
		decisionDuplicate:        metrics.FilterDecisions.WithLabelValues("duplicate"),
		decisionBudgetExhausted:  metrics.FilterDecisions.WithLabelValues("budget_exhausted"),
		decisionPacingLimit:      metrics.FilterDecisions.WithLabelValues("pacing_limit"),
		decisionFrequencyCapped:  metrics.FilterDecisions.WithLabelValues("frequency_capped"),
		decisionGeoBlocked:       metrics.FilterDecisions.WithLabelValues("geo_blocked"),
		decisionScheduleBlocked:  metrics.FilterDecisions.WithLabelValues("schedule_blocked"),
		decisionCampaignNotFound: metrics.FilterDecisions.WithLabelValues("campaign_not_found"),
		decisionBidFloor:         metrics.FilterDecisions.WithLabelValues("bid_floor"),
		decisionFilterTimeout:    metrics.FilterDecisions.WithLabelValues("filter_timeout"),
		decisionFraud:            metrics.FilterDecisions.WithLabelValues("fraud"),
		decisionConsentDenied:    metrics.FilterDecisions.WithLabelValues("consent_denied"),
		decisionInfraUnavailable: metrics.FilterDecisions.WithLabelValues("infra_unavailable"),
		decisionRegistryStale:    metrics.FilterDecisions.WithLabelValues("registry_stale"),
		decisionShardUnavailable: metrics.FilterDecisions.WithLabelValues("shard_unavailable"),
		decisionProducerOverload: metrics.FilterDecisions.WithLabelValues("producer_overload"),

		blockedEmergencyBreaker: metrics.FilterBlockedTotal.WithLabelValues("emergency_breaker"),
		blockedRateLimit:        metrics.FilterBlockedTotal.WithLabelValues("rate_limit"),
		blockedDuplicate:        metrics.FilterBlockedTotal.WithLabelValues("duplicate"),
		blockedBudget:           metrics.FilterBlockedTotal.WithLabelValues("budget"),
		blockedPacing:           metrics.FilterBlockedTotal.WithLabelValues("pacing"),
		blockedFreq:             metrics.FilterBlockedTotal.WithLabelValues("freq"),
		blockedGeo:              metrics.FilterBlockedTotal.WithLabelValues("geo"),
		blockedSchedule:         metrics.FilterBlockedTotal.WithLabelValues("schedule"),
		blockedCampaignNotFound: metrics.FilterBlockedTotal.WithLabelValues("campaign_not_found"),
		blockedBidFloor:         metrics.FilterBlockedTotal.WithLabelValues("bid_floor"),
		blockedFilterTimeout:    metrics.FilterBlockedTotal.WithLabelValues("filter_timeout"),
		blockedFraud:            metrics.FilterBlockedTotal.WithLabelValues("fraud"),
		blockedConsent:          metrics.FilterBlockedTotal.WithLabelValues("consent_denied"),
		blockedInfra:            metrics.FilterBlockedTotal.WithLabelValues("infra_unavailable"),
		blockedRegistryStale:    metrics.FilterBlockedTotal.WithLabelValues("registry_stale"),
		blockedShardUnavailable: metrics.FilterBlockedTotal.WithLabelValues("shard_unavailable"),
		blockedProducerOverload: metrics.FilterBlockedTotal.WithLabelValues("producer_overload"),
	}
}

func (m *PreboundTrackMetrics) recordFilterReject(kind filter.FilterRejectKind) {
	switch kind {
	case filter.FilterRejectEmergencyBreaker:
		m.blockedEmergencyBreaker.Inc()
		m.decisionEmergencyBreaker.Inc()
	case filter.FilterRejectRateLimit:
		m.blockedRateLimit.Inc()
		m.decisionRateLimited.Inc()
	case filter.FilterRejectDuplicate:
		m.blockedDuplicate.Inc()
		m.decisionDuplicate.Inc()
	case filter.FilterRejectBudget:
		m.blockedBudget.Inc()
		m.decisionBudgetExhausted.Inc()
	case filter.FilterRejectPacing:
		m.blockedPacing.Inc()
		m.decisionPacingLimit.Inc()
	case filter.FilterRejectFreq:
		m.blockedFreq.Inc()
		m.decisionFrequencyCapped.Inc()
	case filter.FilterRejectGeo:
		m.blockedGeo.Inc()
		m.decisionGeoBlocked.Inc()
	case filter.FilterRejectSchedule:
		m.blockedSchedule.Inc()
		m.decisionScheduleBlocked.Inc()
	case filter.FilterRejectCampaignNotFound:
		m.blockedCampaignNotFound.Inc()
		m.decisionCampaignNotFound.Inc()
	case filter.FilterRejectBidFloor:
		m.blockedBidFloor.Inc()
		m.decisionBidFloor.Inc()
	case filter.FilterRejectTimeout:
		m.blockedFilterTimeout.Inc()
		m.decisionFilterTimeout.Inc()
	case filter.FilterRejectFraud, filter.FilterRejectFraudBlocked:
		m.blockedFraud.Inc()
		m.decisionFraud.Inc()
	case filter.FilterRejectConsent:
		m.blockedConsent.Inc()
		m.decisionConsentDenied.Inc()
	case filter.FilterRejectInfra:
		m.blockedInfra.Inc()
		m.decisionInfraUnavailable.Inc()
	case filter.FilterRejectRegistryStale:
		m.blockedRegistryStale.Inc()
		m.decisionRegistryStale.Inc()
	case filter.FilterRejectShardUnavailable:
		m.blockedShardUnavailable.Inc()
		m.decisionShardUnavailable.Inc()
	case filter.FilterRejectProducerOverload:
		m.blockedProducerOverload.Inc()
		m.decisionProducerOverload.Inc()
	}
}
