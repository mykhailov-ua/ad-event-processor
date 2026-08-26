package ingestion

import (
	"context"
	"testing"

	"ad-event-processor/internal/domain"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestEnsureIngestGeo_cachesForGeoFilter(t *testing.T) {
	geo := &countingGeoProvider{country: "US"}
	evt := &domain.Event{IP: "8.8.8.8"}

	ensureIngestGeo(geo, evt)
	require.True(t, evt.IngestGeoResolved)
	assert.Equal(t, "US", evt.GeoCountry)
	assert.Equal(t, GeoHashFromCountry("US"), evt.GeoHash)
	assert.Equal(t, 1, geo.countryCalls)

	ensureIngestGeo(geo, evt)
	assert.Equal(t, 1, geo.countryCalls, "second call must reuse cache")

	f := NewGeoFilter(geo, &mockRegistry{})
	campID := uuid.MustParse("11111111-1111-1111-1111-111111111111")
	staticCampaign.ID = campID
	staticCampaign.TargetCountries = map[string]struct{}{"US": {}}
	t.Cleanup(resetStaticCampaignBaseline)
	evt.CampaignID = campID
	err := f.Check(t.Context(), evt)
	assert.NoError(t, err)
	assert.Equal(t, 1, geo.countryCalls, "GeoFilter must not lookup again")
}

func TestEnsureIngestGeo_cachesAnonymousForFraudFilter(t *testing.T) {
	geo := &countingGeoProvider{country: "US", anonymous: true}
	evt := &domain.Event{IP: "1.1.1.66", StringBuffer: make([]byte, 0, 64)}

	ensureIngestGeo(geo, evt)
	require.True(t, evt.IngestGeoResolved)
	assert.True(t, evt.IngestAnonymous)
	assert.Equal(t, 1, geo.countryCalls)
	assert.Equal(t, 1, geo.anonCalls)

	engine := NewFilterEngine(0, NewFraudFilter(geo))
	err := engine.Check(t.Context(), evt)
	require.NoError(t, err)
	assert.Equal(t, 1, geo.anonCalls, "FraudFilter must reuse ingest anonymous cache")
	assert.Contains(t, evt.FraudReason, FraudReasonCodeDatacenterIP)
}

func TestBrandCreativeStore_selectCreative_weightedZeroAlloc(t *testing.T) {
	store := NewBrandCreativeStore(nil, 0)
	brandID := uuid.MustParse("550e8400-e29b-41d4-a716-446655440000")
	store.cache.Store(&brandCreativeMapSnapshot{byBrand: map[uuid.UUID][]brandCreativeEntry{
		brandID: {
			{ID: "a", URL: "https://a.example", Weight: 70},
			{ID: "b", URL: "https://b.example", Weight: 30},
		},
	}})
	ctx := context.Background()
	for range 100 {
		_ = store.SelectLandingURLBytes(ctx, brandID, "sticky-user", nil)
	}
	avg := testing.AllocsPerRun(100, func() {
		_ = store.SelectLandingURLBytes(ctx, brandID, "sticky-user", nil)
	})
	if avg > 0 {
		t.Fatalf("SelectLandingURLBytes weighted pick allocated %.1f times per run, want 0", avg)
	}
}

func TestParseCategoryMask(t *testing.T) {
	assert.Equal(t, uint64(4), parseCategoryMask([]byte(`{"category_mask":4,"bid_micro":100}`)))
}

type countingGeoProvider struct {
	country      string
	anonymous    bool
	countryCalls int
	anonCalls    int
}

func (c *countingGeoProvider) GetCountry(ip string) (string, error) {
	c.countryCalls++
	return c.country, nil
}

func (c *countingGeoProvider) IsAnonymous(ip string) (bool, error) {
	c.anonCalls++
	return c.anonymous, nil
}

func (c *countingGeoProvider) Close() error { return nil }
