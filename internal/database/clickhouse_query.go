package database

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"sync/atomic"
	"time"

	"ad-event-processor/internal/metrics"

	"github.com/ClickHouse/clickhouse-go/v2"
	"github.com/ClickHouse/clickhouse-go/v2/lib/driver"
)

const (
	defaultClickHouseQueryMaxMemoryBytes = 1 << 30
	defaultClickHouseQueryMaxExecSeconds = 30
	defaultClickHouseStaleThreshold      = 5 * time.Minute
	defaultClickHouseQueryMaxConcurrency = 8
	defaultClickHouseQueryTimeout        = 30 * time.Second
	defaultClickHouseQuerySlowThreshold  = 2 * time.Second
	clickHouseQueryAcquireTimeout        = 10 * time.Millisecond
)

var ErrClickHouseQueryRejected = errors.New("clickhouse_query: concurrency gate full")

type ClickHouseQueryConfig struct {
	MaxMemoryBytes      uint64
	MaxExecutionTimeSec int
	MaxConcurrency      int
	QueryTimeout        time.Duration
	SlowQueryThreshold  time.Duration
}

// ClickHouseQuery: sem limits concurrent admin/report queries; per-query max_memory via SETTINGS.
type ClickHouseQuery struct {
	conn          driver.Conn
	maxMemory     uint64
	maxExecSec    int
	queryTimeout  time.Duration
	slowThreshold time.Duration
	sem           chan struct{}
	inFlight      atomic.Int32
}

func NewClickHouseQuery(conn driver.Conn, cfg ClickHouseQueryConfig) *ClickHouseQuery {
	maxMem := cfg.MaxMemoryBytes
	if maxMem == 0 {
		maxMem = defaultClickHouseQueryMaxMemoryBytes
	}
	maxExec := cfg.MaxExecutionTimeSec
	if maxExec == 0 {
		maxExec = defaultClickHouseQueryMaxExecSeconds
	}
	timeout := cfg.QueryTimeout
	if timeout <= 0 {
		timeout = defaultClickHouseQueryTimeout
	}
	slow := cfg.SlowQueryThreshold
	if slow <= 0 {
		slow = defaultClickHouseQuerySlowThreshold
	}
	maxConc := cfg.MaxConcurrency
	if maxConc <= 0 {
		maxConc = defaultClickHouseQueryMaxConcurrency
	}
	q := &ClickHouseQuery{
		conn:          conn,
		maxMemory:     maxMem,
		maxExecSec:    maxExec,
		queryTimeout:  timeout,
		slowThreshold: slow,
	}
	if maxConc > 0 {
		q.sem = make(chan struct{}, maxConc)
	}
	return q
}

func (q *ClickHouseQuery) settings() clickhouse.Settings {
	return clickhouse.Settings{
		"readonly":           1,
		"max_memory_usage":   q.maxMemory,
		"max_execution_time": q.maxExecSec,
	}
}

func (q *ClickHouseQuery) withSettings(ctx context.Context) context.Context {
	return clickhouse.Context(ctx, clickhouse.WithSettings(q.settings()))
}

func (q *ClickHouseQuery) acquire(ctx context.Context) error {
	if q == nil || q.sem == nil {
		return nil
	}
	timer := time.NewTimer(clickHouseQueryAcquireTimeout)
	defer timer.Stop()
	select {
	case q.sem <- struct{}{}:
		q.inFlight.Add(1)
		return nil
	case <-timer.C:
		return ErrClickHouseQueryRejected
	case <-ctx.Done():
		return ctx.Err()
	}
}

func (q *ClickHouseQuery) release() {
	if q == nil || q.sem == nil {
		return
	}
	q.inFlight.Add(-1)
	<-q.sem
}

func (q *ClickHouseQuery) observe(start time.Time, query string, err error) {
	elapsed := time.Since(start)
	metrics.ClickHouseQueryDurationSeconds.Observe(elapsed.Seconds())
	if elapsed >= q.slowThreshold {
		slog.Warn("clickhouse_query slow",
			"duration", elapsed,
			"query_prefix", queryPrefix(query),
			"err", err,
		)
	}
}

func queryPrefix(query string) string {
	const maxLen = 120
	if len(query) <= maxLen {
		return query
	}
	return query[:maxLen]
}

func (q *ClickHouseQuery) Query(ctx context.Context, query string, args ...any) (driver.Rows, error) {
	if q == nil || q.conn == nil {
		return nil, fmt.Errorf("clickhouse_query: no connection")
	}
	if err := q.acquire(ctx); err != nil {
		if errors.Is(err, ErrClickHouseQueryRejected) {
			metrics.ClickHouseQueryRejectedTotal.Inc()
		}
		return nil, err
	}

	qctx := ctx
	var cancel context.CancelFunc
	if q.queryTimeout > 0 {
		qctx, cancel = context.WithTimeout(ctx, q.queryTimeout)
	}

	start := time.Now()
	rows, err := q.conn.Query(q.withSettings(qctx), query, args...)
	if err != nil {
		if cancel != nil {
			cancel()
		}
		q.release()
		q.observe(start, query, err)
		return nil, err
	}
	return &governedRows{
		Rows:    rows,
		release: q.release,
		observe: func(closeErr error) { q.observe(start, query, closeErr) },
		cancel:  cancel,
	}, nil
}

