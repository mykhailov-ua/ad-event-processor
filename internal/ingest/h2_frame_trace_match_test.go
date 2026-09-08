package ingest

import (
	"context"
	"testing"

	"ad-event-processor/internal/config"
	"ad-event-processor/internal/domain"
	"ad-event-processor/internal/filter"
	"ad-event-processor/internal/metrics"

	"github.com/google/uuid"
	"github.com/prometheus/client_golang/prometheus/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseHTTP1_XH2FrameTrace(t *testing.T) {
	wire := []byte("POST /track HTTP/1.1\r\nX-H2-FRAME-TRACE: settings:0,headers:1,window_update:2,data:0\r\nContent-Length: 0\r\n\r\n")
	n, req, err := parseHTTP1(wire, 1024, nil)
	require.NoError(t, err)
	require.Equal(t, uint8(1), req.H2FrameTraceSet)
	assert.Equal(t, uint32(0x121379f3), req.H2FrameTraceHash)
	assert.Equal(t, len(wire), n)
}

func TestH2FrameTrace_holdoutSafariUAPasses(t *testing.T) {
	ua := "Mozilla/5.0 (iPhone; CPU iPhone OS 17_0 like Mac OS X) AppleWebKit/605.1.15"
	require.False(t, h2FrameTraceMismatch(ua, 0x121379f3))
}

func TestH2FrameTrace_holdoutAutomatorTraceSafariUAFails(t *testing.T) {
	ua := "Mozilla/5.0 (iPhone; CPU iPhone OS 17_0 like Mac OS X) AppleWebKit/605.1.15"
	require.True(t, h2FrameTraceMismatch(ua, 0xb0e6d89e))
}

func TestH2FrameTrace_holdoutUnknownHashFailOpen(t *testing.T) {
	ua := "Mozilla/5.0 (iPhone; CPU iPhone OS 17_0 like Mac OS X) AppleWebKit/605.1.15"
	require.False(t, h2FrameTraceMismatch(ua, 0xdeadbeef))
}

func TestH2FrameTrace_disabledSkipsSignal(t *testing.T) {
	f := filter.NewDeviceFilter(nil)
	f.SetH2FrameTraceCorpusEnabled(false)
	evt := &domain.Event{
		CampaignID:       uuid.New(),
		UA:               "Mozilla/5.0 (iPhone; CPU iPhone OS 17_0 like Mac OS X) AppleWebKit/605.1.15",
		H2FrameTraceHash: 0xb0e6d89e,
		H2FrameTraceSet:  1,
	}
	require.NoError(t, f.Check(context.Background(), evt))
	assert.Empty(t, evt.FraudReason)
}

func TestH2FrameTrace_enabledIncrementsMetric(t *testing.T) {
	f := filter.NewDeviceFilter(nil)
	f.SetH2FrameTraceCorpusEnabled(true)
	before := testutil.ToFloat64(metrics.H2FrameTraceMismatchTotal)
	evt := &domain.Event{
		CampaignID:       uuid.New(),
		UA:               "Mozilla/5.0 (iPhone; CPU iPhone OS 17_0 like Mac OS X) AppleWebKit/605.1.15",
		H2FrameTraceHash: 0xb0e6d89e,
		H2FrameTraceSet:  1,
	}
	require.NoError(t, f.Check(context.Background(), evt))
	assert.Equal(t, before+1, testutil.ToFloat64(metrics.H2FrameTraceMismatchTotal))
}

func h2FrameTraceMismatch(ua string, hash uint32) bool {
	return filter.H2FrameTraceCorpusMismatch(ua, hash)
}

func TestH2FrameTraceMetrics_registered(t *testing.T) {
	require.NotNil(t, metrics.H2FrameTraceMismatchTotal)
	require.NotNil(t, metrics.H2FrameTraceSkippedTotal)
	_ = metrics.H2FrameTraceSkippedTotal.WithLabelValues("no_h2_frame_trace")
}

func TestH2FrameTraceConfig_defaultOff(t *testing.T) {
	cfg := &config.Config{}
	cfg.H2FrameTraceCorpusEnabled = false
	assert.False(t, cfg.H2FrameTraceCorpusEnabled)
}
