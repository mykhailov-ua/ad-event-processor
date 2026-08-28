package opsadmin

import (
	"bytes"
	"context"
	"encoding/csv"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"sort"
	"strconv"
	"time"

	"ad-event-processor/internal/domain"
	db "ad-event-processor/internal/domain/db"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/redis/go-redis/v9"
)

const (
	defaultOpsMetricScrapeInterval = 15 * time.Second
	defaultOpsMetricRetention      = 24 * time.Hour
	defaultOpsMetricScrapeTimeout  = 5 * time.Second
)

const insertOpsMetricSampleSQL = `INSERT INTO ops.metric_samples (name, labels_hash, ts, value)
		VALUES ($1, $2, $3, $4)
		ON CONFLICT (name, labels_hash, ts) DO UPDATE SET value = EXCLUDED.value`

func (r *Reader) GetIncidentSnapshot(ctx context.Context) (IncidentSnapshotDTO, error) {
	report, err := r.deps.GetShardHealth(ctx)
	if err != nil {
		return IncidentSnapshotDTO{}, err
	}
	snap := IncidentSnapshotDTO{
		EmergencyBreaker: report.EmergencyBreaker,
		Shards:           report.Shards,
		Outbox:           report.Outbox,
		StreamLag:        []ShardStreamLag{},
		BreakerStates:    map[string]string{},
	}
	stale, _ := r.incidentDashboardStale(ctx)
	snap.StaleDashboard = stale
	if stale {
		campaigns, err := r.listAffectedCampaigns(ctx, 50)
		if err == nil && len(campaigns) > 0 {
			snap.AffectedCampaigns = campaigns
		}
	}
	return snap, nil
}

func (r *Reader) ListOutboxEvents(ctx context.Context, status, eventType, cursor string, limit int32) (OutboxListResult, error) {
	if r.pool() == nil {
		return OutboxListResult{}, fmt.Errorf("postgres pool not configured")
	}
	if limit <= 0 {
		limit = 50
	}
	var cursorID int64
	if cursor != "" {
		n, err := strconv.ParseInt(cursor, 10, 64)
		if err != nil {
			return OutboxListResult{}, errInvalidQuery("invalid cursor")
		}
		cursorID = n
	}
	rows, err := r.pool().Query(ctx, `
		SELECT id, event_type, status, created_at
		FROM outbox_events
		WHERE ($1::text = '' OR status = $1)
		 AND ($2::text = '' OR event_type = $2)
		 AND ($3::bigint = 0 OR id < $3)
		ORDER BY id DESC
		LIMIT $4`, status, eventType, cursorID, limit+1)
	if err != nil {
		return OutboxListResult{}, err
	}
	defer rows.Close()

	var items []OutboxEventDTO
	for rows.Next() {
		var id int64
		var eventTypeVal, statusVal string
		var createdAt time.Time
		if err := rows.Scan(&id, &eventTypeVal, &statusVal, &createdAt); err != nil {
			return OutboxListResult{}, err
		}
		items = append(items, OutboxEventDTO{
			ID:        id,
			EventType: eventTypeVal,
			Status:    statusVal,
			CreatedAt: createdAt.UTC().Format(time.RFC3339),
		})
	}
	result := OutboxListResult{Items: items, Total: int64(len(items))}
	if int32(len(items)) > limit {
		result.Items = items[:limit]
		result.NextCursor = strconv.FormatInt(result.Items[len(result.Items)-1].ID, 10)
	}
	return result, rows.Err()
}

func (r *Reader) ListDLQEntries(ctx context.Context, cursor string, limit int) (FanOutResult[DLQEntryDTO], error) {
	return r.listDLQEntries(ctx, cursor, limit)
}

func (r *Reader) EnqueueDLQRetry(ctx context.Context, payload DLQRetryPayload, idempotencyKey string) error {
	return r.enqueueDLQRetry(ctx, payload, idempotencyKey)
}

