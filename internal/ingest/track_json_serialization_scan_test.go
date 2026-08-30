package ingest

import (
	"context"
	"testing"

	"ad-event-processor/internal/domain"
	"ad-event-processor/internal/metrics"

	"github.com/google/uuid"
	"github.com/prometheus/client_golang/prometheus/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func browserOrderTrackJSON() []byte {
	return []byte(`{"type":"click","campaign_id":"a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a11","user_id":"u1","click_id":"c1","payload":{"slot":"top"}}`)
}

func sortedKeyTrackJSON() []byte {
	return []byte(`{"campaign_id":"a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a11","click_id":"c1","payload":{"slot":"top"},"type":"click","user_id":"u1"}`)
}

func pythonSpacingTrackJSON() []byte {
	return []byte(`{"type": "click", "campaign_id": "a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a11"}`)
}

func longTimestampTrackJSON() []byte {
	return []byte(`{"type":"click","campaign_id":"a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a11","ts":1234567890123456}`)
}

func TestTrackJSONSerialization_holdoutBrowserOrderPasses(t *testing.T) {
	flags := scanTrackJSONSerialization(browserOrderTrackJSON())
	assert.Equal(t, uint8(0), flags)
}

func TestTrackJSONSerialization_holdoutSortedKeysFails(t *testing.T) {
	flags := scanTrackJSONSerialization(sortedKeyTrackJSON())
	assert.True(t, flags&jsonSerFlagSortedKeys != 0)
}

func TestTrackJSONSerialization_holdoutPythonSpacingFails(t *testing.T) {
	flags := scanTrackJSONSerialization(pythonSpacingTrackJSON())
	assert.True(t, flags&jsonSerFlagPythonSpacing != 0)
}

func TestTrackJSONSerialization_holdoutLongTimestampFails(t *testing.T) {
	flags := scanTrackJSONSerialization(longTimestampTrackJSON())
	assert.True(t, flags&jsonSerFlagLongTimestamp != 0)
}

func TestTrackJSONSerialization_protobufPathUnscanned(t *testing.T) {
	flags := scanTrackJSONSerialization(nil)
	assert.Equal(t, uint8(0), flags)
}

func TestParseTrackRequestJSON_serializationScanParity(t *testing.T) {
	var req TrackRequest
	require.NoError(t, ParseTrackRequestJSONOpt(&req, sortedKeyTrackJSON()))
	req.JSONSerializationFlags = scanTrackJSONSerialization(sortedKeyTrackJSON())
	assert.True(t, req.JSONSerializationFlags&jsonSerFlagSortedKeys != 0)
}

func TestJSONSerializationFilter_holdoutCampaignDisabledFailOpen(t *testing.T) {
	reg := &Registry{}
	campID := uuid.New()
	reg.SeedCampaignForTest(&domain.Campaign{ID: campID})
	f := NewJSONSerializationFilter(reg)
	f.SetEnabled(true)

	evt := domain.EventPool.Get().(*domain.Event)
	defer domain.EventPool.Put(evt)
	evt.Reset()
	acc := attachFraudAccumulator(evt)
	defer releaseFraudAccumulator(evt, acc)

	evt.CampaignID = uuid.New()
	evt.JSONSerializationFlags = jsonSerFlagSortedKeys

	require.NoError(t, f.Check(context.Background(), evt))
	assert.False(t, acc.Has(FraudReasonJSONSerializationBot))
}

func TestJSONSerializationFilter_holdoutEnabledFlagsSignal(t *testing.T) {
	before := testutil.ToFloat64(metrics.JSONSerializationBotTotal)
	campID := uuid.New()
	reg := &Registry{}
	reg.SeedCampaignForTest(&domain.Campaign{
		ID:                       campID,
		JSONSerializationEnabled: true,
	})
	f := NewJSONSerializationFilter(reg)
	f.SetEnabled(true)

	evt := domain.EventPool.Get().(*domain.Event)
	defer domain.EventPool.Put(evt)
	evt.Reset()
	acc := attachFraudAccumulator(evt)
	defer releaseFraudAccumulator(evt, acc)

	evt.CampaignID = campID
	evt.JSONSerializationFlags = jsonSerFlagSortedKeys

	require.NoError(t, f.Check(context.Background(), evt))
	assert.True(t, acc.Has(FraudReasonJSONSerializationBot))
	assert.Equal(t, before+1, testutil.ToFloat64(metrics.JSONSerializationBotTotal))
}

func TestJSONSerializationBotMetric_registered(t *testing.T) {
	require.NotNil(t, metrics.JSONSerializationBotTotal)
}
