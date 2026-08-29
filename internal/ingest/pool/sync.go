package pool

import (
	"context"
	"log/slog"
	"sync/atomic"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

type Sync struct {
	pool     *pgxpool.Pool
	table    *Table
	interval time.Duration
	gen      atomic.Uint64
}

func NewSync(pool *pgxpool.Pool, table *Table, interval time.Duration) *Sync {
	if pool == nil || table == nil {
		return nil
	}
	if interval <= 0 {
		interval = 30 * time.Second
	}
	return &Sync{pool: pool, table: table, interval: interval}
}

func (s *Sync) Start(ctx context.Context) {
	if s == nil {
		return
	}
	s.reloadOnce(ctx)
	ticker := time.NewTicker(s.interval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			s.reloadOnce(ctx)
		}
	}
}

func (s *Sync) reloadOnce(ctx context.Context) {
	rows, err := s.pool.Query(ctx, `
		SELECT pool_id, hostname, status
		FROM domain_pool_domains
		WHERE status IN ('active', 'banned')
		ORDER BY pool_id, sort_order, hostname`)
	if err != nil {
		slog.Warn("domain pool sync query failed", "error", err)
		return
	}
	defer rows.Close()

	var list []SyncRow
	for rows.Next() {
		var row SyncRow
		if err := rows.Scan(&row.PoolID, &row.Hostname, &row.Status); err != nil {
			slog.Warn("domain pool sync scan failed", "error", err)
			return
		}
		list = append(list, row)
	}
	if err := rows.Err(); err != nil {
		slog.Warn("domain pool sync rows failed", "error", err)
		return
	}
	gen := s.gen.Add(1)
	s.table.Publish(BuildSnapshotFromRows(list, gen))
}
