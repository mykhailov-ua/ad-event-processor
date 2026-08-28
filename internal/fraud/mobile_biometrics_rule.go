package fraud

import (
	"context"
	"encoding/hex"
	"fmt"
	"time"

	"ad-event-processor/internal/database"
)

const mobileBiometricsQuery = `
SELECT
 ip_hash,
 count() AS samples,
 countIf(mobile_gyro_flat = 1 AND mobile_gyro_samples >= ?) AS flat_hits,
 countIf(mobile_gyro_samples = 0 AND mobile_touch_count = 0) AS motionless_hits
FROM conversions
WHERE created_at >= now() - toIntervalSecond(?)
 AND mobile_biometric_set = 1
 AND (device_type IN ('mobile', 'tablet') OR mobile_biometric_mobile = 1)
 AND ` + emptyIPHashFilter + `
GROUP BY ip_hash
HAVING samples >= ?
 AND (flat_hits >= ? OR motionless_hits >= ?)`

type mobileBiometricsRule struct {
	clickhouseQuery *database.ClickHouseQuery
	cfg             AnalyzerConfig
}

func (r *mobileBiometricsRule) Name() string { return "mobile_biometrics" }

func (r *mobileBiometricsRule) Find(ctx context.Context) ([]SuspiciousIP, error) {
	if r == nil || r.clickhouseQuery == nil || !r.cfg.MobileBiometricsEnabled {
		return nil, nil
	}
	minSamples := r.cfg.MobileBiometricsMinSamples
	if minSamples == 0 {
		minSamples = defaultMobileBiometricsMinSamples
	}
	minFlatHits := r.cfg.MobileBiometricsMinFlatHits
	if minFlatHits == 0 {
		minFlatHits = defaultMobileBiometricsMinFlatHits
	}
	minMotionless := r.cfg.MobileBiometricsMinMotionless
	if minMotionless == 0 {
		minMotionless = defaultMobileBiometricsMinMotionless
	}
	minGyroSamples := r.cfg.MobileBiometricsMinGyroSamples
	if minGyroSamples == 0 {
		minGyroSamples = defaultMobileBiometricsMinGyroSamples
	}

	windowSec := database.ClampCHWindowSeconds(int64(r.cfg.Window / time.Second))
	rows, err := r.clickhouseQuery.Query(
		ctx,
		mobileBiometricsQuery,
		minGyroSamples,
		windowSec,
		minSamples,
		minFlatHits,
		minMotionless,
	)
	if err != nil {
		return nil, fmt.Errorf("mobile biometrics query: %w", err)
	}
	defer func() { _ = rows.Close() }()

	var out []SuspiciousIP
	for rows.Next() {
		var ipHash []byte
		var samples, flatHits, motionlessHits uint64
		if err := rows.Scan(&ipHash, &samples, &flatHits, &motionlessHits); err != nil {
			return nil, fmt.Errorf("scan mobile biometrics row: %w", err)
		}
		if samples < minSamples || len(ipHash) == 0 {
			continue
		}
		reason := "ivt_mobile_biometrics_flat_gyro"
		if motionlessHits >= minMotionless && flatHits < minFlatHits {
			reason = "ivt_mobile_biometrics_motionless"
		}
		out = append(out, SuspiciousIP{
			IP:         hex.EncodeToString(ipHash),
			Reason:     reason,
			Score:      60,
			Action:     "silent_reject",
			TTLSeconds: 3600,
		})
	}
	return out, rows.Err()
}

const (
	defaultMobileBiometricsMinSamples     = 5
	defaultMobileBiometricsMinFlatHits    = 4
	defaultMobileBiometricsMinMotionless  = 5
	defaultMobileBiometricsMinGyroSamples = 3
)