func (q *ClickHouseQuery) QueryRow(ctx context.Context, query string, args ...any) driver.Row {
	if q == nil || q.conn == nil {
		return &errRow{err: fmt.Errorf("clickhouse_query: no connection")}
	}
	if err := q.acquire(ctx); err != nil {
		if errors.Is(err, ErrClickHouseQueryRejected) {
			metrics.ClickHouseQueryRejectedTotal.Inc()
		}
		return &errRow{err: err}
	}

	qctx := ctx
	var cancel context.CancelFunc
	if q.queryTimeout > 0 {
		qctx, cancel = context.WithTimeout(ctx, q.queryTimeout)
	}

	start := time.Now()
	row := q.conn.QueryRow(q.withSettings(qctx), query, args...)
	return &governedRow{
		row:     row,
		release: q.release,
		observe: func(err error) { q.observe(start, query, err) },
		cancel:  cancel,
	}
}

func (q *ClickHouseQuery) Exec(ctx context.Context, query string, args ...any) error {
	if q == nil || q.conn == nil {
		return fmt.Errorf("clickhouse_query: no connection")
	}
	if err := q.acquire(ctx); err != nil {
		if errors.Is(err, ErrClickHouseQueryRejected) {
			metrics.ClickHouseQueryRejectedTotal.Inc()
		}
		return err
	}
	defer q.release()

	qctx := ctx
	if q.queryTimeout > 0 {
		var cancel context.CancelFunc
		qctx, cancel = context.WithTimeout(ctx, q.queryTimeout)
		defer cancel()
	}

	start := time.Now()
	err := q.conn.Exec(q.withSettings(qctx), query, args...)
	q.observe(start, query, err)
	return err
}

func (q *ClickHouseQuery) InFlight() int {
	if q == nil {
		return 0
	}
	return int(q.inFlight.Load())
}

func (q *ClickHouseQuery) IngestionLag(ctx context.Context) (time.Duration, error) {
	var latest time.Time
	err := q.QueryRow(ctx, `
SELECT max(latest) FROM (
 SELECT max(created_at) AS latest FROM impressions
 UNION ALL
 SELECT max(created_at) FROM clicks
 UNION ALL
 SELECT max(created_at) FROM conversions
)`).Scan(&latest)
	if err != nil {
		return 0, err
	}
	if latest.IsZero() {
		return 0, nil
	}
	lag := time.Since(latest)
	if lag < 0 {
		return 0, nil
	}
	return lag, nil
}

func Freshness(lag, staleThreshold time.Duration) (stale bool, lagSeconds int) {
	if staleThreshold <= 0 {
		staleThreshold = defaultClickHouseStaleThreshold
	}
	lagSeconds = int(lag.Seconds())
	if lagSeconds < 0 {
		lagSeconds = 0
	}
	return lag > staleThreshold, lagSeconds
}

type errRow struct {
	err error
}

func (r *errRow) Scan(dest ...any) error {
	return r.err
}

func (r *errRow) ScanStruct(dest any) error {
	return r.err
}

func (r *errRow) Err() error {
	return r.err
}

type governedRow struct {
	row     driver.Row
	release func()
	observe func(error)
	cancel  context.CancelFunc
	done    bool
}

func (r *governedRow) finish(err error) {
	if r.done {
		return
	}
	r.done = true
	r.release()
	r.observe(err)
	if r.cancel != nil {
		r.cancel()
	}
}

func (r *governedRow) Scan(dest ...any) error {
	err := r.row.Scan(dest...)
	r.finish(err)
	return err
}

func (r *governedRow) ScanStruct(dest any) error {
	err := r.row.ScanStruct(dest)
	r.finish(err)
	return err
}

func (r *governedRow) Err() error {
	return r.row.Err()
}

type governedRows struct {
	driver.Rows
	release  func()
	observe  func(error)
	cancel   context.CancelFunc
	finished bool
}

func (r *governedRows) finish(err error) {
	if r.finished {
		return
	}
	r.finished = true
	if r.observe != nil {
		r.observe(err)
	}
	if r.release != nil {
		r.release()
		r.release = nil
	}
	if r.cancel != nil {
		r.cancel()
		r.cancel = nil
	}
}

func (r *governedRows) Close() error {
	err := r.Rows.Close()
	r.finish(err)
	return err
}
