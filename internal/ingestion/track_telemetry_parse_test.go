package ingestion

import (
	"context"
	"strings"
	"testing"

	"ad-event-processor/internal/domain"
	"ad-event-processor/internal/metrics"

	"github.com/google/uuid"
	"github.com/prometheus/client_golang/prometheus/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func trackJSONWithTelemetry(eventsJSON string) []byte {
	return []byte(`{"type":"conversion","campaign_id":"a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a11","user_id":"u1","telemetry":{"events":[` + eventsJSON + `]}}`)
}

func TestParseTrackTelemetry_holdoutParsesMouseEvents(t *testing.T) {
	body := trackJSONWithTelemetry(`{"t":"mousemove","ts":100,"x":10,"y":20},{"t":"mousemove","ts":110,"x":15,"y":25}`)
	var req TrackRequest
	require.NoError(t, ParseTrackRequestJSONOpt(&req, body))
	require.Equal(t, uint8(1), req.TelemetrySet)
	require.Len(t, req.TelemetryEvents, 2)
	assert.Equal(t, "mousemove", req.TelemetryEvents[0].T)
	assert.Equal(t, int64(100), req.TelemetryEvents[0].TS)
	assert.Equal(t, 10, req.TelemetryEvents[0].X)
	assert.Equal(t, 20, req.TelemetryEvents[0].Y)
}

func TestParseTrackTelemetry_reusesScratchZeroAlloc(t *testing.T) {
	body := trackJSONWithTelemetry(`{"t":"mousemove","ts":100,"x":10,"y":20},{"t":"mousemove","ts":110,"x":15,"y":25}`)
	var req TrackRequest
	require.NoError(t, ParseTrackRequestJSONOpt(&req, body))
	allocs := testing.AllocsPerRun(100, func() {
		_ = ParseTrackRequestJSONOpt(&req, body)
	})
	assert.Equal(t, float64(0), allocs)
}

func TestParseTrackTelemetry_holdoutEmptyObjectPasses(t *testing.T) {
	body := []byte(`{"type":"click","campaign_id":"a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a11","telemetry":{}}`)
	var req TrackRequest
	require.NoError(t, ParseTrackRequestJSONOpt(&req, body))
	assert.Equal(t, uint8(1), req.TelemetrySet)
	assert.Empty(t, req.TelemetryEvents)
}

func TestParseTrackTelemetry_holdoutEventCapFails(t *testing.T) {
	var b strings.Builder
	b.WriteString(`{"type":"click","campaign_id":"a0eebc99-9c0b-4ef8-bb6d-6bb9bd380a11","telemetry":{"events":[`)
	for i := 0; i < trackTelemetryMaxEvents+1; i++ {
		if i > 0 {
			b.WriteByte(',')
		}
		b.WriteString(`{"t":"mousemove","ts":1,"x":1,"y":1}`)
	}
	b.WriteString(`]}}`)
	var req TrackRequest
	err := ParseTrackRequestJSONOpt(&req, []byte(b.String()))
	require.Error(t, err)
}

func TestBehaviorTelemetryFilter_holdoutMissingOnConversion(t *testing.T) {
	before := testutil.ToFloat64(metrics.BehaviorTelemetryMissingTotal)
	reg := &Registry{}
	campID := uuid.New()
	reg.manuallyAdded = map[uuid.UUID]bool{campID: true}
	reg.storeCampaignSnapshot(&campaignMapSnapshot{
		byID: map[uuid.UUID]campaignInfo{
			campID: {campaign: &domain.Campaign{
				ID:                 campID,
				SafePageEnabled:    true,
				AttestationEnabled: true,
			}},
		},
	})
	f := NewBehaviorTelemetryFilter(reg)
	f.SetEnabled(true)

	evt := domain.EventPool.Get().(*domain.Event)
	defer domain.EventPool.Put(evt)
	evt.Reset()
	acc := attachFraudAccumulator(evt)
	defer releaseFraudAccumulator(evt, acc)
	evt.CampaignID = campID
	evt.Type = "conversion"
	evt.UA = "Mozilla/5.0 (Windows NT 10.0; Win64; x64)"

	require.NoError(t, f.Check(context.Background(), evt))
	assert.True(t, acc.has(FraudReasonBehaviorTelemetryMissing))
	assert.Equal(t, before+1, testutil.ToFloat64(metrics.BehaviorTelemetryMissingTotal))
}

