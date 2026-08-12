package ingestion

import (
	"github.com/bidshard/ad-event-processor/internal/domain"
	"github.com/bidshard/ad-event-processor/internal/metrics"
	"github.com/google/uuid"
)

func safePageEligibleReject(kind filterRejectKind) bool {
	switch kind {
	case filterRejectFraud, filterRejectPlacementBlocked:
		return true
	default:
		return false
	}
}

func resolveSafePageLanding(registry domain.CampaignRegistry, campaignID uuid.UUID) (string, bool) {
	if registry == nil {
		return "", false
	}
	camp, ok := registry.GetCampaign(campaignID)
	if !ok || camp == nil || !camp.SafePageEnabled {
		return "", false
	}
	if camp.SafePageURL == "" {
		return "", false
	}
	return camp.SafePageURL, true
}

func trySafePageRedirect(registry domain.CampaignRegistry, campaignID uuid.UUID, outcome trackOutcome) (string, bool) {
	if outcome.Status != trackStatusFraudAccepted &&
		!(outcome.Status == trackStatusRejected && safePageEligibleReject(outcome.RejectKind)) {
		return "", false
	}
	url, ok := resolveSafePageLanding(registry, campaignID)
	if !ok {
		return "", false
	}
	metrics.SafePageRedirectTotal.Inc()
	return url, true
}
