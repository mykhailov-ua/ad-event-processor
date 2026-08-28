package fraudadmin

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"strconv"
	"sync"
	"time"

	"ad-event-processor/internal/metrics"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/redis/go-redis/v9"
)

const (
	defaultFraudEvalReportPath = "var/fraudscore/shadow_eval_report.json"
	fraudEvalReportCacheTTL    = 60 * time.Second
	defaultFraudEvalStaleHours = 48
	mlEvalReportID             = "shadow_eval"
)

type FraudMLSnapshot struct {
	VersionID        string
	ArtifactHash     string
	Precision        float64
	Recall           float64
	DriftDetected    bool
	DriftSummary     string
	EvalGeneratedAt  string
	EvalStatus       string
	EvalStale        bool
	LabelMethod      string
	ShardsConsistent *bool
}

type fraudEvalReport struct {
	Status        string
	GeneratedAt   string
	GeneratedTime time.Time
	LabelMethod   string
	Precision     float64
	Recall        float64
	DriftDetected bool
	DriftSummary  string
	Available     bool
}

var fraudEvalReportCache struct {
	mu       sync.Mutex
	loadedAt time.Time
	path     string
	modTime  time.Time
	report   fraudEvalReport
}

func fraudEvalReportPath() string {
	path := os.Getenv("FRAUD_EVAL_REPORT_PATH")
	if path == "" {
		return defaultFraudEvalReportPath
	}
	return path
}

func fraudEvalStaleHours() time.Duration {
	raw := os.Getenv("FRAUD_EVAL_STALE_HOURS")
	if raw == "" {
		return defaultFraudEvalStaleHours * time.Hour
	}
	hours, err := strconv.Atoi(raw)
	if err != nil || hours <= 0 {
		return defaultFraudEvalStaleHours * time.Hour
	}
	return time.Duration(hours) * time.Hour
}

func parseFraudEvalReportJSON(data []byte) fraudEvalReport {
	var raw struct {
		Status        string  `json:"status"`
		GeneratedAt   string  `json:"generated_at"`
		LabelMethod   string  `json:"label_method"`
		Precision     float64 `json:"precision"`
		Recall        float64 `json:"recall"`
		DriftDetected bool    `json:"drift_detected"`
		Drift         *struct {
			DriftDetected bool    `json:"drift_detected"`
			MaxDrift      float64 `json:"max_drift"`
			Status        string  `json:"status"`
		} `json:"drift"`
	}
	if err := json.Unmarshal(data, &raw); err != nil {
		return fraudEvalReport{Status: "eval_unavailable"}
	}

	report := fraudEvalReport{
		Status:      raw.Status,
		GeneratedAt: raw.GeneratedAt,
		LabelMethod: raw.LabelMethod,
		Precision:   raw.Precision,
		Recall:      raw.Recall,
		Available:   true,
	}
	if raw.GeneratedAt != "" {
		if parsed, err := time.Parse(time.RFC3339, raw.GeneratedAt); err == nil {
			report.GeneratedTime = parsed.UTC()
		}
	}
	if raw.Drift != nil {
		report.DriftDetected = raw.Drift.DriftDetected
		if raw.Drift.DriftDetected {
			report.DriftSummary = "Traffic mix changed more than 30% vs training on one or more raw features."
		} else if raw.Drift.MaxDrift > 0 {
			report.DriftSummary = "Input drift within training band."
		}
	} else {
		report.DriftDetected = raw.DriftDetected
		if raw.DriftDetected {
			report.DriftSummary = "Traffic mix changed more than 30% vs training on one or more raw features."
		}
	}
	if report.LabelMethod == "" {
		report.LabelMethod = "proxy"
	}
	return report
}

func parseFraudEvalReportFromPGRow(
	status string,
	generatedAt time.Time,
	precision, recall float64,
	driftJSON []byte,
	labelMethod string,
) fraudEvalReport {
	report := fraudEvalReport{
		Status:        status,
		GeneratedAt:   generatedAt.UTC().Format(time.RFC3339),
		GeneratedTime: generatedAt.UTC(),
		LabelMethod:   labelMethod,
		Precision:     precision,
		Recall:        recall,
		Available:     true,
	}
	if len(driftJSON) > 0 {
		var drift struct {
			DriftDetected bool    `json:"drift_detected"`
			MaxDrift      float64 `json:"max_drift"`
			Status        string  `json:"status"`
		}
		if err := json.Unmarshal(driftJSON, &drift); err == nil {
			report.DriftDetected = drift.DriftDetected
			if drift.DriftDetected {
				report.DriftSummary = "Traffic mix changed more than 30% vs training on one or more raw features."
			} else if drift.MaxDrift > 0 {
				report.DriftSummary = "Input drift within training band."
			}
		}
	}
	if report.LabelMethod == "" {
		report.LabelMethod = "proxy"
	}
	return report
}

func loadFraudEvalReportFromPG(ctx context.Context, pool *pgxpool.Pool) (fraudEvalReport, bool) {
	if pool == nil {
		return fraudEvalReport{}, false
	}
	var status, labelMethod string
	var precision, recall float64
	var generatedAt time.Time
	var driftJSON []byte
	err := pool.QueryRow(ctx, `
		SELECT generated_at, precision, recall, drift_json, status, label_method
		FROM ml_eval_reports
		WHERE id = $1`,
		mlEvalReportID,
	).Scan(&generatedAt, &precision, &recall, &driftJSON, &status, &labelMethod)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return fraudEvalReport{}, false
		}
		return fraudEvalReport{}, false
	}
	return parseFraudEvalReportFromPGRow(status, generatedAt, precision, recall, driftJSON, labelMethod), true
}

