package fraud

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

type DetectorConfig struct {
	ScanInterval       time.Duration
	OutboxPendingLimit int64
	ManagementTimeout  time.Duration
	Analyzer           AnalyzerConfig
}

func DefaultDetectorConfig() DetectorConfig {
	return DetectorConfig{
		ScanInterval:       5 * time.Minute,
		OutboxPendingLimit: 500,
		ManagementTimeout:  10 * time.Second,
		Analyzer:           DefaultAnalyzerConfig(),
	}
}

type RunResult struct {
	Candidates int
	Enqueued   int
	Skipped    int
	Backlogged bool
}

type suspiciousFinder = SuspiciousFinder

type Detector struct {
	analyzer   suspiciousFinder
	idem       *IdempotencyStore
	management BlacklistBlocker
	pool       *pgxpool.Pool
	cfg        DetectorConfig
}

func NewDetector(
	analyzer suspiciousFinder,
	idem *IdempotencyStore,
	management BlacklistBlocker,
	pool *pgxpool.Pool,
	cfg DetectorConfig,
) *Detector {
	return &Detector{
		analyzer:   analyzer,
		idem:       idem,
		management: management,
		pool:       pool,
		cfg:        cfg,
	}
}

func (d *Detector) RunLoop(ctx context.Context) error {
	if d == nil {
		return fmt.Errorf("detector: nil receiver")
	}

	interval := d.cfg.ScanInterval
	if interval <= 0 {
		interval = 5 * time.Minute
	}

	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	if _, err := d.Run(ctx); err != nil && !errors.Is(err, ErrOutboxBackpressure) && ctx.Err() == nil {
		slog.Error("ivt detector initial cycle failed", "error", err)
	}

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-ticker.C:
			result, err := d.Run(ctx)
			if errors.Is(err, ErrOutboxBackpressure) {
				slog.Warn("ivt detector paused for outbox backpressure",
					"candidates", result.Candidates,
					"pending_limit", d.cfg.OutboxPendingLimit,
				)
				continue
			}
			if err != nil && ctx.Err() == nil {
				slog.Error("ivt detector cycle failed", "error", err)
				continue
			}
			if result.Enqueued > 0 || result.Candidates > 0 {
				slog.Info("ivt detector cycle complete",
					"candidates", result.Candidates,
					"enqueued", result.Enqueued,
					"skipped", result.Skipped,
				)
			}
		}
	}
}

func (d *Detector) outboxBacklogged(ctx context.Context) (bool, error) {
	pending, err := d.countOutboxBackpressurePending(ctx)
	if err != nil {
		return false, err
	}
	limit := d.cfg.OutboxPendingLimit
	recordOutboxBackpressureState(pending >= limit && limit > 0, pending, limit)
	if limit <= 0 {
		return false, nil
	}
	return pending >= limit, nil
}

func (d *Detector) countOutboxBackpressurePending(ctx context.Context) (int64, error) {
	if d.pool == nil {
		return 0, nil
	}
	var pending int64
	err := d.pool.QueryRow(ctx, outboxBackpressurePendingSQL, OutboxEnforcementEventTypes).Scan(&pending)
	if err != nil {
		return 0, fmt.Errorf("count pending outbox events: %w", err)
	}
	return pending, nil
}

func recordOutboxBackpressureState(active bool, pending, limit int64) {
	if active {
		ivtOutboxBackpressureActive.Set(1)
	} else {
		ivtOutboxBackpressureActive.Set(0)
	}
	ivtOutboxBackpressurePending.Set(float64(pending))
	ivtOutboxBackpressureLimit.Set(float64(limit))
}

func (d *Detector) PendingOutboxCount(ctx context.Context) (int64, error) {
	if d.pool == nil {
		return 0, fmt.Errorf("detector: nil pool")
	}
	var pending int64
	err := d.pool.QueryRow(ctx,
		"SELECT COUNT(*) FROM outbox_events WHERE status = 'PENDING'",
	).Scan(&pending)
	if err != nil {
		return 0, fmt.Errorf("count pending outbox events: %w", err)
	}
	return pending, nil
}