func (r *Reader) GetShardHealthFanOut(ctx context.Context) (ShardHealthAPIResponse, error) {
	report, err := r.deps.GetShardHealth(ctx)
	if err != nil {
		return ShardHealthAPIResponse{}, err
	}
	return ShardHealthAPIResponse{
		ShardHealthReport: report,
	}, nil
}

func (r *Reader) ExportAuditCSV(ctx context.Context, cursor string, redactPII bool, w io.Writer) (AuditExportResult, error) {
	var cursorID int64
	if cursor != "" {
		n, err := strconv.ParseInt(cursor, 10, 64)
		if err != nil {
			return AuditExportResult{}, errInvalidQuery("invalid cursor")
		}
		cursorID = n
	}
	cw := csv.NewWriter(w)
	if cursorID == 0 {
		_ = cw.Write([]string{"id", "admin_id", "action", "target_type", "target_id", "is_masked", "created_at"})
	}
	rows, err := db.New(r.pool()).ListAuditLogsExport(ctx, db.ListAuditLogsExportParams{
		Column1: cursorID,
		Limit:   500,
	})
	if err != nil {
		return AuditExportResult{}, err
	}
	var lastID int64
	for _, row := range rows {
		adminID := ""
		if row.AdminID.Valid {
			adminID = uuid.UUID(row.AdminID.Bytes).String()
		}
		targetID := ""
		if row.TargetID.Valid {
			targetID = uuid.UUID(row.TargetID.Bytes).String()
		}
		createdAt := ""
		if row.CreatedAt.Valid {
			createdAt = row.CreatedAt.Time.UTC().Format(time.RFC3339)
		}
		if redactPII {
			if adminID != "" {
				adminID = "***"
			}
			if targetID != "" {
				targetID = "***"
			}
		}
		_ = cw.Write([]string{
			strconv.FormatInt(row.ID, 10),
			adminID,
			row.Action,
			row.TargetType,
			targetID,
			strconv.FormatBool(row.IsMasked),
			createdAt,
		})
		lastID = row.ID
	}
	cw.Flush()
	if err := cw.Error(); err != nil {
		return AuditExportResult{}, err
	}
	byteCount := 0
	if buf, ok := w.(*bytes.Buffer); ok {
		byteCount = buf.Len()
	}
	result := AuditExportResult{Bytes: byteCount}
	if len(rows) >= 500 {
		result.Truncated = true
		result.NextCursor = strconv.FormatInt(lastID, 10)
	}
	return result, nil
}

func (r *Reader) LookupLedgerIDForPaymentIntent(ctx context.Context, intentID string) (string, error) {
	paymentIntentID, err := uuid.Parse(intentID)
	if err != nil {
		return "", err
	}
	row, err := db.New(r.pool()).GetLedgerByPaymentIntent(ctx, domain.ToUUID(paymentIntentID))
	if err != nil {
		return "", err
	}
	return strconv.FormatInt(row.ID, 10), nil
}

func (r *Reader) ListReconRuns(ctx context.Context, service string, limit, offset int32) ([]ReconRunDTO, int64, error) {
	return r.deps.ListReconRuns(ctx, service, limit, offset)
}

const dashboardMetricsBucketSec = 300

const mlManualLabelsListLimit = 100

func (r *Reader) GetDashboardSummary(ctx context.Context) (DashboardSummaryDTO, error) {
	if r == nil || r.pool() == nil {
		return DashboardSummaryDTO{}, fmt.Errorf("service not configured")
	}
	now := time.Now().UTC()
	snap, err := r.GetIncidentSnapshot(ctx)
	if err != nil {
		return DashboardSummaryDTO{}, err
	}
	services := buildDashboardTopology(ctx, r.deps, snap)
	driftMax, rps, err := r.readDashboardLiveSignals(ctx, now)
	if err != nil {
		return DashboardSummaryDTO{}, err
	}
	return DashboardSummaryDTO{
		GeneratedAt:      now.Format(time.RFC3339),
		Services:         services,
		DriftMicroMax:    driftMax,
		DriftAlert:       driftMax > 0,
		RPSEstimate:      rps,
		OutboxPending:    snap.Outbox.Pending,
		EmergencyBreaker: snap.EmergencyBreaker,
	}, nil
}

