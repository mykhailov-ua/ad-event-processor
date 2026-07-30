package ingestion

import (
	"espx/internal/campaignmodel"

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

func (f *UnifiedFilter) needsFullLuaPath(evt *campaignmodel.Event, campInfo *campaignmodel.Campaign) bool {
	if evt.Type != "impression" {
		return true
	}
	if !f.fastPathEnabled.Load() {
		return true
	}
	if campInfo.FreqLimit > 0 && evt.UserID != "" {
		return true
	}
	if campInfo.PacingMode == campaignmodel.PacingModeEven {
		return true
	}
	if ttcEnabled(f.ttcMinMsAny) {
		if evt.Type == "click" || evt.Type == "impression" {
			return true
		}
	}
	if f.quotaEnabledAny == oneAny && quotaRefillSample(evt.CampaignID) {
		return true
	}
	return false
}
