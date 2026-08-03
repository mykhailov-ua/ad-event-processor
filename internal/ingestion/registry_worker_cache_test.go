package ingestion

import (
	"testing"

	"espx/internal/domain"
	db "espx/internal/domain/db"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

func TestRegistry_GetCampaignWorker_cacheHit(t *testing.T) {
	r := NewRegistry(nil)
	id := uuid.New()
	camp := &domain.Campaign{ID: id, CustomerID: uuid.New(), PacingMode: domain.PacingModeAsap}
	r.storeCampaignSnapshot(&campaignMapSnapshot{byID: map[uuid.UUID]campaignInfo{
		id: {campaign: camp, status: db.CampaignStatusTypeACTIVE},
	}})

	got, ok := r.GetCampaignWorker(3, id)
	require.True(t, ok)
	require.Equal(t, camp, got)

	got2, ok2 := r.GetCampaignWorker(3, id)
	require.True(t, ok2)
	require.Equal(t, camp, got2)
}

func TestRegistry_GetCampaignWorker_invalidatesOnReload(t *testing.T) {
	r := NewRegistry(nil)
	id := uuid.New()
	campV1 := &domain.Campaign{ID: id, DailyBudget: 1}
	r.storeCampaignSnapshot(&campaignMapSnapshot{byID: map[uuid.UUID]campaignInfo{
		id: {campaign: campV1},
	}})
	_, ok := r.GetCampaignWorker(1, id)
	require.True(t, ok)

	campV2 := &domain.Campaign{ID: id, DailyBudget: 2}
	r.storeCampaignSnapshot(&campaignMapSnapshot{byID: map[uuid.UUID]campaignInfo{
		id: {campaign: campV2},
	}})
	got, ok := r.GetCampaignWorker(1, id)
	require.True(t, ok)
	require.Equal(t, int64(2), got.DailyBudget)
}
