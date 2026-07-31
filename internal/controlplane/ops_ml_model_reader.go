package controlplane

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"sort"
	"strconv"
	"time"

	"espx/internal/controlplane/adminapi"

	"github.com/jackc/pgx/v5"
	"github.com/redis/go-redis/v9"
)

const mlManualLabelsListLimit = 100

func (r *opsReader) GetMLModelStatus(ctx context.Context) (adminapi.MLModelStatusDTO, error) {
	if r == nil || r.svc == nil || r.svc.GetPool() == nil {
		return adminapi.MLModelStatusDTO{}, fmt.Errorf("postgres pool not configured")
	}

	status := adminapi.MLModelStatusDTO{
		ShardSync: []adminapi.MLShardSyncDTO{},
	}

	active, err := r.loadMLModelVersion(ctx, "ACTIVE")
	if err != nil {
		return adminapi.MLModelStatusDTO{}, err
	}
	if active != nil {
		status.ActiveVersion = active
		status.Importance = topFeatureImportance(active.ArtifactMetadata, 5)
	}

	syncing, err := r.loadMLModelVersion(ctx, "SYNCING")
	if err != nil {
		return adminapi.MLModelStatusDTO{}, err
	}
	if syncing != nil {
		status.SyncingVersion = syncing
		if len(status.Importance) == 0 {
			status.Importance = topFeatureImportance(syncing.ArtifactMetadata, 5)
		}
	}

	shardSync, err := r.loadMLShardSyncState(ctx)
	if err != nil {
		return adminapi.MLModelStatusDTO{}, err
	}
	status.ShardSync = shardSync

	redisStatus, err := r.readMLModelRedis(ctx)
	if err != nil {
		return adminapi.MLModelStatusDTO{}, err
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

func (r *opsReader) readMLDriftReport() (mlEvalReport, error) {
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

func topFeatureImportance(metadata []byte, limit int) []adminapi.MLFeatureImportanceDTO {
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
	out := make([]adminapi.MLFeatureImportanceDTO, len(names))
	for i, name := range names {
		out[i] = adminapi.MLFeatureImportanceDTO{
			Name:  name,
			Value: meta.Importance[name],
		}
	}
	return out
}

func (r *opsReader) AddMLManualLabel(ctx context.Context, ipHash string, label int, reason string) error {
	if r == nil || r.svc == nil || r.svc.GetPool() == nil {
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
	_, err := r.svc.GetPool().Exec(ctx, `
		INSERT INTO ml_manual_labels (ip_hash, label, reason, source, created_at)
		VALUES ($1, $2, $3, 'admin_ui', NOW())
		ON CONFLICT (ip_hash) DO UPDATE SET
			label = EXCLUDED.label,
			reason = EXCLUDED.reason,
			created_at = NOW()`,
		ipHash, label, reason)
	return err
}

func (r *opsReader) ListMLManualLabels(ctx context.Context) ([]adminapi.MLManualLabelDTO, error) {
	if r == nil || r.svc == nil || r.svc.GetPool() == nil {
		return nil, fmt.Errorf("postgres pool not configured")
	}
	rows, err := r.svc.GetPool().Query(ctx, `
		SELECT ip_hash, label, reason, source, created_at
		FROM ml_manual_labels
		ORDER BY created_at DESC
		LIMIT $1`, mlManualLabelsListLimit)
	if err != nil {
		return nil, fmt.Errorf("query ml_manual_labels: %w", err)
	}
	defer rows.Close()

	var out []adminapi.MLManualLabelDTO
	for rows.Next() {
		var row adminapi.MLManualLabelDTO
		var createdAt time.Time
		if err := rows.Scan(&row.IPHash, &row.Label, &row.Reason, &row.Source, &createdAt); err != nil {
			return nil, err
		}
		row.CreatedAt = createdAt.UTC().Format(time.RFC3339)
		out = append(out, row)
	}
	return out, rows.Err()
}

func (r *opsReader) loadMLModelVersion(ctx context.Context, modelStatus string) (*adminapi.MLModelVersionDTO, error) {
	var id, artifactHash, status string
	var metricsJSON []byte
	var createdAt time.Time
	err := r.svc.GetPool().QueryRow(ctx, `
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
	return &adminapi.MLModelVersionDTO{
		ID:               id,
		ArtifactHash:     artifactHash,
		Status:           status,
		CreatedAt:        createdAt.UTC().Format(time.RFC3339),
		ArtifactMetadata: json.RawMessage(metricsJSON),
	}, nil
}

func (r *opsReader) loadMLShardSyncState(ctx context.Context) ([]adminapi.MLShardSyncDTO, error) {
	rows, err := r.svc.GetPool().Query(ctx, `
		SELECT shard_id, model_version, phase, started_at
		FROM ml_shard_sync_state
		ORDER BY shard_id, model_version`)
	if err != nil {
		return nil, fmt.Errorf("query ml_shard_sync_state: %w", err)
	}
	defer rows.Close()

	var out []adminapi.MLShardSyncDTO
	for rows.Next() {
		var shardID int
		var modelVersion, phase string
		var startedAt time.Time
		if err := rows.Scan(&shardID, &modelVersion, &phase, &startedAt); err != nil {
			return nil, err
		}
		out = append(out, adminapi.MLShardSyncDTO{
			ShardID:      shardID,
			ModelVersion: modelVersion,
			Phase:        phase,
			StartedAt:    startedAt.UTC().Format(time.RFC3339),
		})
	}
	return out, rows.Err()
}

func (r *opsReader) readMLModelRedis(ctx context.Context) (adminapi.MLModelRedisDTO, error) {
	rdbs := r.svc.rdbs
	if len(rdbs) == 0 {
		return adminapi.MLModelRedisDTO{}, nil
	}

	type shardRedis struct {
		version   string
		hash      string
		appliedAt int64
	}

	var shards []shardRedis
	for _, rdb := range rdbs {
		if rdb == nil {
			continue
		}
		version, err := rdb.Get(ctx, "ml:model:version").Result()
		if err != nil && !errors.Is(err, redis.Nil) {
			return adminapi.MLModelRedisDTO{}, fmt.Errorf("query ml:model:version: %w", err)
		}
		hash, err := rdb.Get(ctx, "ml:model:hash").Result()
		if err != nil && !errors.Is(err, redis.Nil) {
			return adminapi.MLModelRedisDTO{}, fmt.Errorf("query ml:model:hash: %w", err)
		}
		appliedRaw, err := rdb.Get(ctx, "ml:model:applied_at").Result()
		if err != nil && !errors.Is(err, redis.Nil) {
			return adminapi.MLModelRedisDTO{}, fmt.Errorf("query ml:model:applied_at: %w", err)
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
		return adminapi.MLModelRedisDTO{}, nil
	}

	ref := shards[0]
	consistent := true
	for _, s := range shards[1:] {
		if s.version != ref.version || s.hash != ref.hash {
			consistent = false
			break
		}
	}

	out := adminapi.MLModelRedisDTO{
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
