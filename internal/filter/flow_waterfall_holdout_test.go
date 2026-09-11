package filter

import (
	"testing"

	"ad-event-processor/internal/domain"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSelectSnapshot_waterfallPicksLowestPriority_holdout(t *testing.T) {
	t.Parallel()
	landerA := uuid.MustParse("00000000-0000-4000-8000-000000000001")
	offerHigh := uuid.MustParse("00000000-0000-4000-8000-000000000101")
	offerLow := uuid.MustParse("00000000-0000-4000-8000-000000000102")
	snap := &FlowPathSnapshot{
		RoutingMode: domain.FlowRoutingModeWaterfall,
		Paths: []FlowPath{{
			Weight: 100,
			Landers: []FlowLanderEntry{
				{LanderID: landerA, Weight: 100, URL: []byte("https://lander.test/lp")},
			},
			Offers: []FlowOfferEntry{
				{OfferID: offerHigh, Weight: 100, Priority: 10},
				{OfferID: offerLow, Weight: 100, Priority: 1},
			},
		}},
	}
	sel, _, ok := SelectSnapshot(snap, []byte("waterfall-user"), FlowSelectContext{})
	require.True(t, ok)
	assert.Equal(t, offerLow, sel.OfferID)
}

func TestSelectSnapshot_waterfallThenLanding_usesLanderWhenOffersCapped_holdout(t *testing.T) {
	t.Parallel()
	landerA := uuid.MustParse("00000000-0000-4000-8000-000000000001")
	offerA := uuid.MustParse("00000000-0000-4000-8000-000000000101")
	snap := &FlowPathSnapshot{
		RoutingMode: domain.FlowRoutingModeWaterfallThenLanding,
		Paths: []FlowPath{{
			Weight: 100,
			Landers: []FlowLanderEntry{
				{LanderID: landerA, Weight: 100, URL: []byte("https://lander.test/lp")},
			},
			Offers: []FlowOfferEntry{
				{OfferID: offerA, Weight: 100, Priority: 1, Capped: true},
			},
		}},
	}
	sel, url, ok := SelectSnapshot(snap, []byte("waterfall-landing-user"), FlowSelectContext{})
	require.True(t, ok)
	require.Equal(t, uuid.Nil, sel.OfferID)
	require.Equal(t, landerA, sel.LanderID)
	require.Equal(t, "https://lander.test/lp", string(url))
}

func TestSelectSnapshot_waterfallThenLanding_holdout_plainWaterfallFailsWhenAllCapped(t *testing.T) {
	t.Parallel()
	landerA := uuid.MustParse("00000000-0000-4000-8000-000000000001")
	offerA := uuid.MustParse("00000000-0000-4000-8000-000000000101")
	snap := &FlowPathSnapshot{
		RoutingMode: domain.FlowRoutingModeWaterfall,
		Paths: []FlowPath{{
			Weight: 100,
			Landers: []FlowLanderEntry{
				{LanderID: landerA, Weight: 100, URL: []byte("https://lander.test/lp")},
			},
			Offers: []FlowOfferEntry{
				{OfferID: offerA, Weight: 100, Priority: 1, Capped: true},
			},
		}},
	}
	_, _, ok := SelectSnapshot(snap, []byte("waterfall-only-user"), FlowSelectContext{})
	require.False(t, ok)
}

func TestSelectSnapshot_waterfallSkipsCapped_holdout(t *testing.T) {
	t.Parallel()
	landerA := uuid.MustParse("00000000-0000-4000-8000-000000000001")
	offerA := uuid.MustParse("00000000-0000-4000-8000-000000000101")
	offerB := uuid.MustParse("00000000-0000-4000-8000-000000000102")
	snap := &FlowPathSnapshot{
		RoutingMode: domain.FlowRoutingModeWaterfall,
		Paths: []FlowPath{{
			Weight: 100,
			Landers: []FlowLanderEntry{
				{LanderID: landerA, Weight: 100, URL: []byte("https://lander.test/lp")},
			},
			Offers: []FlowOfferEntry{
				{OfferID: offerA, Weight: 100, Priority: 1, Capped: true},
				{OfferID: offerB, Weight: 100, Priority: 2},
			},
		}},
	}
	sel, _, ok := SelectSnapshot(snap, []byte("waterfall-cap"), FlowSelectContext{})
	require.True(t, ok)
	assert.Equal(t, offerB, sel.OfferID)
}
