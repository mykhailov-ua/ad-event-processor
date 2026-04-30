package integration

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/mykhailov-ua/ad-event-processor/internal/ads"
	"github.com/mykhailov-ua/ad-event-processor/internal/ads/repository"
	"github.com/mykhailov-ua/ad-event-processor/internal/config"
	"github.com/mykhailov-ua/ad-event-processor/internal/database"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGracefulShutdown_NoDataLoss(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	pool, cleanup := setupTestDB(t)
	defer cleanup()

	ctx, cancel := context.WithCancel(context.Background())
	queries := repository.New(pool)

	// Config with large batches and slow flushes to ensure data stays in memory
	cfg := &config.Config{
		EventBatchSize: 500,
		EventFlushMs:   5000, // Very slow
		StatsFlushMs:   5000, // Very slow
		MaxWorkers:     4,
		WriteTimeoutMs: 5000,
	}

	// 1. Setup partitions and campaign
	pm := database.NewPartitionManager(pool, 7, 1)
	require.NoError(t, pm.Run(ctx))

	campaignID := uuid.New()
	_, err := pool.Exec(ctx, "INSERT INTO campaigns (id, name, status) VALUES ($1, $2, $3)", campaignID, "Shutdown Test", "active")
	require.NoError(t, err)

	registry := ads.NewRegistry(queries)
	_, _ = registry.Sync(ctx)

	// 2. Initialize and start components
	eventProc := ads.NewProcessor(queries, cfg.EventBatchSize, cfg.MaxWorkers, 5*time.Second, 5*time.Second)
	eventProc.Start(ctx)

	statsAgg := ads.NewAggregator(queries, 5*time.Second, 5*time.Second, cfg.MaxWorkers)
	statsAgg.Start(ctx)

	router := ads.NewRouter(cfg, registry, eventProc, statsAgg)
	srv := httptest.NewServer(router)
	defer srv.Close()

	// 3. Send bursts of events
	const eventCount = 1234
	var wg sync.WaitGroup
	var acceptedCount int64
	var mu sync.Mutex

	for i := 0; i < eventCount; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			payload := map[string]any{
				"campaign_id": campaignID,
				"type":        "click",
				"payload":     map[string]string{"idx": fmt.Sprintf("%d", idx)},
			}
			body, _ := json.Marshal(payload)
			resp, err := http.Post(srv.URL+"/track", "application/json", bytes.NewBuffer(body))
			if err == nil && resp.StatusCode == http.StatusAccepted {
				mu.Lock()
				acceptedCount++
				mu.Unlock()
			}
			if resp != nil {
				resp.Body.Close()
			}
		}(i)
	}

	wg.Wait()
	require.Equal(t, int64(eventCount), acceptedCount, "All events should be accepted by HTTP layer")

	// 4. Initiate shutdown
	// At this point, most events are in Processor channel or Aggregator memory
	// because flush interval is 5 seconds.
	fmt.Printf("Shutting down... Data should be in memory. Accepted: %d\n", acceptedCount)
	
	shutdownStart := time.Now()
	cancel() // Signal context cancellation
	
	// Close and Wait Processor
	eventProc.Close()
	eventProc.Wait()
	
	// Stop Aggregator
	statsAgg.Stop()
	
	fmt.Printf("Shutdown complete in %v\n", time.Since(shutdownStart))

	// 5. Verify data in DB
	var dbEventCount int64
	err = pool.QueryRow(context.Background(), "SELECT count(*) FROM events WHERE campaign_id = $1", campaignID).Scan(&dbEventCount)
	assert.NoError(t, err)
	assert.Equal(t, acceptedCount, dbEventCount, "Database should contain ALL accepted events after shutdown")

	var dbClickCount int64
	err = pool.QueryRow(context.Background(), "SELECT clicks_count FROM campaign_stats WHERE campaign_id = $1", campaignID).Scan(&dbClickCount)
	assert.NoError(t, err)
	assert.Equal(t, acceptedCount, dbClickCount, "Aggregated stats should also match the accepted count")
}
