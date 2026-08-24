package ingestion

import (
	"context"
	"testing"

	"ad-event-processor/internal/domain"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

func TestResolveLandingURLBytes_TgClick(t *testing.T) {
	t.Parallel()
	brandID := uuid.MustParse("660e8400-e29b-41d4-a716-446655440001")
	campID := uuid.MustParse("550e8400-e29b-41d4-a716-446655440000")

	staticCampaignMu.Lock()
	staticCampaign = &domain.Campaign{
		ID:         campID,
		CustomerID: uuid.Nil,
		BrandID:    &brandID,
		Location:   staticCampaign.Location,
	}
	staticCampaignMu.Unlock()
	cachedMockCamp.Store(nil)

	store := NewBrandCreativeStore(nil, 0)
	store.cache.Store(&brandCreativeMapSnapshot{
		byBrand: map[uuid.UUID][]brandCreativeEntry{
			brandID: brandCreativeEntriesReady([]brandCreativeEntry{{
				URL:    "https://offer.example/lp",
				Weight: 100,
			}}),
		},
	})

	evt := &domain.Event{
		Type:       "tg_click",
		CampaignID: campID,
		UserID:     "u1",
	}
	got := ResolveLandingURLBytes(context.Background(), &mockRegistry{}, store, evt)
	require.Equal(t, []byte("https://offer.example/lp"), got)
}
