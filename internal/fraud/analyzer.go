package fraud

import (
	"context"
	"encoding/hex"
	"fmt"
	"time"

	"espx/internal/database"
)

type SuspiciousIP struct {
	IP         string
	Reason     string
	Score      float64
	CampaignID string
	Action     string
	Boost      int32
	TTLSeconds int64
}

type AnalyzerConfig struct {
	Window               time.Duration
	MinClicks            uint64
	MinImpressions       uint64
	ClickToImpRatio      float64
	MinIPsPerUA          uint64
	MinEventsPerIP       uint64
	IntervalMinIntervals uint64
	IntervalMaxVariance  float64
}

func DefaultAnalyzerConfig() AnalyzerConfig {
	return AnalyzerConfig{
		Window:               time.Hour,
		MinClicks:            10,
		MinImpressions:       1,
		ClickToImpRatio:      5.0,
		MinIPsPerUA:          8,
		MinEventsPerIP:       5,
		IntervalMinIntervals: defaultIntervalMinIntervals,
		IntervalMaxVariance:  defaultIntervalMaxVariance,
	}
}

type Analyzer struct {
	q   *database.CHQuery
	cfg AnalyzerConfig
}

func NewAnalyzer(q *database.CHQuery, cfg AnalyzerConfig) *Analyzer {
	return &Analyzer{q: q, cfg: cfg}
}

func (analyzer *Analyzer) FindSuspiciousIPs(ctx context.Context) ([]SuspiciousIP, error) {
	reg := NewAnalyzerRegistry(analyzer.q, nil, nil, analyzer.cfg, nil, nil, 0, nil)
	return reg.FindSuspiciousIPs(ctx)
}

func (analyzer *Analyzer) findHighClickToImpRatio(ctx context.Context, windowSec int64) ([]SuspiciousIP, error) {
	query := `
SELECT
    c.ip_hash,
    c.click_count,
    coalesce(i.imp_count, toUInt64(0)) AS imp_count
FROM (
    SELECT ip_hash, count() AS click_count
    FROM clicks
    WHERE created_at >= now() - toIntervalSecond(?)
      AND ` + emptyIPHashFilter + `
    GROUP BY ip_hash
    HAVING click_count >= ?
) AS c
LEFT JOIN (
    SELECT ip_hash, count() AS imp_count
    FROM impressions
    WHERE created_at >= now() - toIntervalSecond(?)
      AND ` + emptyIPHashFilter + `
    GROUP BY ip_hash
) AS i ON c.ip_hash = i.ip_hash
WHERE c.click_count >= ?
  AND (
    imp_count < ?
    OR (toFloat64(c.click_count) / greatest(toFloat64(imp_count), 1.0)) >= ?
  )`

	rows, err := analyzer.q.Query(
		ctx,
		query,
		windowSec,
		analyzer.cfg.MinClicks,
		windowSec,
		analyzer.cfg.MinClicks,
		analyzer.cfg.MinImpressions,
		analyzer.cfg.ClickToImpRatio,
	)
	if err != nil {
		return nil, fmt.Errorf("high click-to-imp query: %w", err)
	}
	defer rows.Close()

	var out []SuspiciousIP
	for rows.Next() {
		var ipHash []byte
		var clickCount, impCount uint64
		if err := rows.Scan(&ipHash, &clickCount, &impCount); err != nil {
			return nil, fmt.Errorf("scan high click-to-imp row: %w", err)
		}
		if len(ipHash) == 0 {
			continue
		}
		ipKey := hex.EncodeToString(ipHash)
		ratio := float64(clickCount)
		if impCount > 0 {
			ratio /= float64(impCount)
		}
		out = append(out, SuspiciousIP{
			IP:     ipKey,
			Reason: "ivt_high_click_to_imp_ratio",
			Score:  ratio,
		})
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate high click-to-imp rows: %w", err)
	}
	return out, nil
}

func (analyzer *Analyzer) findSharedFingerprintClusters(ctx context.Context, windowSec int64) ([]SuspiciousIP, error) {
	query := `
SELECT ip_hash
FROM (
    SELECT
        ua_hash,
        groupUniqArray(ip_hash) AS ips,
        uniqCombined(ip_hash) AS ip_count
    FROM (
        SELECT ip_hash, ua_hash
        FROM impressions
        WHERE created_at >= now() - toIntervalSecond(?)
          AND ` + emptyIPHashFilter + `
          AND ua_hash != ''
        UNION ALL
        SELECT ip_hash, ua_hash
        FROM clicks
        WHERE created_at >= now() - toIntervalSecond(?)
          AND ` + emptyIPHashFilter + `
          AND ua_hash != ''
    )
    GROUP BY ua_hash
    HAVING ip_count >= ?
)
ARRAY JOIN ips AS ip_hash
GROUP BY ip_hash
HAVING count() >= 1`

	rows, err := analyzer.q.Query(
		ctx,
		query,
		windowSec,
		windowSec,
		analyzer.cfg.MinIPsPerUA,
	)
	if err != nil {
		return nil, fmt.Errorf("shared fingerprint query: %w", err)
	}
	defer rows.Close()

	var out []SuspiciousIP
	for rows.Next() {
		var ipHash []byte
		if err := rows.Scan(&ipHash); err != nil {
			return nil, fmt.Errorf("scan shared fingerprint row: %w", err)
		}
		if len(ipHash) == 0 {
			continue
		}
		out = append(out, SuspiciousIP{
			IP:     hex.EncodeToString(ipHash),
			Reason: "ivt_shared_fingerprint_cluster",
			Score:  float64(analyzer.cfg.MinIPsPerUA),
		})
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate shared fingerprint rows: %w", err)
	}
	return out, nil
}
