package fraud

import (
	"context"
	"encoding/hex"
	"fmt"
	"time"

	"ad-event-processor/internal/database"
)

const (
	defaultIntervalMinIntervals = 30
	defaultIntervalMaxVariance  = 0.005
	intervalBotReason           = "ivt_interval_bot"
)

type intervalBotnetRule struct {
	q   *database.CHQuery
	cfg AnalyzerConfig
}

func (r *intervalBotnetRule) Name() string { return "interval_bot" }

func (r *intervalBotnetRule) Find(ctx context.Context) ([]SuspiciousIP, error) {
	if r.q == nil {
		return nil, fmt.Errorf("interval bot rule: nil chquery")
	}

	windowSec := int64(r.cfg.Window / time.Second)
	if windowSec <= 0 {
		windowSec = 3600
	}
	minIntervals := r.cfg.IntervalMinIntervals
	if minIntervals == 0 {
		minIntervals = defaultIntervalMinIntervals
	}
	maxVariance := r.cfg.IntervalMaxVariance
	if maxVariance <= 0 {
		maxVariance = defaultIntervalMaxVariance
	}

	query := `
SELECT
 sample_ip_hash,
 variance,
 n_intervals
FROM (
 SELECT
 ip_hash,
 any(ip_hash) AS sample_ip_hash,
 varPop(delta_t) AS variance,
 count() AS n_intervals
 FROM (
 SELECT
 ip_hash,
 dateDiff(
 'millisecond',
 lagInFrame(created_at, 1, created_at) OVER (PARTITION BY ip_hash ORDER BY created_at),
 created_at
 ) / 1000.0 AS delta_t
 FROM clicks
 WHERE created_at >= now() - toIntervalSecond(?)
 AND ` + emptyIPHashFilter + `
 )
 WHERE delta_t > 0
 GROUP BY ip_hash
 HAVING n_intervals >= ? AND variance < ?
)
WHERE length(sample_ip_hash) > 0`

	rows, err := r.q.Query(ctx, query, windowSec, minIntervals, maxVariance)
	if err != nil {
		return nil, fmt.Errorf("interval bot query: %w", err)
	}
	defer func() { _ = rows.Close() }()

	var out []SuspiciousIP
	for rows.Next() {
		var ipHash []byte
		var variance float64
		var nIntervals uint64
		if err := rows.Scan(&ipHash, &variance, &nIntervals); err != nil {
			return nil, fmt.Errorf("scan interval bot row: %w", err)
		}
		if len(ipHash) == 0 {
			continue
		}
		out = append(out, SuspiciousIP{
			IP:     hex.EncodeToString(ipHash),
			Reason: intervalBotReason,
			Score:  variance,
		})
	}
	return out, rows.Err()
}

func popVariance(samples []float64) float64 {
	if len(samples) == 0 {
		return 0
	}
	var sum float64
	for _, v := range samples {
		sum += v
	}
	mean := sum / float64(len(samples))
	var acc float64
	for _, v := range samples {
		d := v - mean
		acc += d * d
	}
	return acc / float64(len(samples))
}

func isIntervalBot(deltas []float64, minIntervals uint64, maxVariance float64) bool {
	if uint64(len(deltas)) < minIntervals {
		return false
	}
	return popVariance(deltas) < maxVariance
}
