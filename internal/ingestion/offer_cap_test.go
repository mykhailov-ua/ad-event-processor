package ingestion

import (
	"fmt"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestOfferIsCapped_holdout(t *testing.T) {
	t.Parallel()
	offerID := uuid.New()
	capDaily := int32(10)
	capTotal := int32(100)
	counts := map[uuid.UUID]offerConversionCounts{
		offerID: {daily: 10, total: 50},
	}
	assert.True(t, offerIsCapped(offerID, &capDaily, nil, counts))
	assert.False(t, offerIsCapped(offerID, &capDaily, nil, map[uuid.UUID]offerConversionCounts{
		offerID: {daily: 9, total: 50},
	}))
	assert.True(t, offerIsCapped(offerID, nil, &capTotal, map[uuid.UUID]offerConversionCounts{
		offerID: {daily: 1, total: 100},
	}))
	assert.False(t, offerIsCapped(offerID, &capDaily, &capTotal, nil))
}

func TestSelectSnapshot_skipsCappedOffer(t *testing.T) {
	t.Parallel()
	landerA := uuid.MustParse("00000000-0000-4000-8000-000000000001")
	offerA := uuid.MustParse("00000000-0000-4000-8000-000000000101")
	offerB := uuid.MustParse("00000000-0000-4000-8000-000000000102")
	snap := &FlowPathSnapshot{
		Paths: []FlowPath{{
			Weight: 100,
			Landers: []FlowLanderEntry{
				{LanderID: landerA, Weight: 100, URL: []byte("https://lander.test/lp")},
			},
			Offers: []FlowOfferEntry{
				{OfferID: offerA, Weight: 100, Capped: true},
				{OfferID: offerB, Weight: 100, Capped: false},
			},
		}},
	}
	for i := 0; i < 500; i++ {
		sel, _, ok := SelectSnapshot(snap, []byte(fmt.Sprintf("cap-user-%d", i)), FlowSelectContext{})
		require.True(t, ok)
		assert.Equal(t, offerB, sel.OfferID)
	}
}

func TestBuildFlowSnapshot_offerCapFromCounts(t *testing.T) {
	t.Parallel()
	offerID := uuid.MustParse("00000000-0000-4000-8000-000000000022")
	landerID := uuid.MustParse("00000000-0000-4000-8000-000000000011")
	paths := []byte(`[{"weight":100,"landers":[{"lander_id":"00000000-0000-4000-8000-000000000011","weight":100}],"offers":[{"offer_id":"00000000-0000-4000-8000-000000000022","weight":100,"cap_daily":5}]}]`)
	snap, ok := buildFlowSnapshot(paths, map[uuid.UUID][]byte{
		landerID: []byte("https://lander.test/"),
	}, map[uuid.UUID][]byte{
		offerID: []byte("https://offer.test/"),
	}, map[uuid.UUID]offerConversionCounts{
		offerID: {daily: 5, total: 5},
	})
	require.True(t, ok)
	require.Len(t, snap.Paths[0].Offers, 1)
	assert.True(t, snap.Paths[0].Offers[0].Capped)
}
