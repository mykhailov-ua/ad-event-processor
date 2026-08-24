//go:build !race

package ingestion

import (
	"testing"

	"ad-event-processor/internal/domain"

	"github.com/google/uuid"
)

func BenchmarkRegistry_GetCampaignWorker_hot(b *testing.B) {
	reg := NewRegistry(nil)
	id := uuid.New()
	camp := &domain.Campaign{ID: id, CustomerID: uuid.New(), PacingMode: domain.PacingModeAsap}
	enrichMockCampaign(camp)
	reg.storeCampaignSnapshot(&campaignMapSnapshot{byID: map[uuid.UUID]campaignInfo{
		id: {campaign: camp},
	}})
	for range 1000 {
		_, _ = reg.GetCampaignWorker(0, id)
	}
	b.ReportAllocs()
	for b.Loop() {
		_, _ = reg.GetCampaignWorker(0, id)
	}
}

func BenchmarkRegistry_GetCampaign_mapLookup(b *testing.B) {
	reg := NewRegistry(nil)
	id := uuid.New()
	camp := &domain.Campaign{ID: id, CustomerID: uuid.New(), PacingMode: domain.PacingModeAsap}
	reg.storeCampaignSnapshot(&campaignMapSnapshot{byID: map[uuid.UUID]campaignInfo{
		id: {campaign: camp},
	}})
	b.ReportAllocs()
	for b.Loop() {
		_, _ = reg.GetCampaign(id)
	}
}
