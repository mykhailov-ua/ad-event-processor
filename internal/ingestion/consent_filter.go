package ingestion

import (
	"context"
	"errors"

	"espx/internal/campaignmodel"
)

var ErrConsentDenied = errors.New("consent not granted")

type ConsentFilter struct {
	registry campaignmodel.CampaignRegistry
	store    *ConsentStore
}

func NewConsentFilter(registry campaignmodel.CampaignRegistry, store *ConsentStore) *ConsentFilter {
	return &ConsentFilter{registry: registry, store: store}
}

func (f *ConsentFilter) Check(ctx context.Context, evt *campaignmodel.Event) error {
	if f == nil || f.store == nil || evt == nil {
		return nil
	}
	camp, ok := f.registry.GetCampaign(evt.CampaignID)
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
