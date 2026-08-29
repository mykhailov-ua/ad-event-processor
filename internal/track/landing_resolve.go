package track

import (
	"context"

	"ad-event-processor/internal/domain"
	"ad-event-processor/internal/filter"
)

func EventTypeUsesBrandLanding(eventType string) bool {
	switch eventType {
	case "click", "tg_click":
		return true
	default:
		return false
	}
}

func ResolveLandingURL(ctx context.Context, registry domain.CampaignRegistry, store BrandStore, evt *domain.Event) string {
	return filter.UnsafeString(ResolveLandingURLBytes(ctx, registry, store, evt))
}

func ResolveLandingURLBytes(ctx context.Context, registry domain.CampaignRegistry, store BrandStore, evt *domain.Event) []byte {
	if store == nil || registry == nil || !EventTypeUsesBrandLanding(evt.Type) {
		return nil
	}
	camp, ok := filter.GetCampaignFromEvent(registry, evt)
	if !ok || camp.BrandID == nil {
		return nil
	}
	return store.SelectLandingURLBytes(ctx, *camp.BrandID, evt.UserID, evt)
}
