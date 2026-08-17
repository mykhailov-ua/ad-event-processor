package ingestion

import (
	"context"
	"log/slog"
	"sort"
	"sync/atomic"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

type domainPoolSyncRow struct {
	poolID   uuid.UUID
	hostname string
	status   string
}

type domainPoolSync struct {
	pool     *pgxpool.Pool
	table    *DomainPoolTable
	interval time.Duration
	gen      atomic.Uint64
}

// NewDomainPoolSync builds the cold-path PG sync worker. Returns nil when disabled.
func NewDomainPoolSync(pool *pgxpool.Pool, table *DomainPoolTable, interval time.Duration) *domainPoolSync {
	if pool == nil || table == nil {
		return nil
	}
	if interval <= 0 {
		interval = 30 * time.Second
	}
	return &domainPoolSync{pool: pool, table: table, interval: interval}
}

func (s *domainPoolSync) Start(ctx context.Context) {
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

func (s *domainPoolSync) reloadOnce(ctx context.Context) {
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

	var list []domainPoolSyncRow
	for rows.Next() {
		var row domainPoolSyncRow
		if err := rows.Scan(&row.poolID, &row.hostname, &row.status); err != nil {
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
	s.table.Publish(buildDomainPoolSnapshotFromRows(list, gen))
}

func buildDomainPoolSnapshotFromRows(rows []domainPoolSyncRow, gen uint64) *domainPoolSnapshot {
	if len(rows) == 0 {
		return &domainPoolSnapshot{gen: gen}
	}
	type poolKey = uuid.UUID
	pools := map[poolKey]*domainPoolRecord{}
	order := make([]poolKey, 0, 4)
	for _, row := range rows {
		rec, ok := pools[row.poolID]
		if !ok {
			rec = &domainPoolRecord{id: int32(len(order)), domains: make([]domainPoolDomain, 0, 4)}
			pools[row.poolID] = rec
			order = append(order, row.poolID)
		}
		var st uint8
		switch row.status {
		case "banned":
			st = domainPoolStatusBanned
		case "active":
			st = domainPoolStatusActive
		default:
			continue
		}
		host := normalizePoolHostname(row.hostname)
		if host == "" {
			continue
		}
		rec.domains = append(rec.domains, domainPoolDomain{host: host, status: st})
	}
	outPools := make([]domainPoolRecord, 0, len(order))
	hosts := make([]hostPoolEntry, 0, len(rows))
	for _, key := range order {
		rec := pools[key]
		if len(rec.domains) == 0 {
			continue
		}
		rec.id = int32(len(outPools))
		outPools = append(outPools, *rec)
		for i, d := range rec.domains {
			hosts = append(hosts, hostPoolEntry{
				host:      d.host,
				poolIdx:   rec.id,
				domainIdx: int32(i),
				status:    d.status,
			})
		}
	}
	sort.Slice(hosts, func(i, j int) bool { return hosts[i].host < hosts[j].host })
	return &domainPoolSnapshot{gen: gen, pools: outPools, hosts: hosts}
}
