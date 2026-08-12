package controlplane

import (
	"github.com/bidshard/ad-event-processor/pkg/faultproof"

	"context"
	"strconv"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/bidshard/ad-event-processor/internal/config"
	"github.com/bidshard/ad-event-processor/internal/database"
	"github.com/bidshard/ad-event-processor/internal/domain"
	"github.com/bidshard/ad-event-processor/internal/domain/db"
	"github.com/bidshard/ad-event-processor/internal/rtb"

	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestFault_rtb_catalog_reload_outbox(t *testing.T) {
	t.Parallel()
	if testing.Short() {
		t.Skip("fault integration test")
	}

	pool, cleanupDB := database.SetupTestDB(t)
	defer cleanupDB()
	rdb, cleanupRedis := database.SetupTestRedis(t)
	defer cleanupRedis()

	cfg := &config.Config{AdminAPIKey: "test-secret"}
	svc := newBareService(t, pool, []redis.UniversalClient{rdb}, cfg)
	ctx := context.Background()
	customerID := uuid.New()
	require.NoError(t, svc.CreateCustomer(ctx, customerID, "Test RTB", 1_000_000, "USD"))

	created, err := svc.CreateRtbDeal(ctx, RtbDealCreateSpec{
		DealID:     "fault-reload-deal",
		FloorMicro: 100_000,
		CustomerID: customerID.String(),
	})
	require.NoError(t, err)

	catalog := rtb.NewDealIndex()
	require.NoError(t, domain.ReloadRtbDeals(ctx, db.New(pool), catalog))
	deal, ok := catalog.Lookup("fault-reload-deal")
	require.True(t, ok)
	require.Equal(t, int64(100_000), deal.FloorMicro)

	channel := domain.RtbCatalogReloadChannel(svc.cfg)
	sub := rdb.Subscribe(ctx, channel)
	defer sub.Close()
	_, err = sub.Receive(ctx)
	require.NoError(t, err)

	_, err = pool.Exec(ctx, "DELETE FROM outbox_events")
	require.NoError(t, err)

	updated, err := svc.UpdateRtbDeal(ctx, created.ID, RtbDealUpdateSpec{
		DealID:     "fault-reload-deal",
		FloorMicro: 275_000,
		CustomerID: customerID.String(),
	})
	require.NoError(t, err)
	_ = updated

	const workers = 24
	var lookups atomic.Uint64
	var wg sync.WaitGroup
	stop := make(chan struct{})
	wg.Add(workers)
	for range workers {
		go func() {
			defer wg.Done()
			for {
				select {
				case <-stop:
					return
				default:
					if d, found := catalog.Lookup("fault-reload-deal"); found {
						if d.FloorMicro == 100_000 || d.FloorMicro == 275_000 {
							lookups.Add(1)
						}
					}
				}
			}
		}()
	}

	worker := NewOutboxWorker(svc)
	n, err := worker.ProcessOutboxWithCount(ctx, 10)
	require.NoError(t, err)
	require.Equal(t, 1, n)

	msg, err := sub.ReceiveTimeout(ctx, 2*time.Second)
	require.NoError(t, err)
	m, ok := msg.(*redis.Message)
	require.True(t, ok)
	assert.Equal(t, "reload", m.Payload)

	require.NoError(t, domain.ReloadRtbDeals(ctx, db.New(pool), catalog))
	close(stop)
	wg.Wait()

	reloaded, ok := catalog.Lookup("fault-reload-deal")
	require.True(t, ok)
	assert.Equal(t, int64(275_000), reloaded.FloorMicro)
	assert.Greater(t, lookups.Load(), uint64(0))

	faultproof.Log(t, "rtb_catalog_reload_outbox", map[string]string{
		"subsystem":       "management_rtb",
		"baseline_ok":     "true",
		"fault_type":      "deal_update_outbox_redis",
		"workers":         "24",
		"floor_before":    "100000",
		"floor_after":     "275000",
		"pubsub_received": "true",
		"lookups":         strconv.FormatUint(lookups.Load(), 10),
	})
}
