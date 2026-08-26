package ingestion

import (
	"context"
	"errors"

	"ad-event-processor/internal/domain"
)

var ErrConsentDenied = errors.New("consent not granted")

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
	camp, ok := getCampaignFromEvent(f.registry, evt)
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
