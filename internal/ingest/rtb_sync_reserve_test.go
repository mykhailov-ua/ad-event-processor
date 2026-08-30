package ingest

import (
	"testing"

	"ad-event-processor/internal/config"
	"ad-event-processor/internal/domain"
	"ad-event-processor/internal/rtb"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
)

func TestRtbSync_reserveMicro(t *testing.T) {
	camp := &domain.Campaign{
		ID:           uuid.New(),
		BudgetLimit:  10_000,
		ReserveMicro: 50_000,
		Status:       domain.CampaignStatusActive,
	}
	cfg := &config.Config{ClickAmount: 100}
	input := RtbInputForCampaign(camp, cfg, nil, 0, nil, nil)
	assert.Equal(t, int64(50_000), input.ReserveMicro)
}

func TestEnrichTargetingDeal_pmp(t *testing.T) {
	store := rtb.NewBudgetStore()
	catalog := NewRtbCatalog(store, BudgetAuthorityShadow)
	catalog.UpdateDeals([]rtb.DealData{{
		DealID:     "deal-x",
		FloorMicro: 100,
		GeoMask:    rtb.GeoBitFromHash(GeoHashFromCountry("US")),
		CatMask:    1,
		PacingOpen: rtb.PacingOpen,
		Seats:      2,
	}})
	var dealBuf [64]byte
	copy(dealBuf[:], "deal-x")
	targeting := catalog.EnrichTargetingDeal(RtbTargetingInput{
		DealIDLen:    6,
		DealIDBuf:    dealBuf,
		GeoHash:      GeoHashFromCountry("US"),
		CategoryMask: 1,
		SeatCount:    2,
	})
	assert.Equal(t, rtb.NoBidNone, targeting.DealBlock)
}
