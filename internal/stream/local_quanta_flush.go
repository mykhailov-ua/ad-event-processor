package stream

import (
	"context"
	_ "embed"
	"fmt"
	"log/slog"
	"time"

	"ad-event-processor/internal/domain"
	"ad-event-processor/internal/metrics"

	"github.com/google/uuid"
	redis "github.com/redis/go-redis/v9"
)

//go:embed local-quota-return.lua
var localQuotaReturnLua string

// local-quota-return.lua (embedded):
//
//	KEYS[1] = budget quota key
//	ARGV[1] = micro-units to return from local ledger to Redis (INCRBY)
//	Returns: new quota balance after INCRBY (or current GET when amt <= 0).
//
// Verify: redis-cli --eval internal/stream/local-quota-return.lua , <quota_key> , <amount_micro>
var localQuotaReturnScript = redis.NewScript(localQuotaReturnLua)

var LocalQuotaReturnScript = localQuotaReturnScript

const (
	FlushReasonPause    = "pause"
	FlushReasonShutdown = "shutdown"
	FlushReasonStrict   = "strict"
)

// TakeRemaining atomically drains all sub-slots (0..3) for one campaign; used before Redis return.
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

// LocalQuantaFlusher returns unused local chunks to Redis via local-quota-return.lua on pause,
// shutdown, or strict-mode entry. Publishes return deltas for cross-region budget sync when wired.
type LocalQuantaFlusher struct {
	ledger      *LocalQuantaLedger
	redisShards []redis.UniversalClient
	sharder     Sharder
	registry    domain.CampaignRegistry
	publisher   *BudgetDeltaPublisher
}

func NewLocalQuantaFlusher(
	ledger *LocalQuantaLedger,
	redisShards []redis.UniversalClient,
	sharder Sharder,
	publisher *BudgetDeltaPublisher,
) *LocalQuantaFlusher {
	if ledger == nil || len(redisShards) == 0 || sharder == nil {
		return nil
	}
	return &LocalQuantaFlusher{
		ledger:      ledger,
		redisShards: redisShards,
		sharder:     sharder,
		publisher:   publisher,
	}
}

func (f *LocalQuantaFlusher) SetCampaignRegistry(reg domain.CampaignRegistry) {
	if f == nil {
		return
	}
	f.registry = reg
}

// FlushLocalQuanta drains ledger for one campaign and INCRBY quota on the campaign shard.
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

// FlushAll walks every occupied ledger cell on shutdown; reason FlushReasonShutdown.
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
	for sub := range n {
		amt := perSlot
		if sub == 0 {
			amt += rem
		}
		if amt <= 0 {
			continue
		}
		shard := spreadHighVolumeShard(len(f.redisShards), campaignID, sub)
		if err := f.returnToRedisSlot(ctx, campaignID, amt, shard, sub); err != nil && firstErr == nil {
			firstErr = err
		}
	}
	return firstErr
}

// returnToRedisSlot runs embedded local-quota-return.lua on the debit sub-slot quota key.
func (f *LocalQuantaFlusher) returnToRedisSlot(ctx context.Context, campaignID uuid.UUID, amount int64, shard, subSlot int) error {
	if amount <= 0 {
		return nil
	}
	if shard < 0 || shard >= len(f.redisShards) || f.redisShards[shard] == nil {
		return fmt.Errorf("invalid shard %d", shard)
	}
	quotaKey := budgetQuotaKeyForDebit(campaignID, subSlot)
	runCtx, cancel := context.WithTimeout(ctx, 2*time.Second)
	defer cancel()
	_, err := localQuotaReturnScript.Run(runCtx, f.redisShards[shard], []string{quotaKey}, amount).Result()
	return err
}

// AdaptiveChunkSizeStrict shrinks refill floor when Redis remaining nears strictEnter threshold.
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