func (r *Reader) GetStackHealthSnapshot(ctx context.Context) (StackHealthSnapshot, error) {
	if r == nil || r.pool() == nil {
		return StackHealthSnapshot{}, fmt.Errorf("service not configured")
	}
	return r.deps.BuildStackHealthSnapshot(ctx)
}

func (r *Reader) GetDashboardMetrics(ctx context.Context, rangeHours int, metricName string) (DashboardMetricsDTO, error) {
	if r == nil || r.pool() == nil || r.pool() == nil {
		return DashboardMetricsDTO{}, fmt.Errorf("postgres pool not configured")
	}
	if rangeHours <= 0 {
		rangeHours = 24
	}
	if rangeHours > 24 {
		rangeHours = 24
	}
	now := time.Now().UTC()
	since := now.Add(-time.Duration(rangeHours) * time.Hour)
	q := db.New(r.pool())
	rows, err := q.ListOpsMetricSamplesDownsampled(ctx, db.ListOpsMetricSamplesDownsampledParams{
		Ts:      pgtype.Timestamptz{Time: since, Valid: true},
		Ts_2:    pgtype.Timestamptz{Time: now, Valid: true},
		Column3: metricName,
		Column4: float64(dashboardMetricsBucketSec),
	})
	if err != nil {
		return DashboardMetricsDTO{}, err
	}
	points := make([]DashboardMetricPoint, 0, len(rows))
	for _, row := range rows {
		ts, ok := metricSampleTime(row.Ts)
		if !ok {
			continue
		}
		points = append(points, DashboardMetricPoint{
			Name:       row.Name,
			LabelsHash: row.LabelsHash,
			Timestamp:  ts.UTC().Format(time.RFC3339),
			Value:      row.Value,
		})
	}
	return DashboardMetricsDTO{
		Range:       fmt.Sprintf("%dh", rangeHours),
		BucketSec:   dashboardMetricsBucketSec,
		Points:      points,
		GeneratedAt: now.Format(time.RFC3339),
	}, nil
}

func buildDashboardTopology(ctx context.Context, deps ReaderDeps, snap IncidentSnapshotDTO) []DashboardServiceCard {
	cards := []DashboardServiceCard{
		{ID: "management", Name: "Management", Status: "ok"},
		{ID: "tracker", Name: "Tracker", Status: "unknown"},
		{ID: "processor", Name: "Processor", Status: "unknown"},
	}
	if deps.Pool != nil {
		status := "ok"
		detail := ""
		if err := deps.Pool.Ping(ctx); err != nil {
			status = "down"
			detail = err.Error()
		}
		cards = append(cards, DashboardServiceCard{ID: "postgres", Name: "Postgres", Status: status, Detail: detail})
	} else {
		cards = append(cards, DashboardServiceCard{ID: "postgres", Name: "Postgres", Status: "down"})
	}
	clickhouseStatus := "disabled"
	if deps.Config != nil && deps.Config.IsClickHouseEnabled() {
		clickhouseStatus = "ok"
		if deps.ClickHouseQuery == nil {
			clickhouseStatus = "down"
		}
	}
	cards = append(cards, DashboardServiceCard{ID: "clickhouse", Name: "ClickHouse", Status: clickhouseStatus})
	for _, shard := range snap.Shards {
		status := "ok"
		if !shard.PingOK {
			status = "down"
		}
		cards = append(cards, DashboardServiceCard{
			ID:     fmt.Sprintf("redis-%d", shard.ShardID),
			Name:   fmt.Sprintf("Redis %d", shard.ShardID),
			Status: status,
			Detail: shard.PingError,
		})
	}
	if snap.Outbox.Pending > 0 {
		for i := range cards {
			if cards[i].ID == "processor" {
				cards[i].Status = "degraded"
				cards[i].Detail = fmt.Sprintf("outbox_pending=%d", snap.Outbox.Pending)
			}
		}
	} else {
		for i := range cards {
			if cards[i].ID == "processor" {
				cards[i].Status = "ok"
			}
		}
	}
	return cards
}

