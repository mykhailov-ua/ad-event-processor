package ingestion

import (
	"context"

	"ad-event-processor/internal/domain"
)

func eventTypeUsesBrandLanding(eventType string) bool {
	switch eventType {
	case "click", "tg_click":
		return true
	default:
		return false
	}
}

func ResolveLandingURL(ctx context.Context, registry domain.CampaignRegistry, store *BrandCreativeStore, evt *domain.Event) string {
	return unsafeString(ResolveLandingURLBytes(ctx, registry, store, evt))
}

func ResolveLandingURLBytes(ctx context.Context, registry domain.CampaignRegistry, store *BrandCreativeStore, evt *domain.Event) []byte {
	if store == nil || registry == nil || !eventTypeUsesBrandLanding(evt.Type) {
		return nil
	}
	camp, ok := getCampaignFromEvent(registry, evt)
	if !ok || camp.BrandID == nil {
		return nil
	}
	return store.SelectLandingURLBytes(ctx, *camp.BrandID, evt.UserID, evt)
}
