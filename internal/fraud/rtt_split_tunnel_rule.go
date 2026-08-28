package fraud

import (
	"context"
	"encoding/hex"
	"fmt"
	"time"

	"ad-event-processor/internal/database"
)

const rttSplitTunnelQuery = `
SELECT
 ip_hash,
 count() AS samples,
 avg(rtt_split_delta_ms) AS avg_delta_ms,
 varPop(rtt_split_delta_ms) AS delta_var
FROM clicks
WHERE created_at >= now() - toIntervalSecond(?)
 AND rtt_split_delta_ms > 0
 AND ` + emptyIPHashFilter + `
GROUP BY ip_hash
HAVING samples >= ? AND avg_delta_ms >= ? AND delta_var <= ?`

type rttSplitTunnelRule struct {
	clickhouseQuery *database.ClickHouseQuery
	cfg             AnalyzerConfig
}

func (r *rttSplitTunnelRule) Name() string { return "rtt_split_tunnel" }

func (r *rttSplitTunnelRule) Find(ctx context.Context) ([]SuspiciousIP, error) {
	if r == nil || r.clickhouseQuery == nil || !r.cfg.RTTSplitTunnelEnabled {
		return nil, nil
	}
	if r.cfg.RTTSplitMinDeltaMS == 0 || r.cfg.RTTSplitMinSamples == 0 {
		return nil, nil
	}

	windowSec := database.ClampCHWindowSeconds(int64(r.cfg.Window / time.Second))
	maxVariance := r.cfg.RTTSplitMaxVariance
	if maxVariance <= 0 {
		maxVariance = defaultRTTSplitMaxVariance
	}

	rows, err := r.clickhouseQuery.Query(
		ctx,
		rttSplitTunnelQuery,
		windowSec,
		r.cfg.RTTSplitMinSamples,
		r.cfg.RTTSplitMinDeltaMS,
		maxVariance,
	)
	if err != nil {
		return nil, fmt.Errorf("rtt split tunnel query: %w", err)
	}
	defer func() { _ = rows.Close() }()

	var out []SuspiciousIP
	for rows.Next() {
		var ipHash []byte
		var samples uint64
		var avgDelta, deltaVar float64
		if err := rows.Scan(&ipHash, &samples, &avgDelta, &deltaVar); err != nil {
			return nil, fmt.Errorf("scan rtt split tunnel row: %w", err)
		}
		if samples < r.cfg.RTTSplitMinSamples {
			continue
		}
		if len(ipHash) == 0 {
			continue
		}
		out = append(out, SuspiciousIP{
			IP:         hex.EncodeToString(ipHash),
			Reason:     "ivt_rtt_split_tunnel",
			Score:      65,
			Action:     "silent_reject",
			TTLSeconds: 3600,
		})
	}
	return out, rows.Err()
}

const defaultRTTSplitMaxVariance = 2500
