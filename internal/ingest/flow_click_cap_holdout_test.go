package ingest

import (
	"testing"

	"ad-event-processor/internal/config"
	"ad-event-processor/internal/domain"

	"github.com/alicebob/miniredis/v2"
	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/require"
)

func TestFlowClickCap_secondClickRoutesNextOffer_holdout(t *testing.T) {
	t.Parallel()
	cid := uuid.New()
	landerID := uuid.MustParse("00000000-0000-4000-8000-000000000001")
	offerA := uuid.MustParse("00000000-0000-4000-8000-000000000101")
	offerB := uuid.MustParse("00000000-0000-4000-8000-000000000102")

	table := NewCampaignFlowTable()
	table.Publish(NewCampaignFlowRegistrySnapshot(map[uuid.UUID]FlowPathSnapshot{
		cid: {
			Paths: []FlowPath{{
				Weight: 100,
				Landers: []FlowLanderEntry{
					{LanderID: landerID, Weight: 100, URL: []byte("https://lander.test/lp?cid={click_id}")},
				},
				Offers: []FlowOfferEntry{
					{OfferID: offerA, Weight: 1000, CapClicksDaily: 1},
					{OfferID: offerB, Weight: 1, CapClicksDaily: 0},
				},
			}},
		},
	}))

	mr := miniredis.RunT(t)
	rdb := redis.NewClient(&redis.Options{Addr: mr.Addr()})

	brandID := uuid.New()
	lockStaticCampaign(func(c *domain.Campaign) {
		c.ID = cid
		c.BrandID = &brandID
	})
	t.Cleanup(func() { cachedMockCamp.Store(nil) })

	cfg := &config.Config{MaxRequestBodySize: 1 << 20}
	h := NewAdsPacketHandler(cfg, &mockRegistry{}, nil, nil, []redis.UniversalClient{rdb}, NewJumpHashSharder(1), "fraud-stream", nil)
	h.ConfigureCampaignFlow(table)

	evt := &domain.Event{
		CampaignID: cid,
		UserID:     "cap-user-1",
		ClickID:    "cap-click-1",
		Type:       "click",
	}

	_, sel1, ok1 := h.selectFlowLandingWithClickCaps(evt)
	require.True(t, ok1)
	require.Equal(t, offerA, sel1.OfferID)

	_, sel2, ok2 := h.selectFlowLandingWithClickCaps(evt)
	require.True(t, ok2)
	require.Equal(t, offerB, sel2.OfferID)
}
