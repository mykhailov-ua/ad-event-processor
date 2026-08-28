package opsadmin

import (
	"bytes"
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"time"

	"ad-event-processor/internal/database"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type EdgeBlockedFetcher func(ctx context.Context) (map[string]uint64, error)

const (
	filterRejectRollupInterval = time.Hour
	filterRejectRollupTimeout  = 30 * time.Second
)

type FilterRejectRollupWorker struct {
	pool            *pgxpool.Pool
	clickhouseQuery *database.ClickHouseQuery
	url             string
	interval        time.Duration
	client          *http.Client
	fetch           func(ctx context.Context, url string) ([]byte, string, error)
	fetchEdge       func(ctx context.Context) (map[string]uint64, error)
}

func NewFilterRejectRollupWorker(pool *pgxpool.Pool, clickhouseQuery *database.ClickHouseQuery, scrapeURL string) *FilterRejectRollupWorker {
	return &FilterRejectRollupWorker{
		pool:            pool,
		clickhouseQuery: clickhouseQuery,
		url:             ResolveFilterRejectMetricsURL(scrapeURL),
		interval:        filterRejectRollupInterval,
		client:          &http.Client{Timeout: 15 * time.Second},
	}
}

func (w *FilterRejectRollupWorker) SetEdgeFetcher(fetch EdgeBlockedFetcher) {
	if w != nil {
		w.fetchEdge = fetch
	}
}

func (w *FilterRejectRollupWorker) Start(ctx context.Context) {
	if w == nil || w.pool == nil || w.clickhouseQuery == nil || w.url == "" {
		return
	}
	slog.Info("filter reject rollup worker starting", "url", w.url, "interval", w.interval)
	ticker := time.NewTicker(w.interval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case t := <-ticker.C:
			if err := w.runHour(ctx, t.UTC()); err != nil {
				slog.Error("filter reject rollup failed", "err", err)
			}
		}
	}
}

func (w *FilterRejectRollupWorker) runHour(ctx context.Context, now time.Time) error {
	runCtx, cancel := context.WithTimeout(ctx, filterRejectRollupTimeout)
	defer cancel()

	fetch := w.fetch
	if fetch == nil {
		fetch = func(ctx context.Context, url string) ([]byte, string, error) {
			return fetchPrometheusMetrics(ctx, w.client, url)
		}
	}
	body, contentType, err := fetch(runCtx, w.url)
	if err != nil {
		return fmt.Errorf("fetch tracker metrics: %w", err)
	}
	snap, err := parseFilterRejectMetrics(bytes.NewReader(body), contentType)
	if err != nil {
		return fmt.Errorf("parse filter reject metrics: %w", err)
	}
	tracker := snap.Totals
	sliceCurrent := mergeFilterRejectSliceSamples(snap.Slices)

	var edge map[string]float64
	if w.fetchEdge != nil {
		if blocked, err := w.fetchEdge(runCtx); err != nil {
			slog.Warn("filter reject rollup edge metrics skipped", "err", err)
		} else {
			edge = edgeBlockedToRejectCounters(blocked)
		}
	}
	current := mergeRejectCounterMaps(tracker, edge)
	if len(current) == 0 && len(sliceCurrent) == 0 {
		return nil
	}

	previous, hadPrevious, err := w.loadCounterState(runCtx)
	if err != nil {
		return err
	}
	prevSlices, hadPrevSlices, err := w.loadSliceCounterState(runCtx)
	if err != nil {
		return err
	}

	rollupHour := now.Truncate(time.Hour)
	if hadPrevious {
		if err := w.insertRollups(runCtx, rollupHour, previous, current); err != nil {
			return err
		}
	}
	if hadPrevSlices && len(sliceCurrent) > 0 {
		if err := w.insertSliceRollups(runCtx, rollupHour, prevSlices, sliceCurrent); err != nil {
			return err
		}
	}
	if len(current) > 0 {
		if err := w.storeCounterState(runCtx, current); err != nil {
			return err
		}
	}
	if len(sliceCurrent) > 0 {
		return w.storeSliceCounterState(runCtx, sliceCurrent)
	}
	return nil
}

