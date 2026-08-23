package ingestion

import (
	"context"
	"testing"

	"github.com/bidshard/ad-event-processor/internal/config"
	"github.com/bidshard/ad-event-processor/internal/domain"

	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDeviceFilter_signals(t *testing.T) {
	cfg := &config.Config{}
	sw := NewSettingsWatcher([]redis.UniversalClient{redis.NewClient(&redis.Options{Addr: "127.0.0.1:9"})}, cfg)
	sw.snapshot.Store(&DynamicConfig{
		TLSHashBlocklist: "abc123def,deadbeef",
	})
	f := NewDeviceFilter(sw)

	evt := domain.EventPool.Get().(*domain.Event)
	defer domain.EventPool.Put(evt)
	evt.Reset()
	acc := attachFraudAccumulator(evt)
	defer releaseFraudAccumulator(evt, acc)

	evt.TLSHash = "abc123def"
	evt.SecCHUA = `"Google Chrome";v="120"`
	evt.UA = "curl/8.0"

	require.NoError(t, f.Check(context.Background(), evt))
	assert.True(t, acc.has(FraudReasonTLSBlocklist))
	assert.True(t, acc.has(FraudReasonDeviceMismatch))
}

func TestDeviceFilter_pass_clean_client(t *testing.T) {
	sw := NewSettingsWatcher(nil, &config.Config{})
	f := NewDeviceFilter(sw)

	evt := domain.EventPool.Get().(*domain.Event)
	defer domain.EventPool.Put(evt)
	evt.Reset()
	acc := attachFraudAccumulator(evt)
	defer releaseFraudAccumulator(evt, acc)

	evt.UA = "Mozilla/5.0 Chrome/120"
	evt.SecCHUA = `"Google Chrome";v="120"`

	require.NoError(t, f.Check(context.Background(), evt))
	assert.Equal(t, uint8(0), acc.count)
}

func TestDeviceFilter_tlsImpersonation(t *testing.T) {
	chromeUA := "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36"
	sw := NewSettingsWatcher(nil, &config.Config{})
	f := NewDeviceFilter(sw)

	evt := domain.EventPool.Get().(*domain.Event)
	defer domain.EventPool.Put(evt)
	evt.Reset()
	acc := attachFraudAccumulator(evt)
	defer releaseFraudAccumulator(evt, acc)

	evt.UA = chromeUA
	evt.TLSHash = suspiciousJA3PythonHash

	require.NoError(t, f.Check(context.Background(), evt))
	assert.True(t, acc.has(FraudReasonDeviceMismatch))

	evt.Reset()
	acc = attachFraudAccumulator(evt)
	evt.UA = chromeUA
	evt.TLSJA3 = "python-requests-ja3-fingerprint"

	require.NoError(t, f.Check(context.Background(), evt))
	assert.True(t, acc.has(FraudReasonDeviceMismatch))

	evt.Reset()
	acc = attachFraudAccumulator(evt)
	evt.UA = chromeUA
	evt.TLSJA3 = "chrome-ja3-fingerprint"

	require.NoError(t, f.Check(context.Background(), evt))
	assert.False(t, acc.has(FraudReasonDeviceMismatch))

	evt.Reset()
	acc = attachFraudAccumulator(evt)
	evt.UA = "python-requests/2.31.0"
	evt.TLSHash = suspiciousJA3PythonHash

	require.NoError(t, f.Check(context.Background(), evt))
	assert.False(t, acc.has(FraudReasonDeviceMismatch))
}

func TestTlsFingerprintImpersonating(t *testing.T) {
	chromeUA := "Mozilla/5.0 Chrome/120.0.0.0"
	assert.True(t, tlsFingerprintImpersonating(chromeUA, nil, nil, []byte(suspiciousJA3PythonHash)))
	assert.True(t, tlsFingerprintImpersonating(chromeUA, []byte("python-requests-ja3"), nil, nil))
	assert.False(t, tlsFingerprintImpersonating(chromeUA, []byte("chrome-ja3-fingerprint"), nil, nil))
	assert.False(t, tlsFingerprintImpersonating("curl/8.0", []byte(suspiciousJA3PythonHash), nil, nil))
}

func TestJa3BytesSuspicious(t *testing.T) {
	assert.True(t, ja3BytesSuspicious([]byte(suspiciousJA3PythonHash)))
	assert.True(t, ja3BytesSuspicious([]byte("python-requests-ja3")))
	assert.False(t, ja3BytesSuspicious([]byte("chrome-ja3-fingerprint")))
}

func TestFilterEngine_deviceFilter_before_lua(t *testing.T) {
	sw := NewSettingsWatcher(nil, &config.Config{})
	sw.snapshot.Store(&DynamicConfig{TLSHashBlocklist: "badja3"})
	deviceFilter := NewDeviceFilter(sw)

	registry := &mockRegistry{}
	engine := NewFilterEngine(0, deviceFilter)
	engine.SetRegistry(registry)

	evt := domain.EventPool.Get().(*domain.Event)
	defer domain.EventPool.Put(evt)
	evt.Reset()
	evt.CampaignID = uuid.New()
	evt.TLSHash = "badja3"

	require.NoError(t, engine.Check(context.Background(), evt))
	assert.Greater(t, evt.FraudScore, uint32(0))
}
