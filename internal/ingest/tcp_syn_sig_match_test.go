package ingest

import (
	"context"
	"testing"

	"ad-event-processor/internal/config"
	"ad-event-processor/internal/domain"
	"ad-event-processor/internal/metrics"

	"github.com/google/uuid"
	"github.com/prometheus/client_golang/prometheus/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestHashTCPSynFields_matchesBPF(t *testing.T) {
	assert.Equal(t, uint32(0x00eaf022), hashTCPSynFields(128, 64240, 5, 10))
	assert.Equal(t, uint32(0x007a1022), hashTCPSynFields(64, 29200, 5, 10))
}

func TestParseHTTP1_XTCPSIG(t *testing.T) {
	wire := []byte("POST /track HTTP/1.1\r\nX-TCP-SIG: 00eaf022\r\nContent-Length: 0\r\n\r\n")
	n, req, err := parseHTTP1(wire, 1024, nil)
	require.NoError(t, err)
	require.Equal(t, uint8(1), req.TCPSigSet)
	assert.Equal(t, uint32(0x00eaf022), req.TCPSig)
	assert.Equal(t, len(wire), n)
}

func TestTCPSynSig_holdoutWindowsSigPasses(t *testing.T) {
	ua := "Mozilla/5.0 (Windows NT 10.0; Win64; x64)"
	require.False(t, tcpSynSigMismatch(ua, hashTCPSynFields(128, 64240, 5, 10)))
}

func TestTCPSynSig_holdoutWindowsUALinuxSigFails(t *testing.T) {
	ua := "Mozilla/5.0 (Windows NT 10.0; Win64; x64)"
	require.True(t, tcpSynSigMismatch(ua, hashTCPSynFields(64, 29200, 5, 10)))
}

func TestTCPSynSig_holdoutLinuxUALinuxSigPasses(t *testing.T) {
	ua := "Mozilla/5.0 (X11; Linux x86_64)"
	require.False(t, tcpSynSigMismatch(ua, hashTCPSynFields(64, 29200, 5, 10)))
}

func TestTCPSynSig_holdoutUnknownHashFailOpen(t *testing.T) {
	ua := "Mozilla/5.0 (Windows NT 10.0; Win64; x64)"
	require.False(t, tcpSynSigMismatch(ua, 0xdeadbeef))
}

func TestDeviceFilter_tcpSynSigMismatch(t *testing.T) {
	before := testutil.ToFloat64(metrics.TCPSynSigMismatchTotal)
	sw := NewSettingsWatcher(nil, &config.Config{})
	f := NewDeviceFilter(sw)
	f.SetTCPSynSigEnabled(true)

	evt := domain.EventPool.Get().(*domain.Event)
	defer domain.EventPool.Put(evt)
	evt.Reset()
	acc := attachFraudAccumulator(evt)
	defer releaseFraudAccumulator(evt, acc)

	evt.UA = "Mozilla/5.0 (Windows NT 10.0; Win64; x64)"
	evt.TCPSigSet = 1
	evt.TCPSig = hashTCPSynFields(64, 29200, 5, 10)

	require.NoError(t, f.Check(context.Background(), evt))
	assert.True(t, acc.has(FraudReasonTCPSynOSMismatch))
	assert.Equal(t, before+1, testutil.ToFloat64(metrics.TCPSynSigMismatchTotal))
}

func TestDeviceFilter_tcpSynSigMissingHeaderFailOpen(t *testing.T) {
	before := testutil.ToFloat64(metrics.TCPSynSigSkippedTotal.WithLabelValues("no_tcp_sig"))
	sw := NewSettingsWatcher(nil, &config.Config{})
	f := NewDeviceFilter(sw)
	f.SetTCPSynSigEnabled(true)

	evt := domain.EventPool.Get().(*domain.Event)
	defer domain.EventPool.Put(evt)
	evt.Reset()
	acc := attachFraudAccumulator(evt)
	defer releaseFraudAccumulator(evt, acc)

	evt.UA = "Mozilla/5.0 (Windows NT 10.0; Win64; x64)"
	evt.TCPSigSet = 0

	require.NoError(t, f.Check(context.Background(), evt))
	assert.False(t, acc.has(FraudReasonTCPSynOSMismatch))
	assert.Equal(t, before+1, testutil.ToFloat64(metrics.TCPSynSigSkippedTotal.WithLabelValues("no_tcp_sig")))
}

func TestDeviceFilter_tcpSynSigDisabled(t *testing.T) {
	sw := NewSettingsWatcher(nil, &config.Config{})
	f := NewDeviceFilter(sw)
	f.SetTCPSynSigEnabled(false)

	evt := domain.EventPool.Get().(*domain.Event)
	defer domain.EventPool.Put(evt)
	evt.Reset()
	acc := attachFraudAccumulator(evt)
	defer releaseFraudAccumulator(evt, acc)

	evt.UA = "Mozilla/5.0 (Windows NT 10.0; Win64; x64)"
	evt.TCPSigSet = 1
	evt.TCPSig = hashTCPSynFields(64, 29200, 5, 10)

	require.NoError(t, f.Check(context.Background(), evt))
	assert.False(t, acc.has(FraudReasonTCPSynOSMismatch))
}

func TestFilterEngine_tcpSynSigL2Shadow(t *testing.T) {
	sw := NewSettingsWatcher(nil, &config.Config{})
	deviceFilter := NewDeviceFilter(sw)
	deviceFilter.SetTCPSynSigEnabled(true)
	engine := NewFilterEngine(0, deviceFilter)
	engine.SetRegistry(&mockRegistry{})

	evt := domain.EventPool.Get().(*domain.Event)
	defer domain.EventPool.Put(evt)
	evt.Reset()
	evt.CampaignID = uuid.New()
	evt.UA = "Mozilla/5.0 (Windows NT 10.0; Win64; x64)"
	evt.TCPSigSet = 1
	evt.TCPSig = hashTCPSynFields(64, 29200, 5, 10)

	require.NoError(t, engine.Check(context.Background(), evt))
	assert.Contains(t, evt.FraudReason, FraudReasonCodeTCPSynOSMismatch)
	assert.True(t, evt.ShadowEvent)
}

func TestTCPSynSigMetrics_registered(t *testing.T) {
	require.NotNil(t, metrics.TCPSynSigMismatchTotal)
	require.NotNil(t, metrics.TCPSynSigSkippedTotal)
	_ = metrics.TCPSynSigSkippedTotal.WithLabelValues("no_tcp_sig")
}
