package ingestion

import (
	"espx/pkg/faultproof"

	"bytes"
	"context"
	"errors"
	"fmt"
	"net"
	"net/http"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"espx/internal/config"

	"github.com/google/uuid"
	"github.com/panjf2000/gnet/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestFault_HTTP1_MalformedCorpus(t *testing.T) {
	var (
		okCount    int
		errCounts  = map[string]int{}
		panicCount atomic.Uint64
	)
	for _, tc := range http1FaultMalformedCases() {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			defer func() {
				if r := recover(); r != nil {
					panicCount.Add(1)
					t.Fatalf("parseHTTP1 panicked: %v", r)
				}
			}()
			n, req, err := parseHTTP1(tc.payload, tc.maxBody, nil)
			if tc.wantOK {
				require.NoError(t, err, "payload=%q", truncateBytes(tc.payload, 80))
				assert.Greater(t, n, 0)
				assert.LessOrEqual(t, n, len(tc.payload))
				if req.HasContentLength {
					assert.Len(t, req.Body, req.ContentLength)
				}
				okCount++
				return
			}
			require.Error(t, err)
			if tc.wantErr != nil {
				assert.ErrorIs(t, err, tc.wantErr, "payload=%q", truncateBytes(tc.payload, 80))
				errCounts[tc.wantErr.Error()]++
			}
		})
	}
	faultproof.Log(t, "http1_malformed_corpus", map[string]string{
		"cases":      fmt.Sprintf("%d", len(http1FaultMalformedCases())),
		"ok":         fmt.Sprintf("%d", okCount),
		"panics":     fmt.Sprintf("%d", panicCount.Load()),
		"incomplete": fmt.Sprintf("%d", errCounts[errIncompleteRequest.Error()]),
		"invalid":    fmt.Sprintf("%d", errCounts[errInvalidRequest.Error()]),
		"too_large":  fmt.Sprintf("%d", errCounts[errPayloadTooLarge.Error()]),
	})
}

func TestFault_HTTP1_OnTrafficMalformed(t *testing.T) {
	cfg := &config.Config{MaxRequestBodySize: 1024}
	h := NewAdsPacketHandler(cfg, &mockRegistry{}, nil, nil, nil, NewJumpHashSharder(1), "fraud", nil)

	cases := []struct {
		name       string
		payload    []byte
		wantStatus int
		wantClose  bool
	}{
		{
			name:       "oversize_cl",
			payload:    append([]byte("POST /track HTTP/1.1\r\nContent-Length: 5000\r\n\r\n"), bytes.Repeat([]byte("z"), 5000)...),
			wantStatus: http.StatusRequestEntityTooLarge,
			wantClose:  true,
		},
		{
			name:       "invalid_request_line",
			payload:    []byte("POST /track\r\n\r\n"),
			wantStatus: http.StatusBadRequest,
			wantClose:  true,
		},
		{
			name:       "binary_garbage",
			payload:    randomWireGarbage(128),
			wantStatus: 0,
			wantClose:  false,
		},
		{
			name:       "missing_cl_on_track",
			payload:    []byte("POST /track HTTP/1.1\r\nContent-Type: application/json\r\n\r\n{}"),
			wantStatus: http.StatusBadRequest,
			wantClose:  true,
		},
		{
			name: "valid_minimal",
			payload: BuildGnetPostTrackJSON([]byte(
				`{"campaign_id":"` + uuid.NewString() + `","type":"click","click_id":"c1"}`,
			)),
			wantStatus: http.StatusAccepted,
			wantClose:  false,
		},
	}

	var accepted, rejected, incomplete int
	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			conn := NewGnetHarnessConn(tc.payload)
			act := h.OnTraffic(conn)
			status := ParseGnetHTTPStatus(conn.Written())
			if tc.wantClose {
				assert.Equal(t, gnet.Close, act)
			}
			if tc.wantStatus != 0 {
				assert.Equal(t, tc.wantStatus, status)
			}
			switch {
			case status == http.StatusAccepted:
				accepted++
			case status == 0:
				incomplete++
			default:
				rejected++
			}
		})
	}
	faultproof.Log(t, "http1_on_traffic_malformed", map[string]string{
		"accepted":   fmt.Sprintf("%d", accepted),
		"rejected":   fmt.Sprintf("%d", rejected),
		"incomplete": fmt.Sprintf("%d", incomplete),
	})
}

