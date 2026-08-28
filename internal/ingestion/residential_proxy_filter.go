package ingestion

import (
	"context"

	"ad-event-processor/internal/domain"
	"ad-event-processor/internal/metrics"
)

type ResidentialProxyFilter struct {
	ring         *ResidentialProxyRing
	intelTable   *ResidentialIntelTable
	intelEnabled bool
}

func NewResidentialProxyFilter(ring *ResidentialProxyRing) *ResidentialProxyFilter {
	if ring == nil {
		return &ResidentialProxyFilter{}
	}
	return &ResidentialProxyFilter{ring: ring}
}

func (f *ResidentialProxyFilter) SetIntelTable(table *ResidentialIntelTable, enabled bool) {
	if f == nil {
		return
	}
	f.intelTable = table
	f.intelEnabled = enabled && table != nil
}

func (f *ResidentialProxyFilter) Check(ctx context.Context, evt *domain.Event) error {
	if f == nil || evt == nil {
		return nil
	}
	if f.intelEnabled && f.intelTable.Ready() && f.intelTable.MatchIP(evt.IP) {
		metrics.ResidentialIntelHotMatchTotal.Inc()
		metrics.ResidentialProxySignalTotal.Inc()
		addFraudSignal(evt, FraudReasonResidentialProxy)
		return nil
	}
	if f.ring == nil {
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
