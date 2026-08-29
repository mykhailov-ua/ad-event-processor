package track

import (
	"encoding/json"
	"testing"

	"ad-event-processor/internal/domain"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestEnrichAnalyticsPayload_holdoutGeoCountryWithoutClientCountry(t *testing.T) {
	t.Parallel()

	evt := &domain.Event{
		CampaignID: uuid.New(),
		Type:       "impression",
		GeoCountry: "US",
		GeoHash:    42,
	}
	out := enrichAnalyticsPayload(evt)
	require.NotEmpty(t, out)

	var fields map[string]string
	require.NoError(t, json.Unmarshal(out, &fields))
	assert.Equal(t, "US", fields["country"])
	assert.Equal(t, "42", fields["geo_hash"])
}

func TestEnrichAnalyticsPayload_holdoutClientCountryWinsOverGeoIP(t *testing.T) {
	t.Parallel()

	evt := &domain.Event{
		CampaignID: uuid.New(),
		Type:       "click",
		GeoCountry: "US",
		Payload:    []byte(`{"country":"DE","sub1":"affiliate"}`),
	}
	dims := extractAnalyticsDimensions(evt)
	assert.Equal(t, "DE", dims.country)
	assert.Equal(t, "affiliate", dims.sub1)

	var fields map[string]string
	require.NoError(t, json.Unmarshal(dims.payload, &fields))
	assert.Equal(t, "DE", fields["country"])
	assert.Equal(t, "affiliate", fields["sub1"])
}

func TestExtractAnalyticsDimensions_deviceTypeFallback(t *testing.T) {
	t.Parallel()

	evt := &domain.Event{
		Payload: []byte(`{"device":"mobile","keyword":"shoes"}`),
	}
	dims := extractAnalyticsDimensions(evt)
	assert.Equal(t, "mobile", dims.deviceType)
	assert.Equal(t, "shoes", dims.keyword)
}

func TestAnalyticsCountryCode_normalizesCase(t *testing.T) {
	t.Parallel()

	assert.Equal(t, "US", analyticsCountryCode("us"))
	assert.Equal(t, "DE", analyticsCountryCode("DE"))
	assert.Equal(t, "", analyticsCountryCode(""))
}