func TestFault_HTTP1_ConcurrentParse(t *testing.T) {
	const (
		workers    = 32
		iterations = 500
	)
	cases := http1FaultMalformedCases()
	cases = append(cases, http1FaultCase{
		name:    "valid_corpus",
		payload: nginxTrackCorpus,
		maxBody: 1024 * 1024,
		wantOK:  true,
	})

	var (
		panics atomic.Uint64
		ok     atomic.Uint64
		fail   atomic.Uint64
	)
	var wg sync.WaitGroup
	wg.Add(workers)
	for w := 0; w < workers; w++ {
		go func(seed int) {
			defer wg.Done()
			for i := 0; i < iterations; i++ {
				tc := cases[(seed*iterations+i)%len(cases)]
				func() {
					defer func() {
						if recover() != nil {
							panics.Add(1)
						}
					}()
					_, _, err := parseHTTP1(tc.payload, tc.maxBody, nil)
					if err == nil {
						ok.Add(1)
					} else {
						fail.Add(1)
					}
				}()
			}
		}(w)
	}
	wg.Wait()

	require.Zero(t, panics.Load(), "concurrent parseHTTP1 must not panic")
	total := ok.Load() + fail.Load()
	require.Equal(t, uint64(workers*iterations), total)
	faultproof.Log(t, "http1_concurrent_parse", map[string]string{
		"workers":    fmt.Sprintf("%d", workers),
		"iterations": fmt.Sprintf("%d", iterations),
		"total":      fmt.Sprintf("%d", total),
		"ok":         fmt.Sprintf("%d", ok.Load()),
		"fail":       fmt.Sprintf("%d", fail.Load()),
		"panics":     fmt.Sprintf("%d", panics.Load()),
	})
}

func TestFault_HTTP1_ConcurrentOnTraffic(t *testing.T) {
	const (
		workers   = 24
		perWorker = 50
		maxBody   = 1024 * 1024
	)
	cfg := &config.Config{MaxRequestBodySize: maxBody}
	h := NewAdsPacketHandler(cfg, &mockRegistry{}, nil, nil, nil, NewJumpHashSharder(1), "fraud", nil)

	validBody := []byte(`{"campaign_id":"` + uuid.NewString() + `","type":"click","click_id":"c1"}`)
	validReq := BuildGnetPostTrackJSON(validBody)
	malformed := http1FaultMalformedCases()

	var (
		panics   atomic.Uint64
		accepted atomic.Uint64
		badClose atomic.Uint64
		other    atomic.Uint64
	)
	var wg sync.WaitGroup
	wg.Add(workers)
	for w := 0; w < workers; w++ {
		go func(workerID int) {
			defer wg.Done()
			for i := 0; i < perWorker; i++ {
				var payload []byte
				if i%3 == 0 {
					payload = validReq
				} else {
					tc := malformed[(workerID*perWorker+i)%len(malformed)]
					payload = tc.payload
				}
				func() {
					defer func() {
						if recover() != nil {
							panics.Add(1)
						}
					}()
					conn := NewGnetHarnessConn(payload)
					act := h.OnTraffic(conn)
					status := ParseGnetHTTPStatus(conn.Written())
					switch {
					case status == http.StatusAccepted:
						accepted.Add(1)
					case act == gnet.Close && status == http.StatusBadRequest:
						badClose.Add(1)
					case act == gnet.Close && status == http.StatusRequestEntityTooLarge:
						badClose.Add(1)
					case status == 0:
					default:
						other.Add(1)
					}
				}()
			}
		}(w)
	}
	wg.Wait()

	require.Zero(t, panics.Load())
	faultproof.Log(t, "http1_concurrent_on_traffic", map[string]string{
		"workers":    fmt.Sprintf("%d", workers),
		"per_worker": fmt.Sprintf("%d", perWorker),
		"accepted":   fmt.Sprintf("%d", accepted.Load()),
		"rejected":   fmt.Sprintf("%d", badClose.Load()),
		"other":      fmt.Sprintf("%d", other.Load()),
		"panics":     fmt.Sprintf("%d", panics.Load()),
	})
}

