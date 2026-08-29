package ingest

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"testing"
	"time"

	"ad-event-processor/pkg/faultproof"

	"ad-event-processor/internal/config"
	"ad-event-processor/internal/metrics"

	"github.com/panjf2000/gnet/v2"
	"github.com/prometheus/client_golang/prometheus/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseTrackRequestJSON_DepthCap(t *testing.T) {
	validCID := "550e8400-e29b-41d4-a716-446655440000"
	build := func(depth int) []byte {
		var b strings.Builder
		b.WriteString(`{"campaign_id":"`)
		b.WriteString(validCID)
		b.WriteString(`","payload":`)
		for range depth {
			b.WriteString(`{"a":`)
		}
		b.WriteString(`"leaf"`)
		for range depth {
			b.WriteString(`}`)
		}
		b.WriteString(`}`)
		return []byte(b.String())
	}

	var tr TrackRequest
	require.NoError(t, ParseTrackRequestJSON(&tr, build(MaxJSONDepth)))
	require.ErrorIs(t, ParseTrackRequestJSON(&tr, build(MaxJSONDepth+1)), ErrMalformed)
	require.ErrorIs(t, ParseTrackRequestJSON(&tr, build(1000)), ErrMalformed)
}

func TestParseTrackRequestJSON_Depth1000Under1us(t *testing.T) {
	validCID := "550e8400-e29b-41d4-a716-446655440000"
	var nested strings.Builder
	nested.WriteString(`{"campaign_id":"`)
	nested.WriteString(validCID)
	nested.WriteString(`","payload":`)
	for range 1000 {
		nested.WriteString(`{"a":`)
	}
	nested.WriteString(`"leaf"`)
	for range 1000 {
		nested.WriteString(`}`)
	}
	nested.WriteString(`}`)
	data := []byte(nested.String())

	var tr TrackRequest
	start := time.Now()
	err := ParseTrackRequestJSON(&tr, data)
	elapsed := time.Since(start)
	require.ErrorIs(t, err, ErrMalformed)
	assert.Less(t, elapsed, time.Microsecond, "depth-1000 reject took %v", elapsed)
}

func TestFault_WireJSONDepthReject(t *testing.T) {
	validCID := "550e8400-e29b-41d4-a716-446655440000"
	var nested strings.Builder
	nested.WriteString(`{"campaign_id":"`)
	nested.WriteString(validCID)
	nested.WriteString(`","payload":`)
	for range 200 {
		nested.WriteString(`{"a":`)
	}
	nested.WriteString(`"leaf"`)
	for range 200 {
		nested.WriteString(`}`)
	}
	nested.WriteString(`}`)

	var tr TrackRequest
	parseErr := ParseTrackRequestJSON(&tr, []byte(nested.String()))
	require.Error(t, parseErr)
	require.True(t, errors.Is(parseErr, ErrMalformed) || errors.Is(parseErr, errMalformedJSON))

	faultproof.Log(t, "wire_json_depth_reject", map[string]string{
		"depth": "200",
		"cap":   fmt.Sprintf("%d", MaxJSONDepth),
		"err":   parseErr.Error(),
	})
}

func TestFault_H2HostileIncompleteDisconnect(t *testing.T) {
	cfg := &config.Config{MaxRequestBodySize: 1 << 20, H2IncompleteMax: 3}
	h := NewAdsPacketHandler(cfg, &mockRegistry{}, nil, nil, nil, NewStaticSlotSharder(1), "fraud", nil)
	defer func() { _ = h.Stop(context.TODO()) }()

	partial := append([]byte(nil), h2ClientPreface[:20]...)
	conn := NewGnetHarnessConn(partial)

	before := testutil.ToFloat64(metrics.H2HostileDisconnectTotal)
	var last gnet.Action
	for range 3 {
		last = h.onTrafficH2(conn, partial)
	}
	assert.Equal(t, gnet.Close, last)
	assert.GreaterOrEqual(t, testutil.ToFloat64(metrics.H2HostileDisconnectTotal), before+1)

	faultproof.Log(t, "h2_hostile_disconnect", map[string]string{
		"incomplete_max": "3",
		"action":         "close",
	})
}