func TestBehaviorTelemetryFilter_holdoutClickSkips(t *testing.T) {
	reg := &Registry{}
	campID := uuid.New()
	reg.manuallyAdded = map[uuid.UUID]bool{campID: true}
	reg.storeCampaignSnapshot(&campaignMapSnapshot{
		byID: map[uuid.UUID]campaignInfo{
			campID: {campaign: &domain.Campaign{
				ID:                 campID,
				SafePageEnabled:    true,
				AttestationEnabled: true,
			}},
		},
	})
	f := NewBehaviorTelemetryFilter(reg)
	f.SetEnabled(true)

	evt := domain.EventPool.Get().(*domain.Event)
	defer domain.EventPool.Put(evt)
	evt.Reset()
	acc := attachFraudAccumulator(evt)
	defer releaseFraudAccumulator(evt, acc)
	evt.CampaignID = campID
	evt.Type = "click"
	evt.UA = "Mozilla/5.0 (Windows NT 10.0; Win64; x64)"

	require.NoError(t, f.Check(context.Background(), evt))
	assert.False(t, acc.has(FraudReasonBehaviorTelemetryMissing))
}

func TestBehaviorTelemetryFilter_holdoutHumanCurvePasses(t *testing.T) {
	reg := &Registry{}
	campID := uuid.New()
	reg.manuallyAdded = map[uuid.UUID]bool{campID: true}
	reg.storeCampaignSnapshot(&campaignMapSnapshot{
		byID: map[uuid.UUID]campaignInfo{
			campID: {campaign: &domain.Campaign{
				ID:                 campID,
				SafePageEnabled:    true,
				AttestationEnabled: true,
			}},
		},
	})
	f := NewBehaviorTelemetryFilter(reg)
	f.SetEnabled(true)

	evt := domain.EventPool.Get().(*domain.Event)
	defer domain.EventPool.Put(evt)
	evt.Reset()
	acc := attachFraudAccumulator(evt)
	defer releaseFraudAccumulator(evt, acc)
	evt.CampaignID = campID
	evt.Type = "conversion"
	evt.UA = "Mozilla/5.0 (Windows NT 10.0; Win64; x64)"
	evt.TelemetrySet = 1
	for _, e := range humanMouseEvents(12) {
		evt.TelemetryEvents = append(evt.TelemetryEvents, domain.BehaviorTelemetryEvent{
			T: e.T, TS: e.TS, X: e.X, Y: e.Y,
		})
	}

	require.NoError(t, f.Check(context.Background(), evt))
	assert.False(t, acc.has(FraudReasonBehaviorTelemetryMissing))
	assert.False(t, acc.has(FraudReasonBehaviorBezierBot))
}

func TestBehaviorTelemetryFilter_holdoutLinearMouseFlagsBezier(t *testing.T) {
	before := testutil.ToFloat64(metrics.BehaviorBezierBotTotal)
	reg := &Registry{}
	campID := uuid.New()
	reg.manuallyAdded = map[uuid.UUID]bool{campID: true}
	reg.storeCampaignSnapshot(&campaignMapSnapshot{
		byID: map[uuid.UUID]campaignInfo{
			campID: {campaign: &domain.Campaign{
				ID:                 campID,
				SafePageEnabled:    true,
				AttestationEnabled: true,
			}},
		},
	})
	f := NewBehaviorTelemetryFilter(reg)
	f.SetEnabled(true)

	evt := domain.EventPool.Get().(*domain.Event)
	defer domain.EventPool.Put(evt)
	evt.Reset()
	acc := attachFraudAccumulator(evt)
	defer releaseFraudAccumulator(evt, acc)
	evt.CampaignID = campID
	evt.Type = "conversion"
	evt.UA = "Mozilla/5.0 (Windows NT 10.0; Win64; x64)"
	evt.TelemetrySet = 1
	for _, e := range linearMouseEvents(12) {
		evt.TelemetryEvents = append(evt.TelemetryEvents, domain.BehaviorTelemetryEvent{
			T: e.T, TS: e.TS, X: e.X, Y: e.Y,
		})
	}

	require.NoError(t, f.Check(context.Background(), evt))
	assert.False(t, acc.has(FraudReasonBehaviorTelemetryMissing))
	assert.True(t, acc.has(FraudReasonBehaviorBezierBot))
	assert.Equal(t, before+1, testutil.ToFloat64(metrics.BehaviorBezierBotTotal))
}
