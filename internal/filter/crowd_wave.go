package filter

import (
	"context"
	"time"

	"ad-event-processor/internal/domain"
	"ad-event-processor/internal/metrics"
	"ad-event-processor/pkg/crowdwave"

	"github.com/google/uuid"
)

type CrowdWaveFilter struct {
	registry domain.CampaignRegistry
	enabled  bool
	store    *CrowdWaveStore
}

func NewCrowdWaveFilter(registry domain.CampaignRegistry) *CrowdWaveFilter {
	return &CrowdWaveFilter{registry: registry}
}

func (f *CrowdWaveFilter) SetEnabled(enabled bool) {
	if f != nil {
		f.enabled = enabled
	}
}

func (f *CrowdWaveFilter) SetStore(store *CrowdWaveStore) {
	if f != nil {
		f.store = store
	}
}

func (f *CrowdWaveFilter) Check(ctx context.Context, evt *domain.Event) error {
	if f == nil || evt == nil || !f.enabled || f.store == nil {
		return nil
	}
	if evt.Type != "click" && evt.Type != "conversion" {
		return nil
	}
	if evt.ProbeClusterSet == 0 || evt.ProbeClusterID[0] == 0 {
		return nil
	}
	if f.registry == nil {
		return nil
	}
	camp, ok := f.registry.GetCampaign(evt.CampaignID)
	if !ok || camp == nil || !campaignRequiresCrowdProbe(camp) {
		return nil
	}
	simhash := BehaviorSimhashForEvent(evt)
	if simhash == 0 {
		return nil
	}
	evt.BehaviorSimhash = simhash
	evt.BehaviorSimhashSet = 1
	state, err := f.store.Observe(ctx, evt.CampaignID, evt.ProbeClusterID, simhash, time.Now().Unix())
	if err != nil {
		return nil
	}
	evt.CrowdWaveScore = state.Score
	if state.Active {
		evt.CrowdWaveActive = 1
		metrics.HybridCrowdWaveTotal.WithLabelValues(evt.CampaignID.String()).Inc()
	}
	return nil
}

func (f *CrowdWaveFilter) WaveActive(ctx context.Context, campaignID uuid.UUID) (crowdwave.WaveState, bool) {
	if f == nil || !f.enabled || f.store == nil || campaignID == uuid.Nil {
		return crowdwave.WaveState{}, false
	}
	state, err := f.store.Snapshot(ctx, campaignID)
	if err != nil || !state.Active {
		return state, false
	}
	return state, true
}

func (f *CrowdWaveFilter) PromotionBlocked(ctx context.Context, campaignID uuid.UUID) (bool, uint16) {
	state, active := f.WaveActive(ctx, campaignID)
	return active, state.Score
}

func BehaviorSimhashForEvent(evt *domain.Event) uint64 {
	if evt == nil {
		return 0
	}
	if evt.TelemetrySet != 0 && len(evt.TelemetryEvents) > 0 {
		return crowdwave.BehaviorSimhash(evt.TelemetryEvents)
	}
	if evt.EventOrderEntropyMilli > 0 {
		return uint64(evt.EventOrderEntropyMilli)<<32 | uint64(evt.ProbeBehaviorScore)
	}
	return 0
}
