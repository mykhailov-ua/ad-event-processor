package ingest

import (
	"context"
	"testing"

	"ad-event-processor/internal/domain"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestConsentFilter_blocksMissingPurposes(t *testing.T) {
	campID := uuid.New()
	registry := NewRegistry(nil)
	registry.SeedCampaignForTest(&domain.Campaign{
		ID:                     campID,
		RequireConsentPurposes: ConsentPurposeAdStorage,
	})

	store := NewConsentStore(nil)
	filter := NewConsentFilter(registry, store)
	evt := &domain.Event{CampaignID: campID, UserID: "user-no-consent", Type: "click"}
	err := filter.Check(context.Background(), evt)
	require.ErrorIs(t, err, ErrConsentDenied)
}

func TestConsentFilter_allowsGrantedPurposes(t *testing.T) {
	campID := uuid.New()
	registry := NewRegistry(nil)
	registry.SeedCampaignForTest(&domain.Campaign{
		ID:                     campID,
		RequireConsentPurposes: ConsentPurposeAdStorage,
	})

	store := NewConsentStore(nil)
	hashHex := HashUserIDHex("user-ok")
	store.UpsertLocal(hashHex, ConsentPurposeAdStorage)
	filter := NewConsentFilter(registry, store)
	evt := &domain.Event{CampaignID: campID, UserID: "user-ok", Type: "click"}
	assert.NoError(t, filter.Check(context.Background(), evt))
}

func TestClassifyFilterErr_consentDenied(t *testing.T) {
	t.Parallel()
	kind, ok := classifyFilterErr(ErrConsentDenied)
	require.True(t, ok)
	assert.Equal(t, filterRejectConsent, kind)
	assert.Equal(t, 204, filterRejectSpecs[kind].status)
}