func (r *Reader) readDashboardLiveSignals(ctx context.Context, now time.Time) (driftMax float64, rps float64, err error) {
	pool := r.pool()
	if pool == nil {
		return 0, 0, nil
	}
	q := db.New(pool)
	driftRow, derr := q.GetLatestOpsMetricSample(ctx, db.GetLatestOpsMetricSampleParams{
		Name:       "ad_recon_drift_micro_max",
		LabelsHash: "",
	})
	if derr == nil {
		driftMax = driftRow.Value
	}
	prevSince := now.Add(-2 * defaultOpsMetricScrapeInterval)
	rows, qerr := q.ListOpsMetricSamplesWindow(ctx, db.ListOpsMetricSamplesWindowParams{
		Ts:      pgtype.Timestamptz{Time: prevSince, Valid: true},
		Ts_2:    pgtype.Timestamptz{Time: now, Valid: true},
		Column3: "ad_http_requests_total",
	})
	if qerr != nil || len(rows) < 2 {
		return driftMax, 0, nil
	}
	first := rows[0]
	last := rows[len(rows)-1]
	if !first.Ts.Valid || !last.Ts.Valid {
		return driftMax, 0, nil
	}
	delta := last.Value - first.Value
	secs := last.Ts.Time.Sub(first.Ts.Time).Seconds()
	if secs > 0 && delta >= 0 {
		rps = delta / secs
	}
	return driftMax, rps, nil
}

func metricSampleTime(v any) (time.Time, bool) {
	switch t := v.(type) {
	case time.Time:
		return t, true
	case pgtype.Timestamptz:
		if t.Valid {
			return t.Time, true
		}
	}
	return time.Time{}, false
}

func (r *Reader) GetMLModelStatus(ctx context.Context) (MLModelStatusDTO, error) {
	if r == nil || r.pool() == nil || r.pool() == nil {
		return MLModelStatusDTO{}, fmt.Errorf("postgres pool not configured")
	}

	status := MLModelStatusDTO{
		ShardSync: []MLShardSyncDTO{},
	}

	active, err := r.loadMLModelVersion(ctx, "ACTIVE")
	if err != nil {
		return MLModelStatusDTO{}, err
	}
	if active != nil {
		status.ActiveVersion = active
		status.Importance = TopFeatureImportance(active.ArtifactMetadata, 5)
	}

	syncing, err := r.loadMLModelVersion(ctx, "SYNCING")
	if err != nil {
		return MLModelStatusDTO{}, err
	}
	if syncing != nil {
		status.SyncingVersion = syncing
		if len(status.Importance) == 0 {
			status.Importance = TopFeatureImportance(syncing.ArtifactMetadata, 5)
		}
	}

	shardSync, err := r.loadMLShardSyncState(ctx)
	if err != nil {
		return MLModelStatusDTO{}, err
	}
	status.ShardSync = shardSync

	redisStatus, err := r.readMLModelRedis(ctx)
	if err != nil {
		return MLModelStatusDTO{}, err
	}
	status.Redis = redisStatus

	if evalReport, err := r.readMLDriftReport(); err == nil {
		status.Drift = evalReport.Drift
		status.DriftDetected = evalReport.DriftDetected
		status.Precision = evalReport.Precision
		status.Recall = evalReport.Recall
	}

	return status, nil
}

type mlEvalReport struct {
	Precision     float64
	Recall        float64
	DriftDetected bool
	Drift         json.RawMessage
}

