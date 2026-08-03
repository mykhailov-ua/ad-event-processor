package ingestion

import (
	"espx/internal/domain"

	"github.com/google/uuid"
)

func quotaRefillSample(campaignID uuid.UUID) bool {
	return campaignID[0]%100 == 0
}

func ttcEnabled(ttcMinMsAny any) bool {
	if ttcMinMsAny == nil || ttcMinMsAny == zeroAny {
		return false
	}
	switch v := ttcMinMsAny.(type) {
	case int64:
		return v > 0
	case int:
		return v > 0
	default:
		return false
	}
}

func (f *UnifiedFilter) needsFullLuaPath(evt *domain.Event, campInfo *domain.Campaign) bool {
	if evt.Type != "impression" && evt.Type != "click" {
		return true
	}
	if !f.fastPathEnabled.Load() {
		return true
	}
	if campInfo.FreqLimit > 0 && evt.UserID != "" {
		if f.settingsWatcher == nil {
			return true
		}
	}
	if campInfo.PacingMode == domain.PacingModeEven {
		if f.roughPacing == nil || !campInfo.RoughPacingEnabled() {
			return true
		}
	}
	if ttcEnabled(f.ttcMinMsAny) && f.localTTC == nil {
		return true
	}
	if f.quotaEnabledAny == oneAny && quotaRefillSample(evt.CampaignID) && f.localQuantaRefill == nil {
		return true
	}
	return false
}
