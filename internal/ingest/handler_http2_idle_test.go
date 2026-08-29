package ingest

import (
	"testing"
	"time"

	"ad-event-processor/internal/config"
	"ad-event-processor/internal/metrics"

	"github.com/panjf2000/gnet/v2"
	"github.com/prometheus/client_golang/prometheus/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestH2Incomplete_IdleClosesDripWithProgress(t *testing.T) {
	cfg := &config.Config{
		MaxRequestBodySize: 1 << 20,
		H2IncompleteMax:    100,
		HTTP1BodyIdleMs:    100,
	}
	h := NewAdsPacketHandler(cfg, &mockRegistry{}, nil, nil, nil, NewJumpHashSharder(1), "fraud", nil)

	preface := append([]byte(nil), h2ClientPreface[:]...)
	partialFrame := []byte{0x00, 0x00, 0x00}
	conn := newFaultGnetConn()
	conn.Append(preface)
	conn.Append(partialFrame)

	buf := conn.inbound
	require.Equal(t, gnet.None, h.onTrafficH2(conn, buf))

	for i := range 3 {
		time.Sleep(25 * time.Millisecond)
		conn.Append([]byte{0x00})
		buf = conn.inbound
		act := h.onTrafficH2(conn, buf)
		require.Equal(t, gnet.None, act, "drip byte %d must not close before idle deadline", i)
	}

	time.Sleep(50 * time.Millisecond)
	buf = conn.inbound
	act := h.onTrafficH2(conn, buf)
	require.Equal(t, gnet.Close, act)
}

func TestH2Incomplete_IdleIncrementsHostileMetric(t *testing.T) {
	cfg := &config.Config{
		MaxRequestBodySize: 1 << 20,
		H2IncompleteMax:    100,
		HTTP1BodyIdleMs:    50,
	}
	h := NewAdsPacketHandler(cfg, &mockRegistry{}, nil, nil, nil, NewJumpHashSharder(1), "fraud", nil)
	before := testutil.ToFloat64(metrics.H2HostileDisconnectTotal)

	preface := append([]byte(nil), h2ClientPreface[:]...)
	conn := newFaultGnetConn()
	conn.Append(preface)
	buf := conn.inbound
	require.Equal(t, gnet.None, h.onTrafficH2(conn, buf))

	time.Sleep(80 * time.Millisecond)
	act := h.onTrafficH2(conn, buf)
	require.Equal(t, gnet.Close, act)
	assert.Greater(t, testutil.ToFloat64(metrics.H2HostileDisconnectTotal), before)
}

func TestH2Hostile_SpinStillClosesZeroProgress(t *testing.T) {
	cfg := &config.Config{MaxRequestBodySize: 1 << 20, H2IncompleteMax: 3, HTTP1BodyIdleMs: 60_000}
	h := NewAdsPacketHandler(cfg, &mockRegistry{}, nil, nil, nil, NewJumpHashSharder(1), "fraud", nil)

	partial := append([]byte(nil), h2ClientPreface[:20]...)
	conn := NewGnetHarnessConn(partial)

	before := testutil.ToFloat64(metrics.H2HostileDisconnectTotal)
	var last gnet.Action
	for range 3 {
		last = h.onTrafficH2(conn, partial)
	}
	require.Equal(t, gnet.Close, last)
	assert.GreaterOrEqual(t, testutil.ToFloat64(metrics.H2HostileDisconnectTotal), before+1)
}
