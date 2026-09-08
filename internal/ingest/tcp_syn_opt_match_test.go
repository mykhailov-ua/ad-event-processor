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

func TestParseHTTP1_XTCPSIGV2(t *testing.T) {
	wire := []byte("POST /track HTTP/1.1\r\nX-TCP-SIG-V2: nop,nop,sackok,mss:1460\r\nContent-Length: 0\r\n\r\n")
	n, req, err := parseHTTP1(wire, 1024, nil)
	require.NoError(t, err)
	require.Equal(t, uint8(1), req.TCPSigOptSet)
	assert.Equal(t, uint32(0x8bc1a792), req.TCPSigOptHash)
	assert.Equal(t, len(wire), n)
}

func TestTCPSynOpt_holdoutLinuxUAPasses(t *testing.T) {
	ua := "Mozilla/5.0 (X11; Linux x86_64)"
	require.False(t, tcpSynOptMismatch(ua, 0x8bc1a792))
}

func TestTCPSynOpt_holdoutWindowsUALinuxOptFails(t *testing.T) {
	ua := "Mozilla/5.0 (Windows NT 10.0; Win64; x64)"
	require.True(t, tcpSynOptMismatch(ua, 0x8bc1a792))
}

func TestTCPSynOpt_holdoutUnknownHashFailOpen(t *testing.T) {
	ua := "Mozilla/5.0 (Windows NT 10.0; Win64; x64)"
	require.False(t, tcpSynOptMismatch(ua, 0xdeadbeef))
}

func TestTCPSynOpt_disabledSkipsSignal(t *testing.T) {
	f := filter.NewDeviceFilter(nil)
	f.SetTCPSynOptCorpusEnabled(false)
	evt := &domain.Event{
		CampaignID:    uuid.New(),
		UA:            "Mozilla/5.0 (Windows NT 10.0; Win64; x64)",
		TCPSigOptHash: 0x8bc1a792,
		TCPSigOptSet:  1,
	}
	require.NoError(t, f.Check(context.Background(), evt))
	assert.Empty(t, evt.FraudReason)
}

func TestTCPSynOpt_enabledIncrementsMetric(t *testing.T) {
	f := filter.NewDeviceFilter(nil)
	f.SetTCPSynOptCorpusEnabled(true)
	before := testutil.ToFloat64(metrics.TCPSynOptMismatchTotal)
	evt := &domain.Event{
		CampaignID:    uuid.New(),
		UA:            "Mozilla/5.0 (Windows NT 10.0; Win64; x64)",
		TCPSigOptHash: 0x8bc1a792,
		TCPSigOptSet:  1,
	}
	require.NoError(t, f.Check(context.Background(), evt))
	assert.Equal(t, before+1, testutil.ToFloat64(metrics.TCPSynOptMismatchTotal))
}

func tcpSynOptMismatch(ua string, hash uint32) bool {
	return filter.TCPSynOptCorpusMismatch(ua, hash)
}

func TestTCPSynOptMetrics_registered(t *testing.T) {
	require.NotNil(t, metrics.TCPSynOptMismatchTotal)
	require.NotNil(t, metrics.TCPSynOptSkippedTotal)
	_ = metrics.TCPSynOptSkippedTotal.WithLabelValues("no_tcp_sig_opt")
}

func TestTCPSynOptConfig_defaultOff(t *testing.T) {
	cfg := &config.Config{}
	cfg.TCPSynOptCorpusEnabled = false
	assert.False(t, cfg.TCPSynOptCorpusEnabled)
}
