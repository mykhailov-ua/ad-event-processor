package ingest

import (
	"context"
	"fmt"
	"testing"

	"ad-event-processor/internal/database"
	"ad-event-processor/internal/domain"
	"ad-event-processor/internal/filter"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRotationSeenRedis_unseenThenFixOn_integration(t *testing.T) {
	if testing.Short() {
		t.Skip("integration: run make test-integration (Docker testcontainers Redis)")
	}

	rdb, cleanup := database.SetupTestRedis(t)
	defer cleanup()

	campaignID := uuid.New()
	visitor := "integration-visitor-1"
	landerA := uuid.MustParse("00000000-0000-4000-8000-000000000001")
	landerB := uuid.MustParse("00000000-0000-4000-8000-000000000002")
	offerID := uuid.MustParse("00000000-0000-4000-8000-000000000101")

	unseenSnap := &filter.FlowPathSnapshot{
		RoutingMode: domain.FlowRoutingModeWeighted,
		Paths: []filter.FlowPath{{
			Weight:       100,
			RotationMode: string(domain.RotationModeUnseen),
			Landers: []filter.FlowLanderEntry{
				{LanderID: landerA, Weight: 50, URL: []byte("https://a.test/")},
				{LanderID: landerB, Weight: 50, URL: []byte("https://b.test/")},
			},
			Offers: []filter.FlowOfferEntry{{OfferID: offerID, Weight: 100}},
		}},
	}
	rot := &filter.RotationSelectContext{
		CampaignID: campaignID,
		VisitorKey: visitor,
		Redis:      rdb,
	}
	ctx := filter.FlowSelectContext{Rotation: rot}

	sel1, _, ok := filter.SelectSnapshot(unseenSnap, []byte("bucket-a"), ctx)
	require.True(t, ok)
	sel2, _, ok := filter.SelectSnapshot(unseenSnap, []byte("bucket-b"), ctx)
	require.True(t, ok)
	assert.NotEqual(t, sel1.LanderID, sel2.LanderID)

	key := filter.RotationSeenRedisKey(campaignID, visitor)
	members, err := rdb.SMembers(context.Background(), key).Result()
	require.NoError(t, err)
	require.NotEmpty(t, members)

	fixOnSnap := &filter.FlowPathSnapshot{
		RoutingMode: domain.FlowRoutingModeWeighted,
		Paths: []filter.FlowPath{{
			Weight:       100,
			RotationMode: string(domain.RotationModeFixOn),
			Landers: []filter.FlowLanderEntry{
				{LanderID: landerA, Weight: 50, URL: []byte("https://a.test/")},
				{LanderID: landerB, Weight: 50, URL: []byte("https://b.test/")},
			},
			Offers: []filter.FlowOfferEntry{{OfferID: offerID, Weight: 100}},
		}},
	}
	visitorFix := "integration-visitor-fixon"
	rotFix := &filter.RotationSelectContext{
		CampaignID: campaignID,
		VisitorKey: visitorFix,
		Redis:      rdb,
	}
	ctxFix := filter.FlowSelectContext{Rotation: rotFix}

	pin1, _, ok := filter.SelectSnapshot(fixOnSnap, []byte("bucket-x"), ctxFix)
	require.True(t, ok)
	pin2, _, ok := filter.SelectSnapshot(fixOnSnap, []byte("bucket-y"), ctxFix)
	require.True(t, ok)
	assert.Equal(t, pin1.LanderID, pin2.LanderID)

	fixKey := filter.RotationSeenRedisKey(campaignID, visitorFix)
	ttl, err := rdb.TTL(context.Background(), fixKey).Result()
	require.NoError(t, err)
	assert.Greater(t, ttl.Seconds(), float64(0), "rotation seen key TTL: %s", fmt.Sprintf("%v", ttl))
}
