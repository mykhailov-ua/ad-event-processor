package ingest

import (
	"fmt"
	"net/http"
	"testing"
	"time"

	"ad-event-processor/internal/config"

	"github.com/google/uuid"
	"github.com/panjf2000/gnet/v2"
	"github.com/stretchr/testify/require"
)

func TestAdsPacketHandler_workerPoolHonorsReactClose(t *testing.T) {
	cfg := &config.Config{MaxRequestBodySize: 1 << 20}
	h := NewAdsPacketHandler(cfg, &mockRegistry{}, nil, nil, nil, NewJumpHashSharder(1), "fraud", nil)
	pool := NewPinnedWorkerPool(1, 64)
	defer pool.Shutdown()
	h.SetWorkerPool(pool)

	conn := NewGnetHarnessConn(nil)
	conn.onWake = func(c *GnetHarnessConn) {
		_ = h.OnTraffic(c)
	}

	conn.Append(BuildGnetHTTP("POST", "/tg/bid", map[string]string{
		"Content-Type": "application/json",
	}, nil))
	require.Equal(t, gnet.None, h.OnTraffic(conn))
	pool.WaitIdle()

	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		if conn.Closed() {
			break
		}
		time.Sleep(time.Millisecond)
	}
	require.True(t, conn.Closed(), "worker pool must close conn after React returns gnet.Close")
}

func TestAdsPacketHandler_workerPoolPipelinedTrack(t *testing.T) {
	campaignID := uuid.NewString()
	cfg := &config.Config{MaxRequestBodySize: 1 << 20}
	h := NewAdsPacketHandler(cfg, &mockRegistry{}, nil, nil, nil, NewJumpHashSharder(1), "fraud", nil)
	pool := NewPinnedWorkerPool(2, 64)
	defer pool.Shutdown()
	h.SetWorkerPool(pool)

	body := func(i int) []byte {
		return []byte(fmt.Sprintf(
			`{"campaign_id":"%s","type":"impression","click_id":"pipe-%d","user_id":"pipe-user"}`,
			campaignID, i,
		))
	}

	conn := NewGnetHarnessConn(nil)
	conn.onWake = func(c *GnetHarnessConn) {
		_ = h.OnTraffic(c)
	}

	for i := range 2 {
		conn.Append(BuildGnetPostTrackJSON(body(i)))
		require.Equal(t, gnet.None, h.OnTraffic(conn))
		pool.WaitIdle()
	}

	require.Equal(t, 2, conn.WriteCount(), "responses written")
	for i, resp := range conn.AllResponses() {
		require.Equal(t, http.StatusAccepted, ParseGnetHTTPStatus(resp), "response %d", i+1)
	}
}

func TestAdsPacketHandler_workerPoolCoalescedKeepAlive(t *testing.T) {
	campaignID := uuid.NewString()
	cfg := &config.Config{MaxRequestBodySize: 1 << 20}
	h := NewAdsPacketHandler(cfg, &mockRegistry{}, nil, nil, nil, NewJumpHashSharder(1), "fraud", nil)
	pool := NewPinnedWorkerPool(1, 64)
	defer pool.Shutdown()
	h.SetWorkerPool(pool)

	conn := NewGnetHarnessConn(nil)
	conn.onWake = func(c *GnetHarnessConn) {
		_ = h.OnTraffic(c)
	}

	for i := range 2 {
		body := []byte(fmt.Sprintf(
			`{"campaign_id":"%s","type":"impression","click_id":"coal-%d","user_id":"coal-user"}`,
			campaignID, i,
		))
		conn.Append(BuildGnetPostTrackJSON(body))
	}
	require.Equal(t, gnet.None, h.OnTraffic(conn))
	pool.WaitIdle()

	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		if conn.WriteCount() >= 2 {
			break
		}
		time.Sleep(time.Millisecond)
	}
	require.Equal(t, 2, conn.WriteCount(), "one response per coalesced request")
	for i, resp := range conn.AllResponses() {
		require.Equal(t, http.StatusAccepted, ParseGnetHTTPStatus(resp), "response %d", i+1)
	}
}

type deferAsyncHarnessConn struct {
	*GnetHarnessConn
	pending []func()
}

