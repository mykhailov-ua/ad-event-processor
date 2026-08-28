package fraudadmin

import (
	"context"
	"log/slog"
	"time"

	db "ad-event-processor/internal/domain/db"

	"github.com/jackc/pgx/v5/pgxpool"
)

const blacklistJanitorBatchSize = 200

const blacklistJanitorBatchTimeout = 2 * time.Minute

type BlacklistJanitorAlerter interface {
	AlertBlacklistJanitorFailed(ctx context.Context, err error)
}

type BlacklistJanitorHost interface {
	Pool() *pgxpool.Pool
	BlacklistJanitorAlerter() BlacklistJanitorAlerter
	UnblockExpiredBlacklist(ctx context.Context, rows []db.ListExpiredBlacklistIPsRow) (int, error)
}

type BlacklistJanitor struct {
	host     BlacklistJanitorHost
	interval time.Duration
}

func NewBlacklistJanitor(host BlacklistJanitorHost, interval time.Duration) *BlacklistJanitor {
	if interval <= 0 {
		interval = time.Minute
	}
	return &BlacklistJanitor{host: host, interval: interval}
}

func (j *BlacklistJanitor) Interval() time.Duration {
	if j == nil {
		return 0
	}
	return j.interval
}

func (j *BlacklistJanitor) Start(ctx context.Context) {
	if j == nil || j.host == nil {
		return
	}

	ticker := time.NewTicker(j.interval)
	defer ticker.Stop()

	j.runOnce(ctx)

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			j.runOnce(ctx)
		}
	}
}

func (j *BlacklistJanitor) runOnce(ctx context.Context) {
	opCtx, cancel := context.WithTimeout(ctx, blacklistJanitorBatchTimeout)
	defer cancel()

	rows, err := db.New(j.host.Pool()).ListExpiredBlacklistIPs(opCtx, blacklistJanitorBatchSize)
	if err != nil {
		slog.Error("blacklist janitor scan failed", "error", err)
		if alerter := j.host.BlacklistJanitorAlerter(); alerter != nil {
			alerter.AlertBlacklistJanitorFailed(opCtx, err)
		}
		return
	}
	if len(rows) == 0 {
		return
	}

	removed, err := j.host.UnblockExpiredBlacklist(opCtx, rows)
	if err != nil {
		slog.Warn("blacklist janitor batch unblock failed", "error", err)
		return
	}

	slog.Info("blacklist janitor cycle complete",
		"expired_found", len(rows),
		"removed", removed,
	)
}
