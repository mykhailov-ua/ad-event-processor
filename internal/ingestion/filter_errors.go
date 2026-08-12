package ingestion

import (
	"errors"
	"net/http"

	"github.com/bidshard/ad-event-processor/internal/database"
	"github.com/bidshard/ad-event-processor/internal/metrics"
	"github.com/bidshard/ad-event-processor/internal/telemetry"
)

type filterRejectKind uint8

const (
	filterRejectEmergencyBreaker filterRejectKind = iota
	filterRejectRateLimit
	filterRejectDuplicate
	filterRejectBudget
	filterRejectPacing
	filterRejectFreq
	filterRejectGeo
	filterRejectSchedule
	filterRejectCampaignNotFound
	filterRejectBidFloor
	filterRejectTimeout
	filterRejectFraud
	filterRejectConsent
	filterRejectInfra
	filterRejectLicenseExpired
	filterRejectDailyQuotaExceeded
	filterRejectPlacementBlocked
	filterRejectSegmentExcluded
	filterRejectSegmentNotIncluded
	filterRejectRegistryStale
	filterRejectShardUnavailable
)

type filterRejectSpec struct {
	status      int
	body        string
	gnetResp    []byte
	metricLabel string
}

var filterRejectSpecs = [...]filterRejectSpec{
	filterRejectEmergencyBreaker:   {http.StatusServiceUnavailable, "service temporarily unavailable", respEmergencyBreaker, "emergency_breaker"},
	filterRejectRateLimit:          {http.StatusTooManyRequests, "rate limit exceeded", respRateLimit, "rate_limit"},
	filterRejectDuplicate:          {http.StatusConflict, "duplicate event", respDuplicate, "duplicate"},
	filterRejectBudget:             {http.StatusPaymentRequired, "budget exhausted", respBudget, "budget"},
	filterRejectPacing:             {http.StatusTooManyRequests, "pacing limit reached", respPacing, "pacing"},
	filterRejectFreq:               {http.StatusForbidden, "frequency limit reached", respFreq, "freq"},
	filterRejectGeo:                {http.StatusForbidden, "geo-targeting blocked", respGeo, "geo"},
	filterRejectSchedule:           {http.StatusForbidden, "outside delivery schedule", respSchedule, "schedule"},
	filterRejectCampaignNotFound:   {http.StatusNotFound, "campaign not found", respCampaignNotFound, "campaign_not_found"},
	filterRejectBidFloor:           {http.StatusPaymentRequired, "bid floor not met", respBidFloorNotMet, "bid_floor"},
	filterRejectTimeout:            {http.StatusGatewayTimeout, "filter timeout", respFilterTimeout, "filter_timeout"},
	filterRejectFraud:              {http.StatusAccepted, "", nil, "fraud"},
	filterRejectConsent:            {http.StatusNoContent, "", respConsentDenied, "consent_denied"},
	filterRejectInfra:              {http.StatusServiceUnavailable, "service unavailable", respInfraUnavailable, "infra_unavailable"},
	filterRejectLicenseExpired:     {http.StatusForbidden, "license expired", respLicenseExpired, "license_expired"},
	filterRejectDailyQuotaExceeded: {http.StatusTooManyRequests, "daily quota exceeded", respDailyQuotaExceeded, "daily_quota_exceeded"},
	filterRejectPlacementBlocked:   {http.StatusForbidden, "placement blocked", respPlacementBlocked, "placement_blocked"},
	filterRejectSegmentExcluded:    {http.StatusForbidden, "segment excluded", respSegmentExcluded, "segment_excluded"},
	filterRejectSegmentNotIncluded: {http.StatusForbidden, "segment not included", respSegmentNotIncluded, "segment_not_included"},
	filterRejectRegistryStale:      {http.StatusServiceUnavailable, "registry_stale", respRegistryStale, "registry_stale"},
	filterRejectShardUnavailable:   {http.StatusServiceUnavailable, "shard_unavailable", respShardUnavailable, "shard_unavailable"},
}

type FraudReasonID uint8

const (
	FraudReasonCodeDatacenterIP   = "datacenter_ip"
	FraudReasonCodeLowTTC         = "low_ttc"
	FraudReasonCodeMissingImpTS   = "missing_imp_ts"
	FraudReasonCodeL3Blocklist    = "l3_blocklist"
	FraudReasonCodeTLSBlocklist   = "tls_blocklist"
	FraudReasonCodeDeviceMismatch = "device_mismatch"
)

const (
	FraudReasonNone FraudReasonID = iota
	FraudReasonDatacenterIP
	FraudReasonLowTTC
	FraudReasonMissingImpTS
	FraudReasonL3Blocklist
	FraudReasonTLSBlocklist
	FraudReasonDeviceMismatch
	fraudReasonCount
)

const (
	fraudSignalL1High uint8 = 1 << 0
	fraudSignalL2Weak uint8 = 1 << 1
	fraudSignalL3     uint8 = 1 << 2
)

type fraudReasonEntry struct {
	code   string
	weight uint8
	flags  uint8
}

var fraudReasonRegistry = [fraudReasonCount]fraudReasonEntry{
	FraudReasonNone:           {},
	FraudReasonDatacenterIP:   {code: FraudReasonCodeDatacenterIP, weight: 45, flags: fraudSignalL1High},
	FraudReasonLowTTC:         {code: FraudReasonCodeLowTTC, weight: 45, flags: fraudSignalL1High},
	FraudReasonMissingImpTS:   {code: FraudReasonCodeMissingImpTS, weight: 35, flags: fraudSignalL2Weak},
	FraudReasonL3Blocklist:    {code: FraudReasonCodeL3Blocklist, weight: 100, flags: fraudSignalL3},
	FraudReasonTLSBlocklist:   {code: FraudReasonCodeTLSBlocklist, weight: 45, flags: fraudSignalL1High},
	FraudReasonDeviceMismatch: {code: FraudReasonCodeDeviceMismatch, weight: 35, flags: fraudSignalL2Weak},
}

