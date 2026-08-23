package ingestion

import (
	"context"
	"testing"

	"github.com/bidshard/ad-event-processor/internal/config"
	"github.com/bidshard/ad-event-processor/internal/domain"
	"github.com/bidshard/ad-event-processor/internal/metrics"

	"github.com/google/uuid"
	"github.com/prometheus/client_golang/prometheus/testutil"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestScanUAFamily(t *testing.T) {
	assert.Equal(t, uaFamilyWindows, scanUAFamily("Mozilla/5.0 (Windows NT 10.0)"))
	assert.Equal(t, uaFamilyMac, scanUAFamily("Mozilla/5.0 (Macintosh; Intel Mac OS X)"))
	assert.Equal(t, uaFamilyLinux, scanUAFamily("Mozilla/5.0 (X11; Linux x86_64)"))
	assert.Equal(t, uaFamilyMobile, scanUAFamily("Mozilla/5.0 (iPhone; CPU iPhone OS 17_0)"))
	assert.Equal(t, uaFamilyMobile, scanUAFamily("Mozilla/5.0 (Linux; Android 14; Pixel)"))
	assert.Equal(t, uaFamilyUnknown, scanUAFamily("curl/8.0"))
}

func TestOSFingerprintMismatch_mobileTTL64NotFlagged(t *testing.T) {
	ua := "Mozilla/5.0 (iPhone; CPU iPhone OS 17_0 like Mac OS X)"
	require.False(t, osFingerprintMismatch(ua, 64, 0, 0))
	require.False(t, osFingerprintMismatch(ua, 63, 1, 65535))
}

func TestOSFingerprintMismatch_windowsTTL64Flagged(t *testing.T) {
	ua := "Mozilla/5.0 (Windows NT 10.0; Win64; x64)"
	require.True(t, osFingerprintMismatch(ua, 64, 0, 0))
}

func TestOSFingerprintMismatch_mobileTTL128Flagged(t *testing.T) {
	ua := "Mozilla/5.0 (Linux; Android 14; Pixel 8)"
	require.True(t, osFingerprintMismatch(ua, 128, 0, 0))
}

func TestParseHTTP1_XTCPTTLAndWindow(t *testing.T) {
	wire := []byte("POST /track HTTP/1.1\r\nX-TCP-TTL: 64\r\nX-TCP-WINDOW: 29200\r\nContent-Length: 0\r\n\r\n")
	n, req, err := parseHTTP1(wire, 1024, nil)
	require.NoError(t, err)
	require.Equal(t, uint8(1), req.TCPTTLSet)
	assert.Equal(t, uint8(64), req.TCPTTL)
	require.Equal(t, uint8(1), req.TCPWindowSet)
	assert.Equal(t, uint16(29200), req.TCPWindow)
	assert.Equal(t, len(wire), n)
}

func TestDeviceFilter_osFingerprintMismatch(t *testing.T) {
	before := testutil.ToFloat64(metrics.OSFingerprintMismatchTotal)
	sw := NewSettingsWatcher(nil, &config.Config{})
	f := NewDeviceFilter(sw)

	evt := domain.EventPool.Get().(*domain.Event)
	defer domain.EventPool.Put(evt)
	evt.Reset()
	acc := attachFraudAccumulator(evt)
	defer releaseFraudAccumulator(evt, acc)

	evt.UA = "Mozilla/5.0 (Windows NT 10.0)"
	evt.TCPTTLSet = 1
	evt.TCPTTL = 64

	require.NoError(t, f.Check(context.Background(), evt))
	assert.True(t, acc.has(FraudReasonOSFingerprint))
	assert.Equal(t, before+1, testutil.ToFloat64(metrics.OSFingerprintMismatchTotal))
}

func TestDeviceFilter_mobileMatchingTTLClean(t *testing.T) {
	sw := NewSettingsWatcher(nil, &config.Config{})
	f := NewDeviceFilter(sw)

	evt := domain.EventPool.Get().(*domain.Event)
	defer domain.EventPool.Put(evt)
	evt.Reset()
	acc := attachFraudAccumulator(evt)
	defer releaseFraudAccumulator(evt, acc)

	evt.UA = "Mozilla/5.0 (iPhone; CPU iPhone OS 17_0)"
	evt.TCPTTLSet = 1
	evt.TCPTTL = 64

	require.NoError(t, f.Check(context.Background(), evt))
	assert.False(t, acc.has(FraudReasonOSFingerprint))
}

func TestDeviceFilter_osFingerprintDisabled(t *testing.T) {
	cfg := &config.Config{}
	sw := NewSettingsWatcher([]redis.UniversalClient{redis.NewClient(&redis.Options{Addr: "127.0.0.1:9"})}, cfg)
	f := NewDeviceFilter(sw)
	f.SetOSFingerprintEnabled(false)

	evt := domain.EventPool.Get().(*domain.Event)
	defer domain.EventPool.Put(evt)
	evt.Reset()
	acc := attachFraudAccumulator(evt)
	defer releaseFraudAccumulator(evt, acc)

	evt.UA = "Mozilla/5.0 (Windows NT 10.0)"
	evt.TCPTTLSet = 1
	evt.TCPTTL = 64

	require.NoError(t, f.Check(context.Background(), evt))
	assert.False(t, acc.has(FraudReasonOSFingerprint))
}

func TestFilterEngine_osFingerprintL2Shadow(t *testing.T) {
	sw := NewSettingsWatcher(nil, &config.Config{})
	deviceFilter := NewDeviceFilter(sw)
	engine := NewFilterEngine(0, deviceFilter)
	engine.SetRegistry(&mockRegistry{})

	evt := domain.EventPool.Get().(*domain.Event)
	defer domain.EventPool.Put(evt)
	evt.Reset()
	evt.CampaignID = uuid.New()
	evt.UA = "Mozilla/5.0 (Windows NT 10.0)"
	evt.TCPTTLSet = 1
	evt.TCPTTL = 64

	require.NoError(t, engine.Check(context.Background(), evt))
	assert.Contains(t, evt.FraudReason, FraudReasonCodeOSFingerprint)
	assert.True(t, evt.ShadowEvent)
}

func TestOSFingerprintMismatchMetric_registered(t *testing.T) {
	require.NotNil(t, metrics.OSFingerprintMismatchTotal)
}
