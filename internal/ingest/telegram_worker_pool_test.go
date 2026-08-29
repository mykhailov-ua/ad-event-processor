package ingest

import (
	"testing"
	"time"

	"ad-event-processor/internal/config"
	"ad-event-processor/internal/domain"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

func TestTelegramClickWorkerPoolNoDoubleParse(t *testing.T) {
	WithStaticCampaign(func(campPtr **domain.Campaign) {
		*campPtr = &domain.Campaign{
			ID:         benchClickCampaignID,
			CustomerID: uuid.Nil,
			BrandID:    &benchClickBrandID,
			Location:   (*campPtr).Location,
		}
	})
	cachedMockCamp.Store(nil)

	store := NewBrandCreativeStore(nil, 0)
	store.cache.Store(&brandCreativeMapSnapshot{
		byBrand: map[uuid.UUID][]brandCreativeEntry{
			benchClickBrandID: brandCreativeEntriesReady([]brandCreativeEntry{{
				URL:    "https://offer.example/lp",
				Weight: 100,
			}}),
		},
	})

	cfg := &config.Config{MaxRequestBodySize: 1 << 20}
	h := NewAdsPacketHandler(cfg, &mockRegistry{}, nil, nil, nil, NewJumpHashSharder(1), "fraud-stream", store)
	pool := NewPinnedWorkerPool(4, 1024)
	h.SetWorkerPool(pool)
	defer pool.Shutdown()

	path := "/tg/click?campaign_id=550e8400-e29b-41d4-a716-446655440000&click_id=d5671191-236b-4e94-825e-399185a9bc8d&bridge_token=token_abc123_"
	inbound := BuildGnetHTTP("GET", path, map[string]string{
		"Connection":     "close",
		"Content-Length": "0",
	}, nil)

	parse400 := 0
	ok := 0
	for range 40 {
		_, conn := ServeGnetHarness(h, inbound)
		pool.WaitIdle()
		deadline := time.Now().Add(2 * time.Second)
		for time.Now().Before(deadline) {
			if len(conn.Written()) > 0 {
				break
			}
			time.Sleep(time.Millisecond)
		}
		status := ParseGnetHTTPStatus(conn.Written())
		switch status {
		case 400:
			parse400++
		case 302, 404:
			ok++
		default:
			t.Fatalf("unexpected status %d body=%q", status, string(conn.Written()))
		}
	}
	require.Zero(t, parse400, "worker pool must not double-parse pinned GET /tg/click")
	require.Equal(t, 40, ok)
}
