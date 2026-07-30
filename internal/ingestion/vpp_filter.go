package ingestion

import (
	"context"

	"espx/internal/campaignmodel"

	"github.com/google/uuid"
)

type VPPFilter struct {
	registry campaignmodel.CampaignRegistry
	watcher  *SettingsWatcher
}

func NewVPPFilter(registry campaignmodel.CampaignRegistry, watcher *SettingsWatcher) *VPPFilter {
	return &VPPFilter{registry: registry, watcher: watcher}
}

func (f *VPPFilter) Check(ctx context.Context, evt *campaignmodel.Event) error {
	if evt == nil || f.registry == nil {
		return nil
	}
	camp, ok := f.registry.GetCampaign(evt.CampaignID)
	if !ok || camp.PacingMode != campaignmodel.PacingModeVpp {
		return nil
	}
	if f.watcher == nil {
		return nil
	}
	ratio := f.watcher.GetVPPRatio(evt.CampaignID)
	if ratio >= 1.0 {
		return nil
	}
	if !vppAllow(evt.CampaignID, ratio, monotonicNano()) {
		return ErrPacingExhausted
	}
	return nil
}

func vppAllow(campaignID uuid.UUID, ratio float32, nowMono int64) bool {
	if ratio <= 0 {
		return false
	}
	if ratio >= 1.0 {
		return true
	}
	var h uint64
	for i := 0; i < 8; i++ {
		h ^= uint64(campaignID[i]) << (uint(i) * 8)
	}
	bucket := uint64(nowMono) ^ h
	threshold := uint64(ratio * 10000)
	return bucket%10000 < threshold
}
