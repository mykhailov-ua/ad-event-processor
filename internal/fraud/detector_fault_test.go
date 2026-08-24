package fraud

import (
	"context"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"ad-event-processor/pkg/faultproof"

	"ad-event-processor/internal/database"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type stubFinder struct {
	ips []SuspiciousIP
}

func (finder stubFinder) FindSuspiciousIPs(context.Context) ([]SuspiciousIP, error) {
	return finder.ips, nil
}

type countingManagement struct {
	mu         sync.Mutex
	calls      map[string]int
	batchCalls int
	fail       atomic.Uint32
}

func (mgmt *countingManagement) BlockIP(_ context.Context, ip string) error {
	mgmt.mu.Lock()
	defer mgmt.mu.Unlock()
	if mgmt.calls == nil {
		mgmt.calls = make(map[string]int)
	}
	if mgmt.fail.Load() > 0 {
		mgmt.fail.Add(^uint32(0))
		return ErrManagementUnavailable
	}
	mgmt.calls[ip]++
	return nil
}

func (mgmt *countingManagement) EnqueueFraudThreat(_ context.Context, action, ip, campaignID string, score float64, boost int32, ttlSeconds int64) error {
	mgmt.mu.Lock()
	defer mgmt.mu.Unlock()
	if mgmt.calls == nil {
		mgmt.calls = make(map[string]int)
	}
	if mgmt.fail.Load() > 0 {
		mgmt.fail.Add(^uint32(0))
		return ErrManagementUnavailable
	}
	mgmt.calls[ip]++
	return nil
}

func (mgmt *countingManagement) EnqueueFraudThreatBatch(_ context.Context, items []FraudThreatEnqueueItem) (int, error) {
	mgmt.mu.Lock()
	defer mgmt.mu.Unlock()
	if mgmt.calls == nil {
		mgmt.calls = make(map[string]int)
	}
	if mgmt.fail.Load() > 0 {
		mgmt.fail.Add(^uint32(0))
		return 0, ErrManagementUnavailable
	}
	mgmt.batchCalls++
	for _, item := range items {
		mgmt.calls[item.IP]++
	}
	return len(items), nil
}

func (mgmt *countingManagement) count(ip string) int {
	mgmt.mu.Lock()
	defer mgmt.mu.Unlock()
	return mgmt.calls[ip]
}

func TestIdempotencyStore_TryClaim(t *testing.T) {
	if testing.Short() {
		t.Skip("integration test")
	}

	pool, cleanup := database.SetupTestDB(t)
	defer cleanup()

	ctx := context.Background()
	store := NewIdempotencyStore(pool)

	claimed, err := store.TryClaim(ctx, "203.0.113.1")
	require.NoError(t, err)
	assert.True(t, claimed)

	claimed, err = store.TryClaim(ctx, "203.0.113.1")
	require.NoError(t, err)
	assert.False(t, claimed)

	require.NoError(t, store.Release(ctx, "203.0.113.1"))

	claimed, err = store.TryClaim(ctx, "203.0.113.1")
	require.NoError(t, err)
	assert.True(t, claimed)
}

func TestFault_ivtDetectorExactlyOnce(t *testing.T) {
	if testing.Short() {
		t.Skip("integration test")
	}

	pool, cleanup := database.SetupTestDB(t)
	defer cleanup()

	ctx := context.Background()
	mgmt := &countingManagement{}
	detector := NewDetector(
		stubFinder{ips: []SuspiciousIP{{IP: "198.51.100.10", Reason: "ivt_detected", Score: 12}}},
		NewIdempotencyStore(pool),
		mgmt,
		pool,
		DetectorConfig{OutboxPendingLimit: 0},
	)

	const goroutines = 20
	var wg sync.WaitGroup
	var success atomic.Int32
	start := make(chan struct{})

	wg.Add(goroutines)
	for range goroutines {
		go func() {
			defer wg.Done()
			<-start
			result, err := detector.Run(ctx)
			if err == nil && result.Enqueued > 0 {
				success.Add(1)
			}
		}()
	}
	close(start)
	wg.Wait()

	assert.Equal(t, 1, mgmt.count("198.51.100.10"))
	assert.Equal(t, int32(1), success.Load())

	hasClaim, err := detector.idem.HasClaim(ctx, "198.51.100.10")
	require.NoError(t, err)
	assert.True(t, hasClaim)

	faultproof.Log(t, "ivt_detector_exactly_once", map[string]string{
		"subsystem":    "ivt_detector",
		"goroutines":   "20",
		"block_calls":  "1",
		"exactly_once": "true",
	})
}

func TestFault_ivtDetectorOutboxBackpressure(t *testing.T) {
	if testing.Short() {
		t.Skip("integration test")
	}

	pool, cleanup := database.SetupTestDB(t)
	defer cleanup()

	ctx := context.Background()
	for range 5 {
		_, err := pool.Exec(ctx, `
			INSERT INTO outbox_events (event_type, payload, status)
			VALUES ('UPDATE_CAMPAIGN_PACING', '{"campaign_id":"00000000-0000-0000-0000-000000000001","pacing_mode":"even"}', 'PENDING')`)
		require.NoError(t, err)
	}

	mgmt := &countingManagement{}
	detector := NewDetector(
		stubFinder{ips: []SuspiciousIP{{IP: "198.51.100.20", Reason: "ivt_detected", Score: 9}}},
		NewIdempotencyStore(pool),
		mgmt,
		pool,
		DetectorConfig{OutboxPendingLimit: 3},
	)

	result, err := detector.Run(ctx)
	require.ErrorIs(t, err, ErrOutboxBackpressure)
	assert.True(t, result.Backlogged)
	assert.Equal(t, 0, mgmt.count("198.51.100.20"))

	faultproof.Log(t, "ivt_detector_outbox_backpressure", map[string]string{
		"subsystem":           "ivt_detector",
		"backpressure_active": "true",
		"pending_limit":       "3",
	})
}

func TestFault_ivtDetectorManagementRetry(t *testing.T) {
	if testing.Short() {
		t.Skip("integration test")
	}

	pool, cleanup := database.SetupTestDB(t)
	defer cleanup()

	ctx := context.Background()
	mgmt := &countingManagement{}
	mgmt.fail.Store(1)

	detector := NewDetector(
		stubFinder{ips: []SuspiciousIP{{IP: "198.51.100.30", Reason: "ivt_detected", Score: 7}}},
		NewIdempotencyStore(pool),
		mgmt,
		pool,
		DetectorConfig{OutboxPendingLimit: 0},
	)

	_, err := detector.Run(ctx)
	require.Error(t, err)
	assert.Equal(t, 0, mgmt.count("198.51.100.30"))

	hasClaim, err := detector.idem.HasClaim(ctx, "198.51.100.30")
	require.NoError(t, err)
	assert.False(t, hasClaim)

	result, err := detector.Run(ctx)
	require.NoError(t, err)
	assert.Equal(t, 1, result.Enqueued)
	assert.Equal(t, 1, mgmt.count("198.51.100.30"))

	faultproof.Log(t, "ivt_detector_management_retry", map[string]string{
		"subsystem":    "ivt_detector",
		"retry_ok":     "true",
		"block_calls":  "1",
		"exactly_once": "true",
	})
}

func TestAnalyzer_FindSuspiciousIPs_clickRatio(t *testing.T) {
	if testing.Short() {
		t.Skip("clickhouse integration test")
	}

	conn, cleanup := setupClickHouseTest(t)
	defer cleanup()

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	seedClickHouseEvents(t, conn, "ratio-ip", "bot-ua-shared", 2, 20)

	analyzer := NewAnalyzer(database.NewCHQuery(conn, database.CHQueryConfig{}), AnalyzerConfig{
		Window:          time.Hour,
		MinClicks:       5,
		MinImpressions:  1,
		ClickToImpRatio: 3.0,
		MinIPsPerUA:     50,
	})

	ips, err := analyzer.FindSuspiciousIPs(ctx)
	require.NoError(t, err)

	found := false
	for _, candidate := range ips {
		if candidate.IP == ipHashHex("ratio-ip") {
			found = true
			assert.Equal(t, "ivt_high_click_to_imp_ratio", candidate.Reason)
		}
	}
	assert.True(t, found, "expected ratio-ip in suspicious set: %+v", ips)
}
