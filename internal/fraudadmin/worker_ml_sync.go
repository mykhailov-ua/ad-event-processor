package fraudadmin

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"ad-event-processor/internal/database"
	"ad-event-processor/pkg/coldpath"

	db "ad-event-processor/internal/domain/db"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/redis/go-redis/v9"
)

type MLSyncHost interface {
	Pool() *pgxpool.Pool
	RedisShardCount() int
	RedisShard(shardID int) redis.UniversalClient
	ClickHouseQuery() *database.ClickHouseQuery
	ClickHouseOpContext(ctx context.Context) (context.Context, context.CancelFunc)
	SyncMLModelMetaOnShard(ctx context.Context, redisClient redis.UniversalClient, versionID, hash string, appliedAt int64) error
	CheckAndHandleStaleEpochs(ctx context.Context) error
}

type mlModelVersionOutboxPayload struct {
	ModelVersion string `json:"model_version"`
	Hash         string `json:"hash"`
	ShardID      int    `json:"shard_id"`
}

type MLSyncOrchestrator struct {
	host MLSyncHost
}

func NewMLSyncOrchestrator(host MLSyncHost) *MLSyncOrchestrator {
	return &MLSyncOrchestrator{host: host}
}

func NewFraudModelSyncOrchestrator(host MLSyncHost) *MLSyncOrchestrator {
	return NewMLSyncOrchestrator(host)
}

type MLSyncWorker struct {
	orchestrator *MLSyncOrchestrator
	interval     time.Duration
}

func NewMLSyncWorker(host MLSyncHost, interval time.Duration) *MLSyncWorker {
	if interval <= 0 {
		interval = 30 * time.Second
	}
	return &MLSyncWorker{
		orchestrator: NewMLSyncOrchestrator(host),
		interval:     interval,
	}
}

func NewFraudModelSyncWorker(host MLSyncHost, interval time.Duration) *MLSyncWorker {
	return NewMLSyncWorker(host, interval)
}

func (w *MLSyncWorker) Start(ctx context.Context) {
	if w == nil || w.orchestrator == nil {
		return
	}
	ticker := time.NewTicker(w.interval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			if err := w.orchestrator.Tick(ctx); err != nil {
				slog.Error("fraud model sync tick failed", "err", err)
			}
			if err := w.orchestrator.host.CheckAndHandleStaleEpochs(ctx); err != nil {
				slog.Warn("fraud model stale epoch handling", "err", err)
			}
		}
	}
}

func (o *MLSyncOrchestrator) Tick(ctx context.Context) error {
	pool := o.host.Pool()
	if pool == nil {
		return fmt.Errorf("postgres pool not available")
	}

	var versionID, artifactHash string
	err := pool.QueryRow(ctx, "SELECT id, artifact_hash FROM ml_model_versions WHERE status = 'SYNCING' LIMIT 1").Scan(&versionID, &artifactHash)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil
		}
		return fmt.Errorf("failed to query syncing model version: %w", err)
	}

	numShards := o.host.RedisShardCount()
	if numShards == 0 {
		return fmt.Errorf("no redis shards configured")
	}

	rows, err := pool.Query(ctx, "SELECT shard_id, phase, started_at FROM ml_shard_sync_state WHERE model_version = $1", versionID)
	if err != nil {
		return fmt.Errorf("failed to query shard sync states: %w", err)
	}
	defer rows.Close()

	type shardState struct {
		phase     string
		startedAt time.Time
	}
	states := make(map[int]shardState)
	for rows.Next() {
		var shardID int
		var phase string
		var startedAt time.Time
		if err := rows.Scan(&shardID, &phase, &startedAt); err != nil {
			return fmt.Errorf("failed to scan shard sync state: %w", err)
		}
		states[shardID] = shardState{phase: phase, startedAt: startedAt}
	}

	activeSyncShard := -1
	for id, state := range states {
		if state.phase == "SYNC" {
			activeSyncShard = id
			break
		}
	}

	if activeSyncShard != -1 {
		state := states[activeSyncShard]
		if time.Since(state.startedAt) > 180*time.Second {
			slog.Error("fraud model sync timed out on shard, triggering rollback", "shard_id", activeSyncShard, "version", versionID)
			return o.rollbackShard(ctx, activeSyncShard, versionID)
		}

		passed, err := o.runCanaryCheck(ctx, activeSyncShard, versionID)
		if err != nil {
			slog.Warn("fraud model sync canary check failed with error, rolling back", "shard_id", activeSyncShard, "version", versionID, "err", err)
			return o.rollbackShard(ctx, activeSyncShard, versionID)
		}

		if passed {
			slog.Info("fraud model sync canary passed, cutting over shard to ACTIVE", "shard_id", activeSyncShard, "version", versionID)
			_, err = pool.Exec(ctx, "UPDATE ml_shard_sync_state SET phase = 'ACTIVE' WHERE shard_id = $1 AND model_version = $2", activeSyncShard, versionID)
			if err != nil {
				return fmt.Errorf("failed to update shard phase to ACTIVE: %w", err)
			}

			redisClient := o.host.RedisShard(activeSyncShard)
			if redisClient != nil {
				if err := o.host.SyncMLModelMetaOnShard(ctx, redisClient, versionID, artifactHash, time.Now().Unix()); err != nil {
					return fmt.Errorf("failed to set ml model keys: %w", err)
				}
			}
		} else {
			slog.Warn("fraud model sync canary failed (high FP rate), rolling back", "shard_id", activeSyncShard, "version", versionID)
			return o.rollbackShard(ctx, activeSyncShard, versionID)
		}

		return nil
	}

	nextShardToSync := -1
	for i := range numShards {
		state, exists := states[i]
		if !exists || state.phase == "ROLLBACK" {
			nextShardToSync = i
			break
		}
	}

	if nextShardToSync != -1 {
		slog.Info("fraud model sync starting on shard", "shard_id", nextShardToSync, "version", versionID)
		_, err = pool.Exec(ctx, `
			INSERT INTO ml_shard_sync_state (shard_id, model_version, phase, started_at)
			VALUES ($1, $2, 'SYNC', NOW())
			ON CONFLICT (shard_id, model_version) DO UPDATE SET phase = 'SYNC', started_at = NOW()`,
			nextShardToSync, versionID)
		if err != nil {
			return fmt.Errorf("failed to insert shard sync state: %w", err)
		}

		payload, err := coldpath.MarshalJSON(mlModelVersionOutboxPayload{
			ModelVersion: versionID,
			Hash:         artifactHash,
			ShardID:      nextShardToSync,
		})
		if err != nil {
			return err
		}

		_, err = db.New(pool).CreateOutboxEvent(ctx, db.CreateOutboxEventParams{
			EventType: "ML_MODEL_VERSION",
			Payload:   payload,
		})
		return err
	}

	slog.Info("fraud model sync complete on all shards, activating version globally", "version", versionID)
	return pgx.BeginFunc(ctx, pool, func(tx pgx.Tx) error {
		_, err := tx.Exec(ctx, "UPDATE ml_model_versions SET status = 'ACTIVE' WHERE id = $1", versionID)
		if err != nil {
			return err
		}
		_, err = tx.Exec(ctx, "UPDATE ml_model_versions SET status = 'RETIRED' WHERE id <> $1 AND status = 'ACTIVE'", versionID)
		return err
	})
}

