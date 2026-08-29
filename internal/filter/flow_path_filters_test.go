package filter

import (
	"fmt"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestFlowPathFiltersMatch_holdout(t *testing.T) {
	t.Parallel()
	filters := FlowPathFilters{
		Countries: [][2]byte{{'U', 'S'}},
		Devices:   flowDeviceMobile,
		OSMask:    flowOSAndroid,
		Languages: [][2]byte{{'e', 'n'}},
	}
	ctx := FlowSelectContext{
		Country:    [2]byte{'U', 'S'},
		DeviceMask: flowDeviceMobile,
		OSMask:     flowOSAndroid,
		Language:   [2]byte{'e', 'n'},
	}
	assert.True(t, flowPathFiltersMatch(filters, ctx))
	assert.False(t, flowPathFiltersMatch(filters, FlowSelectContext{Country: [2]byte{'D', 'E'}}))
}

func TestSelectSnapshot_pathGeoFilter(t *testing.T) {
	t.Parallel()
	landerUS := uuid.MustParse("00000000-0000-4000-8000-000000000001")
	landerDE := uuid.MustParse("00000000-0000-4000-8000-000000000002")
	offerA := uuid.MustParse("00000000-0000-4000-8000-000000000101")
	snap := &FlowPathSnapshot{
		Paths: []FlowPath{
			{
				Weight:  100,
				Filters: FlowPathFilters{Countries: [][2]byte{{'U', 'S'}}},
				Landers: []FlowLanderEntry{{LanderID: landerUS, Weight: 100, URL: []byte("https://us.test/lp")}},
				Offers:  []FlowOfferEntry{{OfferID: offerA, Weight: 100}},
			},
			{
				Weight:  100,
				Filters: FlowPathFilters{Countries: [][2]byte{{'D', 'E'}}},
				Landers: []FlowLanderEntry{{LanderID: landerDE, Weight: 100, URL: []byte("https://de.test/lp")}},
				Offers:  []FlowOfferEntry{{OfferID: offerA, Weight: 100}},
			},
		},
	}
	usCtx := FlowSelectContext{Country: [2]byte{'U', 'S'}}
	deCtx := FlowSelectContext{Country: [2]byte{'D', 'E'}}
	for i := range 100 {
		sel, url, ok := SelectSnapshot(snap, []byte(fmt.Sprintf("geo-user-%d", i)), usCtx)
		require.True(t, ok)
		assert.Equal(t, landerUS, sel.LanderID)
		assert.Contains(t, string(url), "us.test")
		sel, url, ok = SelectSnapshot(snap, []byte(fmt.Sprintf("geo-user-%d", i)), deCtx)
		require.True(t, ok)
		assert.Equal(t, landerDE, sel.LanderID)
		assert.Contains(t, string(url), "de.test")
	}
	_, _, ok := SelectSnapshot(snap, []byte("geo-user-unknown"), FlowSelectContext{})
	assert.False(t, ok)
}

func TestBuildFlowSnapshot_compilesPathFilters(t *testing.T) {
	t.Parallel()
	landerID := uuid.New()
	offerID := uuid.New()
	raw := []byte(`[{"weight":100,"filters":{"countries":["US"],"devices":["mobile"],"os":["android"],"languages":["en"]},"landers":[{"lander_id":"` + landerID.String() + `","weight":100}],"offers":[{"offer_id":"` + offerID.String() + `","weight":100}]}]`)
	snap, ok := buildFlowSnapshot(raw, map[uuid.UUID][]byte{landerID: []byte("https://lander.test/")}, map[uuid.UUID][]byte{offerID: []byte("https://offer.test/")}, nil)
	require.True(t, ok)
	require.Len(t, snap.Paths, 1)
	assert.Equal(t, [2]byte{'U', 'S'}, snap.Paths[0].Filters.Countries[0])
	assert.Equal(t, flowDeviceMobile, snap.Paths[0].Filters.Devices)
	assert.Equal(t, flowOSAndroid, snap.Paths[0].Filters.OSMask)
	assert.Equal(t, [2]byte{'e', 'n'}, snap.Paths[0].Filters.Languages[0])
}

func TestFlowDeviceMaskFromUA_holdout(t *testing.T) {
	t.Parallel()
	assert.Equal(t, flowDeviceMobile, flowDeviceMaskFromUA("Mozilla/5.0 (Linux; Android 13; Pixel)"))
	assert.Equal(t, flowDeviceTablet, flowDeviceMaskFromUA("Mozilla/5.0 (iPad; CPU OS 16_0 like Mac OS X)"))
	assert.Equal(t, flowDeviceDesktop, flowDeviceMaskFromUA("Mozilla/5.0 (Windows NT 10.0; Win64; x64)"))
}