func FraudReasonCode(id FraudReasonID) string {
	if id >= fraudReasonCount {
		return ""
	}
	return fraudReasonRegistry[id].code
}

func FraudSignalWeight(id FraudReasonID) uint8 {
	if id >= fraudReasonCount {
		return 0
	}
	return fraudReasonRegistry[id].weight
}

func FraudSignalFlags(id FraudReasonID) uint8 {
	if id >= fraudReasonCount {
		return 0
	}
	return fraudReasonRegistry[id].flags
}

func classifyFilterErr(err error) (filterRejectKind, bool) {
	switch {
	case errors.Is(err, ErrEmergencyBreakerActive):
		return filterRejectEmergencyBreaker, true
	case errors.Is(err, ErrFilterTimeout):
		return filterRejectTimeout, true
	case isInfraFilterErr(err):
		return filterRejectInfra, true
	case errors.Is(err, ErrRateLimitExceeded):
		return filterRejectRateLimit, true
	case errors.Is(err, ErrDuplicateEvent):
		return filterRejectDuplicate, true
	case errors.Is(err, ErrBudgetExhausted):
		return filterRejectBudget, true
	case errors.Is(err, ErrPacingExhausted):
		return filterRejectPacing, true
	case errors.Is(err, ErrFreqLimitExceeded):
		return filterRejectFreq, true
	case errors.Is(err, ErrGeoBlocked):
		return filterRejectGeo, true
	case errors.Is(err, ErrScheduleBlocked):
		return filterRejectSchedule, true
	case errors.Is(err, ErrCampaignNotFound):
		return filterRejectCampaignNotFound, true
	case errors.Is(err, ErrRegistryStale):
		return filterRejectRegistryStale, true
	case errors.Is(err, ErrShardUnavailable):
		return filterRejectShardUnavailable, true
	case errors.Is(err, ErrBidFloorNotMet):
		return filterRejectBidFloor, true
	case errors.Is(err, ErrMigrationFenced):
		return filterRejectInfra, true
	case errors.Is(err, ErrFraudDetected):
		return filterRejectFraud, true
	case errors.Is(err, ErrConsentDenied):
		return filterRejectConsent, true
	case errors.Is(err, ErrLicenseExpired):
		return filterRejectLicenseExpired, true
	case errors.Is(err, ErrDailyQuotaExceeded):
		return filterRejectDailyQuotaExceeded, true
	case errors.Is(err, ErrPlacementBlocked):
		return filterRejectPlacementBlocked, true
	case errors.Is(err, ErrSegmentExcluded):
		return filterRejectSegmentExcluded, true
	case errors.Is(err, ErrSegmentNotIncluded):
		return filterRejectSegmentNotIncluded, true
	default:
		return 0, false
	}
}

func isInfraFilterErr(err error) bool {
	if errors.Is(err, database.ErrRedisCircuitOpen) {
		return true
	}
	return database.IsNetworkOrSystemError(err)
}

func (m *preboundTrackMetrics) recordFilterReject(kind filterRejectKind) {
	switch kind {
	case filterRejectEmergencyBreaker:
		m.blockedEmergencyBreaker.Inc()
		m.decisionEmergencyBreaker.Inc()
	case filterRejectRateLimit:
		m.blockedRateLimit.Inc()
		m.decisionRateLimited.Inc()
	case filterRejectDuplicate:
		m.blockedDuplicate.Inc()
		m.decisionDuplicate.Inc()
	case filterRejectBudget:
		m.blockedBudget.Inc()
		m.decisionBudgetExhausted.Inc()
	case filterRejectPacing:
		m.blockedPacing.Inc()
		m.decisionPacingLimit.Inc()
	case filterRejectFreq:
		m.blockedFreq.Inc()
		m.decisionFrequencyCapped.Inc()
	case filterRejectGeo:
		m.blockedGeo.Inc()
		m.decisionGeoBlocked.Inc()
	case filterRejectSchedule:
		m.blockedSchedule.Inc()
		m.decisionScheduleBlocked.Inc()
	case filterRejectCampaignNotFound:
		m.blockedCampaignNotFound.Inc()
		m.decisionCampaignNotFound.Inc()
	case filterRejectBidFloor:
		m.blockedBidFloor.Inc()
		m.decisionBidFloor.Inc()
	case filterRejectTimeout:
		m.blockedFilterTimeout.Inc()
		m.decisionFilterTimeout.Inc()
	case filterRejectFraud:
		m.blockedFraud.Inc()
		m.decisionFraud.Inc()
	case filterRejectConsent:
		m.blockedConsent.Inc()
		m.decisionConsentDenied.Inc()
	case filterRejectInfra:
		m.blockedInfra.Inc()
		m.decisionInfraUnavailable.Inc()
	case filterRejectRegistryStale:
		m.blockedRegistryStale.Inc()
		m.decisionRegistryStale.Inc()
	case filterRejectShardUnavailable:
		m.blockedShardUnavailable.Inc()
		m.decisionShardUnavailable.Inc()
	case filterRejectLicenseExpired, filterRejectDailyQuotaExceeded:
	}
}

func recordHTTPFilterReject(kind filterRejectKind) {
	metrics.FilterBlockedTotal.WithLabelValues(filterRejectSpecs[kind].metricLabel).Inc()
	telemetry.RecordRejected()
}