func loadFraudEvalReport(ctx context.Context, pool *pgxpool.Pool, now time.Time) fraudEvalReport {
	if report, ok := loadFraudEvalReportFromPG(ctx, pool); ok {
		return report
	}
	return loadFraudEvalReportFromFile(now)
}

func loadFraudEvalReportFromFile(now time.Time) fraudEvalReport {
	path := fraudEvalReportPath()
	info, err := os.Stat(path)
	if err != nil {
		return fraudEvalReport{Status: "eval_unavailable"}
	}

	fraudEvalReportCache.mu.Lock()
	defer fraudEvalReportCache.mu.Unlock()

	if fraudEvalReportCache.path == path &&
		fraudEvalReportCache.modTime.Equal(info.ModTime()) &&
		now.Sub(fraudEvalReportCache.loadedAt) < fraudEvalReportCacheTTL {
		return fraudEvalReportCache.report
	}

	data, err := os.ReadFile(path)
	if err != nil {
		report := fraudEvalReport{Status: "eval_unavailable"}
		fraudEvalReportCache.path = path
		fraudEvalReportCache.modTime = info.ModTime()
		fraudEvalReportCache.loadedAt = now
		fraudEvalReportCache.report = report
		return report
	}

	report := parseFraudEvalReportJSON(data)

	fraudEvalReportCache.path = path
	fraudEvalReportCache.modTime = info.ModTime()
	fraudEvalReportCache.loadedAt = now
	fraudEvalReportCache.report = report
	return report
}

func computeMLEvalStatus(report fraudEvalReport, stale bool) string {
	if !report.Available || report.Status == "skipped" || report.Status == "empty" || report.Status == "error" || report.Status == "eval_unavailable" {
		return "eval_unavailable"
	}
	if stale {
		return "eval_stale"
	}
	if report.DriftDetected {
		return "drift_detected"
	}
	return "healthy"
}

func readMLShardsConsistent(ctx context.Context, shards []redis.UniversalClient) *bool {
	if len(shards) == 0 {
		return nil
	}

	type shardRedis struct {
		version string
		hash    string
	}

	var shardRows []shardRedis
	for _, redisClient := range shards {
		if redisClient == nil {
			continue
		}
		pipe := redisClient.Pipeline()
		verCmd := pipe.Get(ctx, "ml:model:version")
		hashCmd := pipe.Get(ctx, "ml:model:hash")
		if _, err := pipe.Exec(ctx); err != nil && !errors.Is(err, redis.Nil) {
			return nil
		}
		version, err := verCmd.Result()
		if err != nil && !errors.Is(err, redis.Nil) {
			return nil
		}
		hash, err := hashCmd.Result()
		if err != nil && !errors.Is(err, redis.Nil) {
			return nil
		}
		shardRows = append(shardRows, shardRedis{version: version, hash: hash})
	}
	if len(shardRows) == 0 {
		return nil
	}

	ref := shardRows[0]
	consistent := true
	for _, shard := range shardRows[1:] {
		if shard.version != ref.version || shard.hash != ref.hash {
			consistent = false
			break
		}
	}
	return &consistent
}

func BuildFraudMLSnapshot(ctx context.Context, host MLSnapshotHost) (FraudMLSnapshot, error) {
	var out FraudMLSnapshot
	pool := host.SnapshotPool()
	if pool != nil {
		err := pool.QueryRow(ctx, `
			SELECT id, artifact_hash FROM ml_model_versions
			WHERE status = 'ACTIVE' ORDER BY created_at DESC LIMIT 1`,
		).Scan(&out.VersionID, &out.ArtifactHash)
		if err != nil && !errors.Is(err, pgx.ErrNoRows) {
			return out, fmt.Errorf("load active ml model version: %w", err)
		}
	}

	now := time.Now().UTC()
	report := loadFraudEvalReport(ctx, pool, now)
	stale := false
	if report.Available && !report.GeneratedTime.IsZero() {
		stale = now.Sub(report.GeneratedTime) > fraudEvalStaleHours()
	} else if report.Available && report.GeneratedAt == "" {
		stale = true
	}

	out.Precision = report.Precision
	out.Recall = report.Recall
	out.DriftDetected = report.DriftDetected
	out.DriftSummary = report.DriftSummary
	out.EvalGeneratedAt = report.GeneratedAt
	out.LabelMethod = report.LabelMethod
	out.EvalStale = stale
	out.EvalStatus = computeMLEvalStatus(report, stale)
	out.ShardsConsistent = readMLShardsConsistent(ctx, host.SnapshotRedisShards())
	return out, nil
}

func RefreshFraudMLEvalMetrics(ctx context.Context, pool *pgxpool.Pool, now time.Time) {
	publishFraudMLEvalMetrics(loadFraudEvalReport(ctx, pool, now))
}

func publishFraudMLEvalMetrics(report fraudEvalReport) {
	if !report.Available {
		metrics.FraudMLShadowPrecision.Set(0)
		metrics.FraudMLShadowRecall.Set(0)
		metrics.FraudMLDriftDetected.Set(0)
		metrics.FraudMLEvalGeneratedAtTimestamp.Set(0)
		return
	}
	metrics.FraudMLShadowPrecision.Set(report.Precision)
	metrics.FraudMLShadowRecall.Set(report.Recall)
	if report.DriftDetected {
		metrics.FraudMLDriftDetected.Set(1)
	} else {
		metrics.FraudMLDriftDetected.Set(0)
	}
	if !report.GeneratedTime.IsZero() {
		metrics.FraudMLEvalGeneratedAtTimestamp.Set(float64(report.GeneratedTime.Unix()))
	} else {
		metrics.FraudMLEvalGeneratedAtTimestamp.Set(0)
	}
}