type faultGnetConn struct {
	gnet.Conn
	mu        sync.Mutex
	inbound   []byte
	written   []byte
	responses [][]byte
	ctx       any
	addr      net.Addr
}

func newFaultGnetConn() *faultGnetConn {
	return &faultGnetConn{
		written: make([]byte, 0, 512),
		addr:    gnetHarnessRemoteAddr,
	}
}

func (c *faultGnetConn) Context() any     { return c.ctx }
func (c *faultGnetConn) SetContext(v any) { c.ctx = v }

func (c *faultGnetConn) Write(b []byte) (int, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.written = append(c.written[:0], b...)
	cp := append([]byte(nil), b...)
	c.responses = append(c.responses, cp)
	return len(b), nil
}

func (c *faultGnetConn) AsyncWrite(buf []byte, callback gnet.AsyncCallback) error {
	_, err := c.Write(buf)
	if callback != nil {
		_ = callback(c, nil)
	}
	return err
}

func (c *faultGnetConn) InboundBuffered() int {
	c.mu.Lock()
	defer c.mu.Unlock()
	return len(c.inbound)
}

func (c *faultGnetConn) Peek(n int) ([]byte, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if n > len(c.inbound) {
		n = len(c.inbound)
	}
	return c.inbound[:n], nil
}

func (c *faultGnetConn) Discard(n int) (int, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if n > len(c.inbound) {
		n = len(c.inbound)
	}
	c.inbound = c.inbound[n:]
	return n, nil
}

func (c *faultGnetConn) RemoteAddr() net.Addr { return c.addr }

func (c *faultGnetConn) Append(b []byte) {
	c.mu.Lock()
	c.inbound = append(c.inbound, b...)
	c.mu.Unlock()
}

func (c *faultGnetConn) SetReadDeadline(time.Time) error  { return nil }
func (c *faultGnetConn) SetWriteDeadline(time.Time) error { return nil }
func (c *faultGnetConn) SetDeadline(time.Time) error      { return nil }
func (c *faultGnetConn) Close() error                     { return nil }

func TestFault_HTTP1_IncrementalConcurrentWrite(t *testing.T) {
	cfg := &config.Config{MaxRequestBodySize: 1024 * 1024}
	h := NewAdsPacketHandler(cfg, &mockRegistry{}, nil, nil, nil, NewJumpHashSharder(1), "fraud", nil)

	payloads := [][]byte{
		nginxTrackCorpus,
		BuildGnetPostTrackJSON([]byte(`{"campaign_id":"` + uuid.NewString() + `","type":"click","click_id":"x"}`)),
		[]byte("POST /track HTTP/1.1\r\nContent-Length: 999\r\n\r\n"),
		randomWireGarbage(200),
	}

	const rounds = 20
	var panics atomic.Uint64
	var completed atomic.Uint64

	for r := 0; r < rounds; r++ {
		raw := payloads[r%len(payloads)]
		conn := newFaultGnetConn()
		var wg sync.WaitGroup

		wg.Add(1)
		go func() {
			defer wg.Done()
			chunk := 1 + (r % 7)
			for i := 0; i < len(raw); i += chunk {
				end := i + chunk
				if end > len(raw) {
					end = len(raw)
				}
				conn.Append(raw[i:end])
				time.Sleep(time.Microsecond)
			}
		}()

		wg.Add(1)
		go func() {
			defer wg.Done()
			deadline := time.Now().Add(200 * time.Millisecond)
			for time.Now().Before(deadline) {
				func() {
					defer func() {
						if recover() != nil {
							panics.Add(1)
						}
					}()
					_ = h.OnTraffic(conn)
				}()
				time.Sleep(10 * time.Microsecond)
			}
		}()

		wg.Wait()
		if ParseGnetHTTPStatus(conn.written) == http.StatusAccepted {
			completed.Add(1)
		}
	}

	require.Zero(t, panics.Load())
	faultproof.Log(t, "http1_incremental_concurrent_write", map[string]string{
		"rounds":    fmt.Sprintf("%d", rounds),
		"completed": fmt.Sprintf("%d", completed.Load()),
		"panics":    fmt.Sprintf("%d", panics.Load()),
	})
}

