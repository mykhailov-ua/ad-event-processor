package ingestion

import (
	"github.com/bidshard/ad-event-processor/internal/domain"
)

func eventTypeUsesBrandLanding(eventType string) bool {
	switch eventType {
	case "click", "tg_click":
		return true
	default:
		return false
	}
}

func ResolveLandingURL(registry domain.CampaignRegistry, store *BrandCreativeStore, evt *domain.Event) string {
	return unsafeString(ResolveLandingURLBytes(registry, store, evt))
}

func ResolveLandingURLBytes(registry domain.CampaignRegistry, store *BrandCreativeStore, evt *domain.Event) []byte {
	if store == nil || registry == nil || !eventTypeUsesBrandLanding(evt.Type) {
		return nil
	}
	camp, ok := registry.GetCampaign(evt.CampaignID)
	if !ok || camp.BrandID == nil {
		return nil
	}
	return store.SelectLandingURLBytes(*camp.BrandID, evt.UserID, evt)
}