func (r *Reader) readMLDriftReport() (mlEvalReport, error) {
	path := os.Getenv("FRAUD_EVAL_REPORT_PATH")
	if path == "" {
		path = "var/fraudscore/shadow_eval_report.json"
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return mlEvalReport{}, err
	}
	var raw struct {
		Precision float64         `json:"precision"`
		Recall    float64         `json:"recall"`
		Drift     json.RawMessage `json:"drift"`
	}
	if err := json.Unmarshal(data, &raw); err != nil {
		return mlEvalReport{}, err
	}
	out := mlEvalReport{
		Precision: raw.Precision,
		Recall:    raw.Recall,
		Drift:     raw.Drift,
	}
	if len(raw.Drift) > 0 {
		var driftBlock struct {
			DriftDetected bool `json:"drift_detected"`
		}
		if json.Unmarshal(raw.Drift, &driftBlock) == nil {
			out.DriftDetected = driftBlock.DriftDetected
		}
	}
	return out, nil
}

func TopFeatureImportance(metadata []byte, limit int) []MLFeatureImportanceDTO {
	if len(metadata) == 0 || limit <= 0 {
		return nil
	}
	var meta struct {
		Importance map[string]float64 `json:"importance"`
	}
	if err := json.Unmarshal(metadata, &meta); err != nil || len(meta.Importance) == 0 {
		return nil
	}
	names := make([]string, 0, len(meta.Importance))
	for name := range meta.Importance {
		names = append(names, name)
	}
	sort.Slice(names, func(i, j int) bool {
		return meta.Importance[names[i]] > meta.Importance[names[j]]
	})
	if len(names) > limit {
		names = names[:limit]
	}
	out := make([]MLFeatureImportanceDTO, len(names))
	for i, name := range names {
		out[i] = MLFeatureImportanceDTO{
			Name:  name,
			Value: meta.Importance[name],
		}
	}
	return out
}

func (r *Reader) AddMLManualLabel(ctx context.Context, ipHash string, label int, reason string) error {
	if r == nil || r.pool() == nil || r.pool() == nil {
		return fmt.Errorf("postgres pool not configured")
	}
	if ipHash == "" {
		return errValidation("ip_hash required")
	}
	if len(ipHash) != 32 {
		return errValidation("ip_hash must be 32 hex characters")
	}
	for _, c := range ipHash {
		if (c >= '0' && c <= '9') || (c >= 'a' && c <= 'f') || (c >= 'A' && c <= 'F') {
			continue
		}
		return errValidation("ip_hash must be 32 hex characters")
	}
	if label != 0 && label != 1 {
		return errValidation("label must be 0 or 1")
	}
	_, err := r.pool().Exec(ctx, `
		INSERT INTO ml_manual_labels (ip_hash, label, reason, source, created_at)
		VALUES ($1, $2, $3, 'admin_ui', NOW())
		ON CONFLICT (ip_hash) DO UPDATE SET
			label = EXCLUDED.label,
			reason = EXCLUDED.reason,
			created_at = NOW()`,
		ipHash, label, reason)
	return err
}

func (r *Reader) ListMLManualLabels(ctx context.Context) ([]MLManualLabelDTO, error) {
	if r == nil || r.pool() == nil || r.pool() == nil {
		return nil, fmt.Errorf("postgres pool not configured")
	}
	rows, err := r.pool().Query(ctx, `
		SELECT ip_hash, label, reason, source, created_at
		FROM ml_manual_labels
		ORDER BY created_at DESC
		LIMIT $1`, mlManualLabelsListLimit)
	if err != nil {
		return nil, fmt.Errorf("query ml_manual_labels: %w", err)
	}
	defer rows.Close()

	var out []MLManualLabelDTO
	for rows.Next() {
		var row MLManualLabelDTO
		var createdAt time.Time
		if err := rows.Scan(&row.IPHash, &row.Label, &row.Reason, &row.Source, &createdAt); err != nil {
			return nil, err
		}
		row.CreatedAt = createdAt.UTC().Format(time.RFC3339)
		out = append(out, row)
	}
	return out, rows.Err()
}

