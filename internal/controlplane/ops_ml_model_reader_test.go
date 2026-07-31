package controlplane

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"

	"espx/internal/database"
	"espx/internal/domain"

	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestOpsReader_GetMLModelStatus(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	pool, cleanupDB := database.SetupTestDB(t)
	defer cleanupDB()

	rdb, cleanupRedis := database.SetupTestRedis(t)
	defer cleanupRedis()

	sharder := domain.NewStaticSlotSharder(1)
	svc := NewService(pool, []redis.UniversalClient{rdb}, sharder, nil)
	defer svc.Close()

	ctx := context.Background()
	artifactMeta := `{"version":"v1","metrics":{"auc":0.9},"importance":{"events":0.5,"clicks":0.3,"unique_users":0.1,"spend_micro":0.05,"budget_limit_micro":0.05}}`
	_, err := pool.Exec(ctx, `
		INSERT INTO ml_model_versions (id, artifact_hash, metrics_json, status, created_at)
		VALUES ('v1', 'hash1', $1, 'ACTIVE', $2)`,
		artifactMeta, time.Now().UTC().Add(-time.Hour))
	require.NoError(t, err)

	_, err = pool.Exec(ctx, `
		INSERT INTO ml_model_versions (id, artifact_hash, metrics_json, status, created_at)
		VALUES ('v2', 'hash2', '{}', 'SYNCING', $1)`,
		time.Now().UTC())
	require.NoError(t, err)

	_, err = pool.Exec(ctx, `
		INSERT INTO ml_shard_sync_state (shard_id, model_version, phase, started_at)
		VALUES (0, 'v2', 'SYNC', $1)`,
		time.Now().UTC())
	require.NoError(t, err)

	appliedAt := time.Now().Unix()
	require.NoError(t, rdb.Set(ctx, "ml:model:version", "v2", 0).Err())
	require.NoError(t, rdb.Set(ctx, "ml:model:hash", "hash2", 0).Err())
	require.NoError(t, rdb.Set(ctx, "ml:model:applied_at", appliedAt, 0).Err())

	tmpDir := t.TempDir()
	reportPath := filepath.Join(tmpDir, "shadow_eval_report.json")
	report := map[string]any{
		"status":    "ok",
		"precision": 0.85,
		"recall":    0.72,
		"drift": map[string]any{
			"status":         "ok",
			"drift_detected": true,
			"max_drift":      0.35,
		},
	}
	reportBytes, err := json.Marshal(report)
	require.NoError(t, err)
	require.NoError(t, os.WriteFile(reportPath, reportBytes, 0644))
	t.Setenv("FRAUD_EVAL_REPORT_PATH", reportPath)

	reader := &opsReader{svc: svc}
	status, err := reader.GetMLModelStatus(ctx)
	require.NoError(t, err)

	require.NotNil(t, status.ActiveVersion)
	assert.Equal(t, "v1", status.ActiveVersion.ID)
	assert.Equal(t, "hash1", status.ActiveVersion.ArtifactHash)
	assert.Equal(t, "ACTIVE", status.ActiveVersion.Status)
	assert.JSONEq(t, artifactMeta, string(status.ActiveVersion.ArtifactMetadata))

	require.NotNil(t, status.SyncingVersion)
	assert.Equal(t, "v2", status.SyncingVersion.ID)

	require.Len(t, status.ShardSync, 1)
	assert.Equal(t, 0, status.ShardSync[0].ShardID)
	assert.Equal(t, "v2", status.ShardSync[0].ModelVersion)
	assert.Equal(t, "SYNC", status.ShardSync[0].Phase)

	assert.Equal(t, "v2", status.Redis.VersionID)
	assert.Equal(t, "hash2", status.Redis.Hash)
	assert.Equal(t, 1, status.Redis.ShardsReporting)
	assert.True(t, status.Redis.ShardsConsistent)
	assert.NotEmpty(t, status.Redis.AppliedAt)

	assert.True(t, status.DriftDetected)
	assert.InDelta(t, 0.85, status.Precision, 0.001)
	assert.InDelta(t, 0.72, status.Recall, 0.001)
	require.NotEmpty(t, status.Drift)
	var driftBlock map[string]any
	require.NoError(t, json.Unmarshal(status.Drift, &driftBlock))
	assert.Equal(t, true, driftBlock["drift_detected"])

	require.Len(t, status.Importance, 5)
	assert.Equal(t, "events", status.Importance[0].Name)
	assert.InDelta(t, 0.5, status.Importance[0].Value, 0.001)
	assert.Equal(t, "clicks", status.Importance[1].Name)
}

func TestOpsReader_AddMLManualLabel_Validation(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	pool, cleanupDB := database.SetupTestDB(t)
	defer cleanupDB()

	sharder := domain.NewStaticSlotSharder(1)
	svc := NewService(pool, nil, sharder, nil)
	defer svc.Close()

	reader := &opsReader{svc: svc}
	ctx := context.Background()

	err := reader.AddMLManualLabel(ctx, "short", 1, "bad hash")
	require.Error(t, err)
	var ve validationError
	require.ErrorAs(t, err, &ve)

	err = reader.AddMLManualLabel(ctx, "0123456789abcdef0123456789abcdef", 2, "bad label")
	require.Error(t, err)
	require.ErrorAs(t, err, &ve)

	validHash := "0123456789abcdef0123456789abcdef"
	require.NoError(t, reader.AddMLManualLabel(ctx, validHash, 1, "fraud signal"))

	labels, err := reader.ListMLManualLabels(ctx)
	require.NoError(t, err)
	require.Len(t, labels, 1)
	assert.Equal(t, validHash, labels[0].IPHash)
	assert.Equal(t, 1, labels[0].Label)
	assert.Equal(t, "fraud signal", labels[0].Reason)
	assert.Equal(t, "admin_ui", labels[0].Source)
	assert.NotEmpty(t, labels[0].CreatedAt)
}

func TestTopFeatureImportance(t *testing.T) {
	t.Parallel()
	meta := []byte(`{"importance":{"z_feat":0.1,"a_feat":0.9,"m_feat":0.5}}`)
	out := topFeatureImportance(meta, 2)
	require.Len(t, out, 2)
	assert.Equal(t, "a_feat", out[0].Name)
	assert.InDelta(t, 0.9, out[0].Value, 0.001)
	assert.Equal(t, "m_feat", out[1].Name)
}
