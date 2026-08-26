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

func BenchmarkGetCampaignFromEvent_hot(b *testing.B) {
	reg := NewRegistry(nil)
	id := uuid.New()
	camp := &domain.Campaign{ID: id, CustomerID: uuid.New(), PacingMode: domain.PacingModeAsap}
	enrichMockCampaign(camp)
	reg.storeCampaignSnapshot(&campaignMapSnapshot{byID: map[uuid.UUID]campaignInfo{
		id: {campaign: camp},
	}})
	evt := &domain.Event{CampaignID: id, FilterWorkerIdx: 4}
	for range 1000 {
		_, _ = getCampaignFromEvent(reg, evt)
	}
	b.ReportAllocs()
	for b.Loop() {
		_, _ = getCampaignFromEvent(reg, evt)
	}
}

func BenchmarkFilterChain_getCampaign_mapLoop(b *testing.B) {
	reg := NewRegistry(nil)
	id := uuid.New()
	camp := &domain.Campaign{ID: id, CustomerID: uuid.New(), PacingMode: domain.PacingModeAsap}
	enrichMockCampaign(camp)
	reg.storeCampaignSnapshot(&campaignMapSnapshot{byID: map[uuid.UUID]campaignInfo{
		id: {campaign: camp},
	}})
	b.ReportAllocs()
	for b.Loop() {
		for range 6 {
			_, _ = reg.GetCampaign(id)
		}
	}
}

func BenchmarkFilterChain_getCampaignFromEvent_loop(b *testing.B) {
	reg := NewRegistry(nil)
	id := uuid.New()
	camp := &domain.Campaign{ID: id, CustomerID: uuid.New(), PacingMode: domain.PacingModeAsap}
	enrichMockCampaign(camp)
	reg.storeCampaignSnapshot(&campaignMapSnapshot{byID: map[uuid.UUID]campaignInfo{
		id: {campaign: camp},
	}})
	evt := &domain.Event{CampaignID: id, FilterWorkerIdx: 4}
	for range 1000 {
		for range 6 {
			_, _ = getCampaignFromEvent(reg, evt)
		}
	}
	b.ReportAllocs()
	for b.Loop() {
		for range 6 {
			_, _ = getCampaignFromEvent(reg, evt)
		}
	}
}
