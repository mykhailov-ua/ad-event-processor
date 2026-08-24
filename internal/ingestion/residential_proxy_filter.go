package ingestion

import (
	"context"

	"ad-event-processor/internal/domain"
	"ad-event-processor/internal/metrics"
)

type ResidentialProxyFilter struct {
	ring *ResidentialProxyRing
}

func NewResidentialProxyFilter(ring *ResidentialProxyRing) *ResidentialProxyFilter {
	if ring == nil {
		return nil
	}
	return &ResidentialProxyFilter{ring: ring}
}

func (f *ResidentialProxyFilter) Check(ctx context.Context, evt *domain.Event) error {
	if f == nil || f.ring == nil || evt == nil {
		return nil
	}
	isClick := evt.Type == "click"
	userHash := hashResidentialProxyUser(evt.UserID)
	uaHash := hashResidentialProxyUA(evt.UA)
	campaignHash := crc32Castagnoli(&evt.CampaignID)
	_, signal := f.ring.observe(campaignHash, isClick, userHash, uaHash, monotonicNano())
	if signal {
		metrics.ResidentialProxySignalTotal.Inc()
		addFraudSignal(evt, FraudReasonResidentialProxy)
	}
	return nil
}