func (c *deferAsyncHarnessConn) AsyncWrite(buf []byte, callback gnet.AsyncCallback) error {
	cp := append([]byte(nil), buf...)
	c.mu.Lock()
	c.pending = append(c.pending, func() {
		_, _ = c.writeLocked(cp)
		if callback != nil {
			_ = callback(c, nil)
		}
	})
	c.mu.Unlock()
	return nil
}

func (c *deferAsyncHarnessConn) flushPending() {
	c.mu.Lock()
	pending := c.pending
	c.pending = nil
	c.mu.Unlock()
	for _, fn := range pending {
		fn()
	}
}

func TestAdsPacketHandler_workerPoolCoalescedKeepAliveDeferredAsyncWrite(t *testing.T) {
	campaignID := uuid.NewString()
	cfg := &config.Config{MaxRequestBodySize: 1 << 20}
	h := NewAdsPacketHandler(cfg, &mockRegistry{}, nil, nil, nil, NewJumpHashSharder(1), "fraud", nil)
	pool := NewPinnedWorkerPool(1, 64)
	defer pool.Shutdown()
	h.SetWorkerPool(pool)

	base := NewGnetHarnessConn(nil)
	conn := &deferAsyncHarnessConn{GnetHarnessConn: base}
	conn.onWake = func(c *GnetHarnessConn) {
		_ = h.OnTraffic(conn)
	}

	for i := range 2 {
		body := []byte(fmt.Sprintf(
			`{"campaign_id":"%s","type":"impression","click_id":"defer-coal-%d","user_id":"coal-user"}`,
			campaignID, i,
		))
		conn.Append(BuildGnetPostTrackJSON(body))
	}
	require.Equal(t, gnet.None, h.OnTraffic(conn))
	pool.WaitIdle()
	require.Equal(t, 0, conn.WriteCount(), "response must wait for deferred AsyncWrite flush")
	require.True(t, http1ConnContext(conn).http1OffloadBusy.Load(), "conn stays busy until async write completes")

	require.Equal(t, gnet.None, h.OnTraffic(conn))
	require.Equal(t, 0, conn.WriteCount(), "second request must not run while first write is pending")

	conn.flushPending()

	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		conn.flushPending()
		if conn.WriteCount() >= 2 {
			break
		}
		pool.WaitIdle()
		_ = h.OnTraffic(conn)
		time.Sleep(time.Millisecond)
	}
	require.Equal(t, 2, conn.WriteCount(), "one response per coalesced request after ordered async writes")
	for i, resp := range conn.AllResponses() {
		require.Equal(t, http.StatusAccepted, ParseGnetHTTPStatus(resp), "response %d", i+1)
	}
}

func TestHTTP1ConnContext_followsOffloadParent(t *testing.T) {
	connCtx := &connContext{workerID: 3}
	offloadCtx := &connContext{http1ConnCtx: connCtx}
	conn := NewGnetHarnessConn(nil)
	conn.SetContext(offloadCtx)
	connCtx.http1OffloadBusy.Store(true)

	got := http1ConnContext(conn)
	require.Same(t, connCtx, got)
	require.True(t, got.http1OffloadBusy.Load())
}

func TestWriteFilterReject_duplicateOnlyClosesConn(t *testing.T) {
	cfg := &config.Config{MaxRequestBodySize: 1 << 20}
	h := NewAdsPacketHandler(cfg, &mockRegistry{}, nil, nil, nil, NewJumpHashSharder(1), "fraud", nil)
	pool := NewPinnedWorkerPool(1, 64)
	defer pool.Shutdown()
	h.SetWorkerPool(pool)

	dupConn := NewGnetHarnessConn(nil)
	dupConnCtx := &connContext{}
	dupOffload := &connContext{http1ConnCtx: dupConnCtx, offloadConn: dupConn}
	h.writeFilterReject(dupConn, respDuplicate, dupOffload)
	pool.WaitIdle()
	require.True(t, dupConn.Closed(), "409 duplicate must close conn on offload path")

	budgetConn := NewGnetHarnessConn(nil)
	budgetConnCtx := &connContext{}
	budgetOffload := &connContext{http1ConnCtx: budgetConnCtx, offloadConn: budgetConn}
	h.writeFilterReject(budgetConn, respBudget, budgetOffload)
	pool.WaitIdle()
	require.False(t, budgetConn.Closed(), "non-duplicate reject keeps keep-alive")
	require.Equal(t, http.StatusPaymentRequired, ParseGnetHTTPStatus(budgetConn.Written()))
}