func (r *Reader) loadMLModelVersion(ctx context.Context, modelStatus string) (*MLModelVersionDTO, error) {
	var id, artifactHash, status string
	var metricsJSON []byte
	var createdAt time.Time
	err := r.pool().QueryRow(ctx, `
		SELECT id, artifact_hash, status, metrics_json, created_at
		FROM ml_model_versions
		WHERE status = $1
		ORDER BY created_at DESC
		LIMIT 1`, modelStatus).Scan(&id, &artifactHash, &status, &metricsJSON, &createdAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, fmt.Errorf("query ml_model_versions %s: %w", modelStatus, err)
	}
	return &MLModelVersionDTO{
		ID:               id,
		ArtifactHash:     artifactHash,
		Status:           status,
		CreatedAt:        createdAt.UTC().Format(time.RFC3339),
		ArtifactMetadata: json.RawMessage(metricsJSON),
	}, nil
}

func (r *Reader) loadMLShardSyncState(ctx context.Context) ([]MLShardSyncDTO, error) {
	rows, err := r.pool().Query(ctx, `
		SELECT shard_id, model_version, phase, started_at
		FROM ml_shard_sync_state
		ORDER BY shard_id, model_version`)
	if err != nil {
		return nil, fmt.Errorf("query ml_shard_sync_state: %w", err)
	}
	defer rows.Close()

	var out []MLShardSyncDTO
	for rows.Next() {
		var shardID int
		var modelVersion, phase string
		var startedAt time.Time
		if err := rows.Scan(&shardID, &modelVersion, &phase, &startedAt); err != nil {
			return nil, err
		}
		out = append(out, MLShardSyncDTO{
			ShardID:      shardID,
			ModelVersion: modelVersion,
			Phase:        phase,
			StartedAt:    startedAt.UTC().Format(time.RFC3339),
		})
	}
	return out, rows.Err()
}

func (r *Reader) readMLModelRedis(ctx context.Context) (MLModelRedisDTO, error) {
	redisShards := r.redisShards()
	if len(redisShards) == 0 {
		return MLModelRedisDTO{}, nil
	}

	type shardRedis struct {
		version   string
		hash      string
		appliedAt int64
	}

	var shards []shardRedis
	for _, redisClient := range redisShards {
		if redisClient == nil {
			continue
		}
		pipe := redisClient.Pipeline()
		verCmd := pipe.Get(ctx, "ml:model:version")
		hashCmd := pipe.Get(ctx, "ml:model:hash")
		appliedCmd := pipe.Get(ctx, "ml:model:applied_at")
		if _, err := pipe.Exec(ctx); err != nil && !errors.Is(err, redis.Nil) {
			return MLModelRedisDTO{}, fmt.Errorf("ml model redis pipeline: %w", err)
		}
		version, err := verCmd.Result()
		if err != nil && !errors.Is(err, redis.Nil) {
			return MLModelRedisDTO{}, fmt.Errorf("query ml:model:version: %w", err)
		}
		hash, err := hashCmd.Result()
		if err != nil && !errors.Is(err, redis.Nil) {
			return MLModelRedisDTO{}, fmt.Errorf("query ml:model:hash: %w", err)
		}
		appliedRaw, err := appliedCmd.Result()
		if err != nil && !errors.Is(err, redis.Nil) {
			return MLModelRedisDTO{}, fmt.Errorf("query ml:model:applied_at: %w", err)
		}
		var appliedAt int64
		if appliedRaw != "" {
			appliedAt, _ = strconv.ParseInt(appliedRaw, 10, 64)
		}
		shards = append(shards, shardRedis{
			version:   version,
			hash:      hash,
			appliedAt: appliedAt,
		})
	}

	if len(shards) == 0 {
		return MLModelRedisDTO{}, nil
	}

	ref := shards[0]
	consistent := true
	for _, s := range shards[1:] {
		if s.version != ref.version || s.hash != ref.hash {
			consistent = false
			break
		}
	}

	out := MLModelRedisDTO{
		VersionID:        ref.version,
		Hash:             ref.hash,
		ShardsReporting:  len(shards),
		ShardsConsistent: consistent,
	}
	if ref.appliedAt > 0 {
		out.AppliedAt = time.Unix(ref.appliedAt, 0).UTC().Format(time.RFC3339)
	}
	return out, nil
}
