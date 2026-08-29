package filter

import (
	"context"

	"ad-event-processor/internal/domain"
)

type ConsentFilter struct {
	registry domain.CampaignRegistry
	store    *domain.ConsentStore
}

func NewConsentFilter(registry domain.CampaignRegistry, store *domain.ConsentStore) *ConsentFilter {
	return &ConsentFilter{registry: registry, store: store}
}

func (f *ConsentFilter) Check(ctx context.Context, evt *domain.Event) error {
	if f == nil || f.store == nil || evt == nil {
		return nil
	}
	camp, ok := GetCampaignFromEvent(f.registry, evt)
	if !ok || camp.RequireConsentPurposes == 0 {
		return nil
	}
	if evt.UserID == "" {
		return ErrConsentDenied
	}
	userPurposes := f.store.PurposesForUser(evt.UserID)
	if (userPurposes & camp.RequireConsentPurposes) != camp.RequireConsentPurposes {
		return ErrConsentDenied
	}
	return nil
}
