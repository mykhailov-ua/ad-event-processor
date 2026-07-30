package management

import (
	"context"
	"fmt"
	"log/slog"
	"math"
	"os"
	"sort"
	"sync"
	"time"

	db "espx/internal/ingestion/sqlc"

	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
)

const (
	defaultNodeMetricsInterval = 10 * time.Second
	defaultNodeMetricsTTL      = 24 * time.Hour
)

type NodeMetricsWorker struct {
	svc      *Service
	pool     *pgxpool.Pool
	interval time.Duration
	ttl      time.Duration
	nodeID   string
	role     string
	region   int16
	acc      metricAccumulator
}

type metricAccumulator struct {
	mu      sync.Mutex
	samples map[string][]float64
}

func newMetricAccumulator() metricAccumulator {
	return metricAccumulator{samples: make(map[string][]float64)}
}

func (a *metricAccumulator) Record(metric string, value float64) {
	if metric == "" || math.IsNaN(value) || math.IsInf(value, 0) {
		return
	}
	a.mu.Lock()
	a.samples[metric] = append(a.samples[metric], value)
	a.mu.Unlock()
}

func (a *metricAccumulator) Drain() map[string][]float64 {
	a.mu.Lock()
	defer a.mu.Unlock()
	if len(a.samples) == 0 {
		return nil
	}
	out := a.samples
	a.samples = make(map[string][]float64)
	return out
}

func NewNodeMetricsWorker(svc *Service) *NodeMetricsWorker {
	nodeID, _ := os.Hostname()
	if svc != nil && svc.cfg != nil && svc.cfg.NodeID != "" {
		nodeID = svc.cfg.NodeID
	}
	role := "management"
	if svc != nil && svc.cfg != nil && svc.cfg.NodeRole != "" {
		role = svc.cfg.NodeRole
	}
	region := int16(0)
	if svc != nil && svc.cfg != nil {
		region = int16(svc.cfg.RegionCode)
	}
	return &NodeMetricsWorker{
		svc:      svc,
		pool:     svc.GetPool(),
		interval: defaultNodeMetricsInterval,
		ttl:      defaultNodeMetricsTTL,
		nodeID:   nodeID,
		role:     role,
		region:   region,
		acc:      newMetricAccumulator(),
	}
}

func (w *NodeMetricsWorker) Record(metric string, value float64) {
	if w == nil {
		return
	}
	w.acc.Record(metric, value)
}

func (w *NodeMetricsWorker) Start(ctx context.Context) {
	if w == nil || w.pool == nil {
		return
	}
	slog.Info("node metrics worker starting",
		"node_id", w.nodeID,
		"role", w.role,
		"region", w.region,
		"interval", w.interval,
	)
	ticker := time.NewTicker(w.interval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			if err := w.Flush(ctx, time.Now().UTC()); err != nil {
				slog.Error("node metrics flush failed", "node_id", w.nodeID, "error", err)
			}
		}
	}
}

func (w *NodeMetricsWorker) Flush(ctx context.Context, now time.Time) error {
	if w == nil || w.pool == nil {
		return nil
	}
	run := func(runCtx context.Context) error {
		if err := w.flushBuckets(runCtx, now); err != nil {
			return err
		}
		return w.expireBuckets(runCtx, now)
	}
	if w.svc != nil {
		return w.svc.withPgLow(ctx, run)
	}
	return run(ctx)
}

func (w *NodeMetricsWorker) flushBuckets(ctx context.Context, now time.Time) error {
	samples := w.acc.Drain()
	if len(samples) == 0 {
		return nil
	}
	bucketTS := now.Truncate(w.interval)
	q := db.New(w.pool)
	for metric, values := range samples {
		p50, p99, mean, count := aggregateSamples(values)
		if count == 0 {
			continue
		}
		if err := q.InsertNodeMetricBucket(ctx, db.InsertNodeMetricBucketParams{
			NodeID:      w.nodeID,
			RegionCode:  w.region,
			Role:        w.role,
			BucketTs:    pgtype.Timestamptz{Time: bucketTS, Valid: true},
			Metric:      metric,
			ValueP50:    pgtype.Float8{Float64: p50, Valid: true},
			ValueP99:    pgtype.Float8{Float64: p99, Valid: true},
			ValueMean:   pgtype.Float8{Float64: mean, Valid: true},
			SampleCount: count,
		}); err != nil {
			return fmt.Errorf("flush node metric bucket node=%s: %w", w.nodeID, err)
		}
	}
	return nil
}

func (w *NodeMetricsWorker) expireBuckets(ctx context.Context, now time.Time) error {
	cutoff := now.Add(-w.ttl)
	q := db.New(w.pool)
	if _, err := q.DeleteExpiredNodeMetricBuckets(ctx, pgtype.Timestamptz{Time: cutoff, Valid: true}); err != nil {
		return fmt.Errorf("expire node metric buckets node=%s: %w", w.nodeID, err)
	}
	return nil
}

func aggregateSamples(values []float64) (p50, p99, mean float64, count int64) {
	if len(values) == 0 {
		return 0, 0, 0, 0
	}
	sorted := append([]float64(nil), values...)
	sort.Float64s(sorted)
	count = int64(len(sorted))
	var sum float64
	for _, v := range sorted {
		sum += v
	}
	mean = sum / float64(count)
	p50 = percentile(sorted, 0.50)
	p99 = percentile(sorted, 0.99)
	return p50, p99, mean, count
}

func percentile(sorted []float64, q float64) float64 {
	if len(sorted) == 0 {
		return 0
	}
	if q <= 0 {
		return sorted[0]
	}
	if q >= 1 {
		return sorted[len(sorted)-1]
	}
	pos := q * float64(len(sorted)-1)
	lo := int(math.Floor(pos))
	hi := int(math.Ceil(pos))
	if lo == hi {
		return sorted[lo]
	}
	frac := pos - float64(lo)
	return sorted[lo]*(1-frac) + sorted[hi]*frac
}