func TestFault_HTTP1_PipelinedMalformedMix(t *testing.T) {
	cfg := &config.Config{MaxRequestBodySize: 1024 * 1024}
	h := NewAdsPacketHandler(cfg, &mockRegistry{}, nil, nil, nil, NewJumpHashSharder(1), "fraud", nil)

	valid := BuildGnetPostTrackJSON([]byte(`{"campaign_id":"` + uuid.NewString() + `","type":"click","click_id":"p1"}`))
	var buf []byte
	for i := 0; i < 5; i++ {
		buf = append(buf, valid...)
	}
	buf = append(buf, []byte("POST /track HTTP/1.1\r\nContent-Length: 99999\r\n\r\n")...)
	buf = append(buf, randomWireGarbage(32)...)

	conn := NewGnetHarnessConn(buf)
	act := h.OnTraffic(conn)
	assert.Equal(t, gnet.None, act)
	assert.Equal(t, 5, conn.WriteCount())
	for _, resp := range conn.AllResponses() {
		assert.Equal(t, http.StatusAccepted, ParseGnetHTTPStatus(resp))
	}

	remaining := buf
	for i := 0; i < 5; i++ {
		n, _, err := parseHTTP1(remaining, cfg.MaxRequestBodySize, nil)
		require.NoError(t, err)
		remaining = remaining[n:]
	}
	_, _, err := parseHTTP1(remaining, cfg.MaxRequestBodySize, nil)
	require.Error(t, err)
	assert.True(t, errors.Is(err, errIncompleteRequest) || errors.Is(err, errPayloadTooLarge))

	faultproof.Log(t, "http1_pipelined_malformed_mix", map[string]string{
		"accepted": "5",
		"tail_err": err.Error(),
	})
}

func TestFault_HTTP1_PipelinedKeepAliveBudget(t *testing.T) {
	if testing.Short() {
		t.Skip("fault integration test")
	}

	ctx := context.Background()
	infra, cleanup := setupAdsFaultInfra(t)
	defer cleanup()

	stack := startAdsIngestStack(t, infra, "ads-fault-http1-pipeline")
	defer stack.Close(t)

	const n = 10
	var pipelined []byte
	for i := 0; i < n; i++ {
		body := fmt.Sprintf(
			`{"campaign_id":"%s","type":"impression","click_id":"pipe-%d","user_id":"pipe-user"}`,
			stack.CampaignID, i,
		)
		pipelined = append(pipelined, BuildGnetPostTrackJSON([]byte(body))...)
	}

	conn := NewGnetHarnessConn(pipelined)
	act := stack.Handler.OnTraffic(conn)
	assert.Equal(t, gnet.None, act)
	require.Equal(t, n, conn.WriteCount())
	for i, resp := range conn.AllResponses() {
		require.Equal(t, http.StatusAccepted, ParseGnetHTTPStatus(resp), "response %d", i+1)
	}

	AssertBudgetInvariant(t, ctx, infra.Pool, infra.Redis, stack.CampaignID)

	faultproof.Log(t, "http1_pipelined_keepalive_budget", map[string]string{
		"pipelined": fmt.Sprintf("%d", n),
		"accepted":  fmt.Sprintf("%d", n),
		"budget_ok": "true",
	})
}

func truncateBytes(b []byte, n int) string {
	if len(b) <= n {
		return string(b)
	}
	return string(b[:n]) + "..."
}
