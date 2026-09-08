package ingest

import (
	"bytes"
	"fmt"
	"testing"
	"time"

	"ad-event-processor/internal/config"

	"github.com/google/uuid"
	"github.com/panjf2000/gnet/v2"
	"github.com/stretchr/testify/require"
)

func buildH2PipelinedTrackBodies(n int, campaignID string) []byte {
	var buf bytes.Buffer
	for i := range n {
		body := []byte(fmt.Sprintf(
			`{"campaign_id":"%s","type":"impression","click_id":"h2-offload-%d","user_id":"h2-user"}`,
			campaignID, i,
		))
		streamID := uint32(1 + i*2)
		buf.Write(buildH2TrackRequestOnStream(streamID, body, i == 0))
	}
	return buf.Bytes()
}

func TestAdsPacketHandler_h2WorkerPoolOffloadsTrack(t *testing.T) {
	campaignID := uuid.NewString()
	cfg := &config.Config{MaxRequestBodySize: 1 << 20}
	h := NewAdsPacketHandler(cfg, &mockRegistry{}, nil, nil, nil, NewJumpHashSharder(1), "fraud", nil)
	pool := NewPinnedWorkerPool(1, 64)
	defer pool.Shutdown()
	h.SetWorkerPool(pool)

	body := []byte(fmt.Sprintf(
		`{"campaign_id":"%s","type":"impression","click_id":"h2-single","user_id":"h2-user"}`,
		campaignID,
	))
	conn := NewGnetHarnessConn(buildH2TrackRequest(body))
	conn.SetOnWake(func(c *GnetHarnessConn) {
		_ = h.OnTraffic(c)
	})

	require.Equal(t, gnet.None, h.OnTraffic(conn))
	pool.WaitIdle()

	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		if conn.WriteCount() >= 1 && conn.InboundBuffered() == 0 {
			break
		}
		time.Sleep(time.Millisecond)
	}
	require.Zero(t, conn.InboundBuffered())
	require.GreaterOrEqual(t, conn.WriteCount(), 1, "H2 settings and/or response frames written")
}

func TestAdsPacketHandler_h2WorkerPoolSecondRequest_holdout(t *testing.T) {
	campaignID := uuid.NewString()
	cfg := &config.Config{MaxRequestBodySize: 1 << 20}
	h := NewAdsPacketHandler(cfg, &mockRegistry{}, nil, nil, nil, NewJumpHashSharder(1), "fraud", nil)
	pool := NewPinnedWorkerPool(1, 64)
	defer pool.Shutdown()
	h.SetWorkerPool(pool)

	conn := NewGnetHarnessConn(nil)
	body1 := []byte(fmt.Sprintf(`{"campaign_id":"%s","type":"impression","click_id":"dbg-0"}`, campaignID))
	conn.Append(buildH2TrackRequestOnStream(1, body1, true))
	require.Equal(t, gnet.None, h.OnTraffic(conn))
	pool.WaitIdle()

	connCtx := http1ConnContext(conn)
	require.True(t, connCtx.ProtoH2)
	require.True(t, connCtx.H2.Established)

	body2 := []byte(fmt.Sprintf(`{"campaign_id":"%s","type":"impression","click_id":"dbg-1"}`, campaignID))
	conn.Append(buildH2TrackRequestOnStream(3, body2, false))
	act := h.OnTraffic(conn)
	require.Equal(t, gnet.None, act, "second H2 request on established conn")
}

func TestAdsPacketHandler_h2WorkerPoolPipelinedTrack(t *testing.T) {
	campaignID := uuid.NewString()
	cfg := &config.Config{MaxRequestBodySize: 1 << 20}
	h := NewAdsPacketHandler(cfg, &mockRegistry{}, nil, nil, nil, NewJumpHashSharder(1), "fraud", nil)
	pool := NewPinnedWorkerPool(1, 64)
	defer pool.Shutdown()
	h.SetWorkerPool(pool)

	conn := NewGnetHarnessConn(nil)
	conn.SetOnWake(func(c *GnetHarnessConn) {
		_ = h.OnTraffic(c)
	})

	for i := range 2 {
		body := []byte(fmt.Sprintf(
			`{"campaign_id":"%s","type":"impression","click_id":"h2-pipe-%d","user_id":"h2-user"}`,
			campaignID, i,
		))
		streamID := uint32(1 + i*2)
		conn.Append(buildH2TrackRequestOnStream(streamID, body, i == 0))
		require.Equal(t, gnet.None, h.OnTraffic(conn))
		pool.WaitIdle()
	}

	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		if conn.WriteCount() >= 2 {
			break
		}
		time.Sleep(time.Millisecond)
	}
	require.GreaterOrEqual(t, conn.WriteCount(), 2, "one H2 response per request")
	require.Zero(t, conn.InboundBuffered())
}

func TestAdsPacketHandler_h2WorkerPoolBusyBlocksSecondRequest(t *testing.T) {
	campaignID := uuid.NewString()
	cfg := &config.Config{MaxRequestBodySize: 1 << 20}
	h := NewAdsPacketHandler(cfg, &mockRegistry{}, nil, nil, nil, NewJumpHashSharder(1), "fraud", nil)
	pool := NewPinnedWorkerPool(1, 64)
	defer pool.Shutdown()
	h.SetWorkerPool(pool)

	base := NewGnetHarnessConn(nil)
	conn := &deferAsyncHarnessConn{GnetHarnessConn: base}
	base.SetOnWake(func(c *GnetHarnessConn) {
		_ = h.OnTraffic(conn)
	})
	conn.Append(buildH2PipelinedTrackBodies(2, campaignID))

	require.Equal(t, gnet.None, h.OnTraffic(conn))
	pool.WaitIdle()
	require.Greater(t, conn.InboundBuffered(), 0, "second H2 request must wait while first offload is in flight")
	require.True(t, http1ConnContext(conn).HTTP1OffloadBusy.Load())

	conn.flushPending()
	pool.WaitIdle()
	_ = h.OnTraffic(conn)

	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		conn.flushPending()
		pool.WaitIdle()
		_ = h.OnTraffic(conn)
		if conn.InboundBuffered() == 0 && conn.WriteCount() >= 2 {
			break
		}
		time.Sleep(time.Millisecond)
	}
	require.Zero(t, conn.InboundBuffered())
	require.GreaterOrEqual(t, conn.WriteCount(), 2)
}