func (o *MLSyncOrchestrator) rollbackShard(ctx context.Context, shardID int, versionID string) error {
	pool := o.host.Pool()
	_, err := pool.Exec(ctx, "UPDATE ml_shard_sync_state SET phase = 'ROLLBACK' WHERE shard_id = $1 AND model_version = $2", shardID, versionID)
	if err != nil {
		return fmt.Errorf("failed to update shard phase to ROLLBACK: %w", err)
	}

	var prevVersionID, prevHash string
	err = pool.QueryRow(ctx, "SELECT id, artifact_hash FROM ml_model_versions WHERE status = 'ACTIVE' LIMIT 1").Scan(&prevVersionID, &prevHash)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			redisClient := o.host.RedisShard(shardID)
			if redisClient != nil {
				redisClient.Del(ctx, "ml:model:version", "ml:model:hash", "ml:model:applied_at")
			}
			return nil
		}
		return fmt.Errorf("failed to query previous active model version: %w", err)
	}

	redisClient := o.host.RedisShard(shardID)
	if redisClient != nil {
		if err := o.host.SyncMLModelMetaOnShard(ctx, redisClient, prevVersionID, prevHash, time.Now().Unix()); err != nil {
			return fmt.Errorf("rollback ml model keys: %w", err)
		}
	}

	return nil
}

func (o *MLSyncOrchestrator) runCanaryCheck(ctx context.Context, shardID int, versionID string) (bool, error) {
	ch := o.host.ClickHouseQuery()
	if ch == nil {
		return true, nil
	}

	clickhouseCtx, cancel := o.host.ClickHouseOpContext(ctx)
	defer cancel()

	query := `
		SELECT window_start, ip_address, campaign_id, events, clicks, spend_micro, budget_limit_micro, unique_users, unique_uas
		FROM ad_event_processor.ml_features_1m
		WHERE window_start >= subtractHours(now(), 1)
		LIMIT 1000`

	rows, err := ch.Query(clickhouseCtx, query)
	if err != nil {
		return false, fmt.Errorf("clickhouse query failed: %w", err)
	}
	defer func() { _ = rows.Close() }()

	var totalRows int
	var highScores int

	for rows.Next() {
		var windowStart time.Time
		var ipAddress, campaignID string
		var events, clicks, uniqueUsers, uniqueUAs uint64
		var spendMicro, budgetLimitMicro int64
		if err := rows.Scan(&windowStart, &ipAddress, &campaignID, &events, &clicks, &spendMicro, &budgetLimitMicro, &uniqueUsers, &uniqueUAs); err != nil {
			return false, fmt.Errorf("clickhouse scan failed: %w", err)
		}
		totalRows++

		if clicks > 10 && clicks*2 > events {
			highScores++
		}
	}

	if totalRows == 0 {
		return true, nil
	}

	fpRate := float64(highScores) / float64(totalRows)
	slog.Info("fraud model sync canary stats", "shard_id", shardID, "total_rows", totalRows, "high_scores", highScores, "fp_rate", fpRate)

	if fpRate > 0.10 {
		return false, nil
	}

	return true, nil
}
