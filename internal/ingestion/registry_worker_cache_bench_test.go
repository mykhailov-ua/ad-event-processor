//go:build !race

package ingestion

import (
	"testing"

	"github.com/bidshard/ad-event-processor/internal/domain"

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
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
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
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = reg.GetCampaign(id)
	}
}
