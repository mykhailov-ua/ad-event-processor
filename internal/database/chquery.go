package database

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"sync/atomic"
	"time"

	"espx/internal/metrics"

	"github.com/ClickHouse/clickhouse-go/v2"
	"github.com/ClickHouse/clickhouse-go/v2/lib/driver"
)

const (
	defaultCHQueryMaxMemoryBytes = 1 << 30
	defaultCHQueryMaxExecSeconds = 30
	defaultCHStaleThreshold      = 5 * time.Minute
	defaultCHQueryMaxConcurrency = 8
	defaultCHQueryTimeout        = 30 * time.Second
	defaultCHQuerySlowThreshold  = 2 * time.Second
	chQueryAcquireTimeout        = 10 * time.Millisecond
)

var ErrCHQueryRejected = errors.New("chquery: concurrency gate full")

type CHQueryConfig struct {
	MaxMemoryBytes      uint64
	MaxExecutionTimeSec int
	MaxConcurrency      int
	QueryTimeout        time.Duration
	SlowQueryThreshold  time.Duration
}

type CHQuery struct {
	conn          driver.Conn
	maxMemory     uint64
	maxExecSec    int
	queryTimeout  time.Duration
	slowThreshold time.Duration
	sem           chan struct{}
	inFlight      atomic.Int32
}

func NewCHQuery(conn driver.Conn, cfg CHQueryConfig) *CHQuery {
	maxMem := cfg.MaxMemoryBytes
	if maxMem == 0 {
		maxMem = defaultCHQueryMaxMemoryBytes
	}
	maxExec := cfg.MaxExecutionTimeSec
	if maxExec == 0 {
		maxExec = defaultCHQueryMaxExecSeconds
	}
	timeout := cfg.QueryTimeout
	if timeout <= 0 {
		timeout = defaultCHQueryTimeout
	}
	slow := cfg.SlowQueryThreshold
	if slow <= 0 {
		slow = defaultCHQuerySlowThreshold
	}
	maxConc := cfg.MaxConcurrency
	if maxConc <= 0 {
		maxConc = defaultCHQueryMaxConcurrency
	}
	q := &CHQuery{
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

func (q *CHQuery) settings() clickhouse.Settings {
	return clickhouse.Settings{
		"readonly":           1,
		"max_memory_usage":   q.maxMemory,
		"max_execution_time": q.maxExecSec,
	}
}

func (q *CHQuery) withSettings(ctx context.Context) context.Context {
	return clickhouse.Context(ctx, clickhouse.WithSettings(q.settings()))
}

func (q *CHQuery) acquire(ctx context.Context) error {
	if q == nil || q.sem == nil {
		return nil
	}
	timer := time.NewTimer(chQueryAcquireTimeout)
	defer timer.Stop()
	select {
	case q.sem <- struct{}{}:
		q.inFlight.Add(1)
		return nil
	case <-timer.C:
		return ErrCHQueryRejected
	case <-ctx.Done():
		return ctx.Err()
	}
}

func (q *CHQuery) release() {
	if q == nil || q.sem == nil {
		return
	}
	q.inFlight.Add(-1)
	<-q.sem
}

func (q *CHQuery) observe(start time.Time, query string, err error) {
	elapsed := time.Since(start)
	metrics.CHQueryDurationSeconds.Observe(elapsed.Seconds())
	if elapsed >= q.slowThreshold {
		slog.Warn("chquery slow",
			"duration", elapsed,
			"query_prefix", queryPrefix(query),
			"err", err,
		)
	}
}

func queryPrefix(query string) string {
	const max = 120
	if len(query) <= max {
		return query
	}
	return query[:max]
}

func (q *CHQuery) Query(ctx context.Context, query string, args ...any) (driver.Rows, error) {
	if q == nil || q.conn == nil {
		return nil, fmt.Errorf("chquery: no connection")
	}
	if err := q.acquire(ctx); err != nil {
		if errors.Is(err, ErrCHQueryRejected) {
			metrics.CHQueryRejectedTotal.Inc()
		}
		return nil, err
	}
	defer q.release()

	qctx := ctx
	if q.queryTimeout > 0 {
		var cancel context.CancelFunc
		qctx, cancel = context.WithTimeout(ctx, q.queryTimeout)
		defer cancel()
	}

	start := time.Now()
	rows, err := q.conn.Query(q.withSettings(qctx), query, args...)
	q.observe(start, query, err)
	return rows, err
}

func (q *CHQuery) QueryRow(ctx context.Context, query string, args ...any) driver.Row {
	if q == nil || q.conn == nil {
		return &errRow{err: fmt.Errorf("chquery: no connection")}
	}
	if err := q.acquire(ctx); err != nil {
		if errors.Is(err, ErrCHQueryRejected) {
			metrics.CHQueryRejectedTotal.Inc()
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

func (q *CHQuery) Exec(ctx context.Context, query string, args ...any) error {
	if q == nil || q.conn == nil {
		return fmt.Errorf("chquery: no connection")
	}
	if err := q.acquire(ctx); err != nil {
		if errors.Is(err, ErrCHQueryRejected) {
			metrics.CHQueryRejectedTotal.Inc()
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

func (q *CHQuery) InFlight() int {
	if q == nil {
		return 0
	}
	return int(q.inFlight.Load())
}

func (q *CHQuery) IngestionLag(ctx context.Context) (time.Duration, error) {
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

func Freshness(lag time.Duration, staleThreshold time.Duration) (stale bool, lagSeconds int) {
	if staleThreshold <= 0 {
		staleThreshold = defaultCHStaleThreshold
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
