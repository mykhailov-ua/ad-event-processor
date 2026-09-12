package filter

import (
	"context"
	"fmt"
	"testing"

	"ad-event-processor/internal/domain"

	"github.com/alicebob/miniredis/v2"
	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSelectUnseenLander_holdoutDifferentUntilPoolExhausted(t *testing.T) {
	t.Parallel()
	landerA := uuid.MustParse("00000000-0000-4000-8000-000000000001")
	landerB := uuid.MustParse("00000000-0000-4000-8000-000000000002")
	landers := []FlowLanderEntry{
		{LanderID: landerA, Weight: 50, URL: []byte("https://a.test/")},
		{LanderID: landerB, Weight: 50, URL: []byte("https://b.test/")},
	}
	seen := map[string]struct{}{}

	idx1, picked1, ok := selectUnseenLander(landers, seen, 1, false)
	require.True(t, ok)
	seen[rotationSeenMember('l', picked1.LanderID)] = struct{}{}

	idx2, picked2, ok := selectUnseenLander(landers, seen, 2, false)
	require.True(t, ok)
	assert.NotEqual(t, picked1.LanderID, picked2.LanderID)
	assert.NotEqual(t, idx1, idx2)
}

func TestSelectSnapshot_unseenRotation_secondClickDifferentLander_holdout(t *testing.T) {
	t.Parallel()
	landerA := uuid.MustParse("00000000-0000-4000-8000-000000000001")
	landerB := uuid.MustParse("00000000-0000-4000-8000-000000000002")
	offerID := uuid.MustParse("00000000-0000-4000-8000-000000000101")
	snap := &FlowPathSnapshot{
		RoutingMode: domain.FlowRoutingModeWeighted,
		Paths: []FlowPath{{
			Weight:       100,
			RotationMode: string(domain.RotationModeUnseen),
			Landers: []FlowLanderEntry{
				{LanderID: landerA, Weight: 50, URL: []byte("https://a.test/")},
				{LanderID: landerB, Weight: 50, URL: []byte("https://b.test/")},
			},
			Offers: []FlowOfferEntry{
				{OfferID: offerID, Weight: 100},
			},
		}},
	}
	mr := miniredis.RunT(t)
	rdb := redis.NewClient(&redis.Options{Addr: mr.Addr()})
	campaignID := uuid.New()
	visitor := "visitor-unseen-1"
	ctx := FlowSelectContext{
		Rotation: &RotationSelectContext{
			CampaignID: campaignID,
			VisitorKey: visitor,
			Redis:      rdb,
		},
	}
	sel1, _, ok := SelectSnapshot(snap, []byte("bucket-1"), ctx)
	require.True(t, ok)
	sel2, _, ok := SelectSnapshot(snap, []byte("bucket-2"), ctx)
	require.True(t, ok)
	assert.NotEqual(t, sel1.LanderID, sel2.LanderID)

	members, err := rdb.SMembers(context.Background(), RotationSeenRedisKey(campaignID, visitor)).Result()
	require.NoError(t, err)
	landerMembers := 0
	for _, member := range members {
		if len(member) > 2 && member[0] == 'l' && member[1] == ':' {
			landerMembers++
		}
	}
	assert.Equal(t, 2, landerMembers)
}

func TestSelectSnapshot_fixOn_pinsLander_holdout(t *testing.T) {
	t.Parallel()
	landerA := uuid.MustParse("00000000-0000-4000-8000-000000000001")
	landerB := uuid.MustParse("00000000-0000-4000-8000-000000000002")
	offerID := uuid.MustParse("00000000-0000-4000-8000-000000000101")
	snap := &FlowPathSnapshot{
		RoutingMode: domain.FlowRoutingModeWeighted,
		Paths: []FlowPath{{
			Weight:       100,
			RotationMode: string(domain.RotationModeFixOn),
			Landers: []FlowLanderEntry{
				{LanderID: landerA, Weight: 50, URL: []byte("https://a.test/")},
				{LanderID: landerB, Weight: 50, URL: []byte("https://b.test/")},
			},
			Offers: []FlowOfferEntry{
				{OfferID: offerID, Weight: 100},
			},
		}},
	}
	mr := miniredis.RunT(t)
	rdb := redis.NewClient(&redis.Options{Addr: mr.Addr()})
	campaignID := uuid.New()
	visitor := "visitor-fix-on-1"
	ctx := FlowSelectContext{
		Rotation: &RotationSelectContext{
			CampaignID: campaignID,
			VisitorKey: visitor,
			Redis:      rdb,
		},
	}
	sel1, _, ok := SelectSnapshot(snap, []byte("bucket-a"), ctx)
	require.True(t, ok)
	sel2, _, ok := SelectSnapshot(snap, []byte("bucket-b"), ctx)
	require.True(t, ok)
	assert.Equal(t, sel1.LanderID, sel2.LanderID)
}

func TestSelectSnapshot_unseenDiffersFromFixOn_holdout(t *testing.T) {
	t.Parallel()
	landerA := uuid.MustParse("00000000-0000-4000-8000-000000000003")
	landerB := uuid.MustParse("00000000-0000-4000-8000-000000000004")
	offerID := uuid.MustParse("00000000-0000-4000-8000-000000000105")
	unseenSnap := &FlowPathSnapshot{
		RoutingMode: domain.FlowRoutingModeWeighted,
		Paths: []FlowPath{{
			Weight:       100,
			RotationMode: string(domain.RotationModeUnseen),
			Landers: []FlowLanderEntry{
				{LanderID: landerA, Weight: 50, URL: []byte("https://a.test/")},
				{LanderID: landerB, Weight: 50, URL: []byte("https://b.test/")},
			},
			Offers: []FlowOfferEntry{{OfferID: offerID, Weight: 100}},
		}},
	}
	fixOnSnap := &FlowPathSnapshot{
		RoutingMode: domain.FlowRoutingModeWeighted,
		Paths: []FlowPath{{
			Weight:       100,
			RotationMode: string(domain.RotationModeFixOn),
			Landers: []FlowLanderEntry{
				{LanderID: landerA, Weight: 50, URL: []byte("https://a.test/")},
				{LanderID: landerB, Weight: 50, URL: []byte("https://b.test/")},
			},
			Offers: []FlowOfferEntry{{OfferID: offerID, Weight: 100}},
		}},
	}
	mr := miniredis.RunT(t)
	rdb := redis.NewClient(&redis.Options{Addr: mr.Addr()})
	campaignID := uuid.New()
	visitor := "visitor-unseen-vs-fix"
	rot := FlowSelectContext{
		Rotation: &RotationSelectContext{
			CampaignID: campaignID,
			VisitorKey: visitor,
			Redis:      rdb,
		},
	}
	unseen1, _, ok := SelectSnapshot(unseenSnap, []byte("bucket-1"), rot)
	require.True(t, ok)
	unseen2, _, ok := SelectSnapshot(unseenSnap, []byte("bucket-2"), rot)
	require.True(t, ok)
	fix1, _, ok := SelectSnapshot(fixOnSnap, []byte("bucket-1"), rot)
	require.True(t, ok)
	fix2, _, ok := SelectSnapshot(fixOnSnap, []byte("bucket-2"), rot)
	require.True(t, ok)
	assert.NotEqual(t, unseen1.LanderID, unseen2.LanderID)
	assert.Equal(t, fix1.LanderID, fix2.LanderID)
}

func TestSelectSnapshot_sequentialPicksFirstUncappedOffer_holdout(t *testing.T) {
	t.Parallel()
	landerID := uuid.MustParse("00000000-0000-4000-8000-000000000006")
	offerA := uuid.MustParse("00000000-0000-4000-8000-000000000201")
	offerB := uuid.MustParse("00000000-0000-4000-8000-000000000202")
	snap := &FlowPathSnapshot{
		RoutingMode: domain.FlowRoutingModeWeighted,
		Paths: []FlowPath{{
			Weight:       100,
			RotationMode: string(domain.RotationModeSequential),
			Landers: []FlowLanderEntry{
				{LanderID: landerID, Weight: 100, URL: []byte("https://sequential.test/")},
			},
			Offers: []FlowOfferEntry{
				{OfferID: offerA, Weight: 100, Capped: true},
				{OfferID: offerB, Weight: 100, Capped: false},
			},
		}},
	}
	for i := 0; i < 100; i++ {
		sel, _, ok := SelectSnapshot(snap, []byte(fmt.Sprintf("sequential-user-%d", i)), FlowSelectContext{})
		require.True(t, ok)
		assert.Equal(t, offerB, sel.OfferID)
		assert.Equal(t, landerID, sel.LanderID)
	}
}
