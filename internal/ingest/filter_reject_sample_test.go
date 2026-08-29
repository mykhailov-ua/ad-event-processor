package ingest

import (
	"sync/atomic"
	"testing"

	"ad-event-processor/internal/domain"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNormalizeRejectCountry_holdout(t *testing.T) {
	t.Parallel()
	assert.Equal(t, "US", normalizeRejectCountry("US"))
	assert.Equal(t, "", normalizeRejectCountry("usa"))
	assert.Equal(t, "", normalizeRejectCountry(""))
}

func TestAppendRejectSamplePayload_holdout(t *testing.T) {
	t.Parallel()
	got := appendRejectSamplePayload(nil, "geo", "zone_1", "DE")
	assert.Equal(t, `{"k":"geo","p":"zone_1","c":"DE"}`, string(got))
}

func TestRecordFilterRejectCountrySample_sampled(t *testing.T) {
	t.Parallel()
	evt := &domain.Event{GeoCountry: "US"}
	var seq atomic.Uint64
	recordFilterRejectCountrySample(filterRejectGeo, evt, &seq, 0)
}

func TestWriteFilterRejectSample_buildsAuditEvent(t *testing.T) {
	t.Parallel()
	evt := &domain.Event{
		CampaignID:  uuid.New(),
		PlacementID: "zone_42",
		GeoCountry:  "DE",
	}
	payload := appendRejectSamplePayload(nil, "geo", evt.PlacementID, evt.GeoCountry)
	require.Contains(t, string(payload), `"k":"geo"`)
	require.Contains(t, string(payload), `"p":"zone_42"`)
	require.Contains(t, string(payload), `"c":"DE"`)
}
