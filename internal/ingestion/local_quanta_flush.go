package ingestion

import (
	"context"
	_ "embed"
	"fmt"
	"log/slog"
	"sync/atomic"
	"time"

	"github.com/bidshard/ad-event-processor/internal/domain"
	"github.com/bidshard/ad-event-processor/internal/metrics"

	"github.com/google/uuid"
	redis "github.com/redis/go-redis/v9"
)

//go:embed local-quota-return.lua
var localQuotaReturnLua string

var localQuotaReturnScript = redis.NewScript(localQuotaReturnLua)

const (
	FlushReasonPause    = "pause"
	FlushReasonShutdown = "shutdown"
	FlushReasonStrict   = "strict"
)

func (l *LocalQuantaLedger) TakeRemaining(id uuid.UUID) int64 {
	if l == nil {
		return 0
	}
	var total int64
	for sub := range 4 {
		total += l.takeRemainingDebit(id, sub)
	}
	return total
}

func (l *LocalQuantaLedger) takeRemainingDebit(id uuid.UUID, subSlot int) int64 {
	cell, h := l.cellForDebit(id, subSlot)
	if cell.campaignHash != h {
		return 0
	}
	for {
		rem := cell.remaining.Load()
		if rem <= 0 {
			return 0
		}
		if cell.remaining.CompareAndSwap(rem, 0) {
			return rem
		}
	}
}

func (l *LocalQuantaLedger) FlushOccupied(fn func(campaignID uuid.UUID, remaining int64)) {
	if l == nil || fn == nil {
		return
	}
	for i := range l.cells {
		cell := &l.cells[i]
		if cell.campaignHash == 0 {
			continue
		}
		id := cell.campaignID
		for {
			rem := cell.remaining.Load()
			if rem <= 0 {
				break
			}
			if cell.remaining.CompareAndSwap(rem, 0) {
				if id != uuid.Nil {
					fn(id, rem)
				}
				break
			}
		}
	}
}

type LocalQuantaFlusher struct {
	ledger    *LocalQuantaLedger
	rdbs      []redis.UniversalClient
	sharder   Sharder
	registry  domain.CampaignRegistry
	publisher *BudgetDeltaPublisher
}

func NewLocalQuantaFlusher(
	ledger *LocalQuantaLedger,
	rdbs []redis.UniversalClient,
	sharder Sharder,
	publisher *BudgetDeltaPublisher,
) *LocalQuantaFlusher {
	if ledger == nil || len(rdbs) == 0 || sharder == nil {
		return nil
	}
	return &LocalQuantaFlusher{
		ledger:    ledger,
		rdbs:      rdbs,
		sharder:   sharder,
		publisher: publisher,
	}
}

func (f *LocalQuantaFlusher) SetCampaignRegistry(reg domain.CampaignRegistry) {
	if f == nil {
		return
	}
	f.registry = reg
}

func (f *LocalQuantaFlusher) FlushLocalQuanta(ctx context.Context, campaignID uuid.UUID, reason string) int64 {
	if f == nil || f.ledger == nil {
		return 0
	}
	taken := f.ledger.TakeRemaining(campaignID)
	if taken <= 0 {
		return 0
	}
	if err := f.returnToRedis(ctx, campaignID, taken); err != nil {
		slog.Warn("local quanta flush redis return failed", "campaign_id", campaignID, "amount", taken, "error", err)
	}
	if f.publisher != nil {
		f.publisher.PublishReturn(campaignID, taken)
	}
	if reason == "" {
		reason = FlushReasonPause
	}
	metrics.LocalQuotaFlushTotal.WithLabelValues(reason).Inc()
	return taken
}

func (f *LocalQuantaFlusher) FlushAll(ctx context.Context) int {
	if f == nil || f.ledger == nil {
		return 0
	}
	n := 0
	f.ledger.FlushOccupied(func(id uuid.UUID, remaining int64) {
		if remaining <= 0 || id == uuid.Nil {
			return
		}
		if err := f.returnToRedis(ctx, id, remaining); err != nil {
			slog.Warn("local quanta flush-all redis return failed", "campaign_id", id, "error", err)
		}
		if f.publisher != nil {
			f.publisher.PublishReturn(id, remaining)
		}
		metrics.LocalQuotaFlushTotal.WithLabelValues(FlushReasonShutdown).Inc()
		n++
	})
	return n
}

func (f *LocalQuantaFlusher) returnToRedis(ctx context.Context, campaignID uuid.UUID, amount int64) error {
	if amount <= 0 {
		return nil
	}
	n := 0
	if f.registry != nil {
		if camp, ok := f.registry.GetCampaign(campaignID); ok {
			n = camp.DebitSubShardCount()
		}
	}
	if n <= 1 {
		shard := f.sharder.GetShard(campaignID)
		return f.returnToRedisSlot(ctx, campaignID, amount, shard, 0)
	}
	perSlot := amount / int64(n)
	rem := amount % int64(n)
	var firstErr error
	for sub := 0; sub < n; sub++ {
		amt := perSlot
		if sub == 0 {
			amt += rem
		}
		if amt <= 0 {
			continue
		}
		shard := spreadHighVolumeShard(len(f.rdbs), campaignID, sub)
		if err := f.returnToRedisSlot(ctx, campaignID, amt, shard, sub); err != nil && firstErr == nil {
			firstErr = err
		}
	}
	return firstErr
}

func (f *LocalQuantaFlusher) returnToRedisSlot(ctx context.Context, campaignID uuid.UUID, amount int64, shard, subSlot int) error {
	if amount <= 0 {
		return nil
	}
	if shard < 0 || shard >= len(f.rdbs) || f.rdbs[shard] == nil {
		return fmt.Errorf("invalid shard %d", shard)
	}
	quotaKey := budgetQuotaKeyForDebit(campaignID, subSlot)
	opCtx := ctx
	if opCtx == nil {
		var cancel context.CancelFunc
		opCtx, cancel = context.WithTimeout(context.Background(), 2*time.Second)
		defer cancel()
	}
	_, err := localQuotaReturnScript.Run(opCtx, f.rdbs[shard], []string{quotaKey}, amount).Result()
	return err
}

func AdaptiveChunkSizeStrict(emaRPS float64, floorMicro, ceilingMicro, baseChunk, redisRemaining, strictThreshold int64) int64 {
	floor := floorMicro
	if strictThreshold > 0 && redisRemaining > 0 && redisRemaining < strictThreshold*2 {
		half := floorMicro / 2
		if half < 100_000 {
			half = 100_000
		}
		if half < floor {
			floor = half
		}
		if redisRemaining < strictThreshold {
			quarter := floorMicro / 4
			if quarter < 50_000 {
				quarter = 50_000
			}
			if quarter < floor {
				floor = quarter
			}
		}
	}
	return AdaptiveChunkSize(emaRPS, floor, ceilingMicro, baseChunk)
}

var registryQuantaFlush atomic.Pointer[func(uuid.UUID)]

func SetRegistryQuantaFlushHook(fn func(uuid.UUID)) {
	if fn == nil {
		registryQuantaFlush.Store(nil)
		return
	}
	registryQuantaFlush.Store(&fn)
}

func invokeRegistryQuantaFlush(id uuid.UUID) {
	p := registryQuantaFlush.Load()
	if p == nil || *p == nil {
		return
	}
	(*p)(id)
}
