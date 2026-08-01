package ingestion

import (
	"espx/internal/domain"
)

func ResolveLandingURL(registry domain.CampaignRegistry, store *BrandCreativeStore, evt *domain.Event) string {
	return unsafeString(ResolveLandingURLBytes(registry, store, evt))
}

func ResolveLandingURLBytes(registry domain.CampaignRegistry, store *BrandCreativeStore, evt *domain.Event) []byte {
	if store == nil || registry == nil || evt.Type != "click" {
		return nil
	}
	camp, ok := registry.GetCampaign(evt.CampaignID)
	if !ok || camp.BrandID == nil {
		return nil
	}
	return store.SelectLandingURLBytes(*camp.BrandID, evt.UserID)
}
