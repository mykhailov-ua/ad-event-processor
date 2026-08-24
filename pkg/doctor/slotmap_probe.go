package doctor

import (
	"context"
	"fmt"
	"time"

	"ad-event-processor/internal/config"
	"ad-event-processor/internal/database"
	"ad-event-processor/internal/domain"

	"github.com/jackc/pgx/v5/pgxpool"
)

type SlotMapProbe struct {
	Deps ProbeDeps
}

func (SlotMapProbe) Name() string { return "slotmap" }

func (p SlotMapProbe) Run(ctx context.Context) Result {
	start := time.Now()
	latency := func() int64 { return time.Since(start).Milliseconds() }

	if p.Deps.Config == nil {
		return Result{Name: "slotmap", Status: StatusSkip, Detail: "config not loaded", Latency: latency()}
	}

	pool, closePool, err := p.pgPool(ctx)
	if err != nil {
		return Result{Name: "slotmap", Status: StatusSkip, Detail: err.Error(), Latency: latency()}
	}
	if closePool != nil {
		defer closePool()
	}

	var pgMap domain.OpsSlotMapResponse
	if p.Deps.SlotMapFromPG != nil {
		pgMap, err = p.Deps.SlotMapFromPG(ctx)
	} else {
		pgMap, err = domain.LoadOpsSlotMapFromPool(ctx, pool)
	}
	if err != nil {
		return Result{
			Name:    "slotmap",
			Status:  StatusFail,
			Detail:  fmt.Sprintf("postgres slot map: %v", err),
			Latency: latency(),
		}
	}

	numShards := len(p.Deps.Config.RedisAddrs)
	if numShards <= 0 {
		numShards = config.ExpectedRedisShardCount
	}

	sharder := domain.NewStaticSlotSharder(numShards)
	if err := domain.ApplySlotMapToSharder(sharder, pgMap); err != nil {
		return Result{Name: "slotmap", Status: StatusFail, Detail: err.Error(), Latency: latency()}
	}

	if mismatches := domain.CheckSlotMapRoutingParity(sharder, pgMap.Slots, domain.DefaultSlotMapParitySamples); mismatches > 0 {
		return Result{
			Name:   "slotmap",
			Status: StatusFail,
			Detail: fmt.Sprintf("CRC32C routing parity: %d/%d mismatches on postgres table",
				mismatches, domain.DefaultSlotMapParitySamples),
			Latency: latency(),
		}
	}

	baseURL := p.Deps.Config.ManagementURL
	if baseURL == "" {
		return Result{
			Name:    "slotmap",
			Status:  StatusWarn,
			Detail:  fmt.Sprintf("postgres v%d ok; edge HTTP check skipped (set CONTROL_URL)", pgMap.Version),
			Latency: latency(),
		}
	}

	httpMap, err := p.fetchSlotMapHTTP(ctx, baseURL)
	if err != nil {
		return Result{
			Name:    "slotmap",
			Status:  StatusWarn,
			Detail:  fmt.Sprintf("postgres v%d ok; edge sync endpoint: %v", pgMap.Version, err),
			Latency: latency(),
		}
	}

	if pgMap.Version != httpMap.Version && httpMap.Version != 0 {
		return Result{
			Name:   "slotmap",
			Status: StatusFail,
			Detail: fmt.Sprintf("version drift: postgres=%d http=%d (nginx edge-slot-map.lua may be stale)",
				pgMap.Version, httpMap.Version),
			Latency: latency(),
		}
	}

	if diffs, firstSlot := domain.CompareSlotMaps(pgMap.Slots, httpMap.Slots); diffs > 0 {
		return Result{
			Name:   "slotmap",
			Status: StatusFail,
			Detail: fmt.Sprintf("%d slot table diffs (first slot %d); edge nginx table != control postgres",
				diffs, firstSlot),
			Latency: latency(),
		}
	}

	return Result{
		Name:   "slotmap",
		Status: StatusPass,
		Detail: fmt.Sprintf("v%d: postgres and /ops/shards/slot-map match; %d CRC32C samples ok",
			pgMap.Version, domain.DefaultSlotMapParitySamples),
		Latency: latency(),
	}
}

func (p SlotMapProbe) pgPool(ctx context.Context) (*pgxpool.Pool, func(), error) {
	if p.Deps.SlotMapFromPG != nil {
		return nil, nil, nil
	}
	if p.Deps.PGPool != nil {
		pool, err := p.Deps.PGPool(ctx)
		if err != nil {
			return nil, nil, err
		}
		if pool == nil {
			return nil, nil, fmt.Errorf("postgres pool unavailable")
		}
		return pool, nil, nil
	}
	if string(p.Deps.Config.DBDSN) == "" {
		return nil, nil, fmt.Errorf("DB_DSN not configured")
	}
	pool, err := database.Connect(ctx, string(p.Deps.Config.DBDSN), 2, 1)
	if err != nil {
		return nil, nil, err
	}
	return pool, func() { pool.Close() }, nil
}

func (p SlotMapProbe) fetchSlotMapHTTP(ctx context.Context, baseURL string) (domain.OpsSlotMapResponse, error) {
	if p.Deps.SlotMapFromHTTP != nil {
		return p.Deps.SlotMapFromHTTP(ctx, baseURL)
	}
	return domain.FetchOpsSlotMapHTTP(ctx, nil, baseURL)
}
