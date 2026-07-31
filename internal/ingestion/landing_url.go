package ingestion

import (
	"espx/internal/domain"
)

func ResolveLandingURL(registry domain.CampaignRegistry, store *BrandCreativeStore, evt *domain.Event) string {
	if store == nil || registry == nil || evt.Type != "click" {
		return ""
	}
	camp, ok := registry.GetCampaign(evt.CampaignID)
	if !ok || camp.BrandID == nil {
		return ""
	}
	return store.SelectLandingURL(*camp.BrandID, evt.UserID)
}