func (w *FilterRejectRollupWorker) loadSliceCounterState(ctx context.Context) (map[string]float64, bool, error) {
	rows, err := w.pool.Query(ctx, `SELECT slice_key, counter_value FROM ops.filter_reject_slice_counters`)
	if err != nil {
		return nil, false, fmt.Errorf("load filter reject slice counters: %w", err)
	}
	defer rows.Close()

	out := make(map[string]float64)
	for rows.Next() {
		var key string
		var value float64
		if err := rows.Scan(&key, &value); err != nil {
			return nil, false, err
		}
		out[key] = value
	}
	if err := rows.Err(); err != nil {
		return nil, false, err
	}
	return out, len(out) > 0, nil
}

func (w *FilterRejectRollupWorker) storeSliceCounterState(ctx context.Context, current map[string]float64) error {
	const upsertSQL = `
INSERT INTO ops.filter_reject_slice_counters (slice_key, counter_value, updated_at)
VALUES ($1, $2, NOW())
ON CONFLICT (slice_key) DO UPDATE
SET counter_value = EXCLUDED.counter_value,
 updated_at = NOW()`
	batch := &pgx.Batch{}
	for key, value := range current {
		batch.Queue(upsertSQL, key, value)
	}
	br := w.pool.SendBatch(ctx, batch)
	var batchErr error
	for range current {
		if _, err := br.Exec(); err != nil && batchErr == nil {
			batchErr = fmt.Errorf("store filter reject slice counters: %w", err)
		}
	}
	if closeErr := br.Close(); closeErr != nil && batchErr == nil {
		batchErr = closeErr
	}
	return batchErr
}

func (w *FilterRejectRollupWorker) insertSliceRollups(
	ctx context.Context,
	rollupHour time.Time,
	previous, current map[string]float64,
) error {
	for key, delta := range filterRejectRollupDeltas(previous, current) {
		if delta == 0 {
			continue
		}
		kind, country := splitFilterRejectSliceKey(key)
		if err := w.clickhouseQuery.Exec(ctx, `
INSERT INTO filter_reject_slices (rollup_hour, reject_kind, placement_id, country, reject_count)
VALUES (?, ?, '', ?, ?)`, rollupHour, kind, country, delta); err != nil {
			return fmt.Errorf("insert filter reject slice key=%s: %w", key, err)
		}
	}
	return nil
}

func (w *FilterRejectRollupWorker) loadCounterState(ctx context.Context) (map[string]float64, bool, error) {
	rows, err := w.pool.Query(ctx, `SELECT reject_kind, counter_value FROM ops.filter_reject_counters`)
	if err != nil {
		return nil, false, fmt.Errorf("load filter reject counters: %w", err)
	}
	defer rows.Close()

	out := make(map[string]float64)
	for rows.Next() {
		var kind string
		var value float64
		if err := rows.Scan(&kind, &value); err != nil {
			return nil, false, err
		}
		out[kind] = value
	}
	if err := rows.Err(); err != nil {
		return nil, false, err
	}
	return out, len(out) > 0, nil
}

func (w *FilterRejectRollupWorker) storeCounterState(ctx context.Context, current map[string]float64) error {
	const upsertSQL = `
INSERT INTO ops.filter_reject_counters (reject_kind, counter_value, updated_at)
VALUES ($1, $2, NOW())
ON CONFLICT (reject_kind) DO UPDATE
SET counter_value = EXCLUDED.counter_value,
 updated_at = NOW()`
	batch := &pgx.Batch{}
	for kind, value := range current {
		batch.Queue(upsertSQL, kind, value)
	}
	br := w.pool.SendBatch(ctx, batch)
	var batchErr error
	for range current {
		if _, err := br.Exec(); err != nil && batchErr == nil {
			batchErr = fmt.Errorf("store filter reject counters: %w", err)
		}
	}
	if closeErr := br.Close(); closeErr != nil && batchErr == nil {
		batchErr = closeErr
	}
	return batchErr
}

func (w *FilterRejectRollupWorker) insertRollups(
	ctx context.Context,
	rollupHour time.Time,
	previous, current map[string]float64,
) error {
	for kind, delta := range filterRejectRollupDeltas(previous, current) {
		if err := w.clickhouseQuery.Exec(ctx, `
INSERT INTO filter_reject_rollups (rollup_hour, reject_kind, reject_count)
VALUES (?, ?, ?)`, rollupHour, kind, delta); err != nil {
			return fmt.Errorf("insert filter reject rollup kind=%s: %w", kind, err)
		}
	}
	return nil
}
