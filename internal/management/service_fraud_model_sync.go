package management

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"strconv"
	"time"

	db "espx/internal/ingestion/sqlc"
	"espx/pkg/coldpath"

	"github.com/jackc/pgx/v5"
	"github.com/redis/go-redis/v9"
)

type FraudModelVersionPayload struct {
	ModelVersion string `json:"model_version"`
	Hash         string `json:"hash"`
	ShardID      int    `json:"shard_id"`
}

type FraudModelSyncOrchestrator struct {
	svc *Service
}

func NewFraudModelSyncOrchestrator(svc *Service) *FraudModelSyncOrchestrator {
	return &FraudModelSyncOrchestrator{svc: svc}
}

func (o *FraudModelSyncOrchestrator) Tick(ctx context.Context) error {
	pool := o.svc.GetPool()
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

	numShards := len(o.svc.rdbs)
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

	var activeSyncShard = -1
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
			slog.Warn("fraud model sync canary check failed with error, rolling back", "shard_id", activeSyncShard, "version", versionID, "error", err)
			return o.rollbackShard(ctx, activeSyncShard, versionID)
		}

		if passed {
			slog.Info("fraud model sync canary passed, cutting over shard to ACTIVE", "shard_id", activeSyncShard, "version", versionID)
			_, err = pool.Exec(ctx, "UPDATE ml_shard_sync_state SET phase = 'ACTIVE' WHERE shard_id = $1 AND model_version = $2", activeSyncShard, versionID)
			if err != nil {
				return fmt.Errorf("failed to update shard phase to ACTIVE: %w", err)
			}

			rdb := o.svc.rdbs[activeSyncShard]
			if rdb != nil {
				rdb.Set(ctx, "ml:model:version", versionID, 0)
				rdb.Set(ctx, "ml:model:hash", artifactHash, 0)
				rdb.Set(ctx, "ml:model:applied_at", time.Now().Unix(), 0)
			}
		} else {
			slog.Warn("fraud model sync canary failed (high FP rate), rolling back", "shard_id", activeSyncShard, "version", versionID)
			return o.rollbackShard(ctx, activeSyncShard, versionID)
		}

		return nil
	}

	var nextShardToSync = -1
	for i := 0; i < numShards; i++ {
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

		payload, err := coldpath.MarshalJSON(FraudModelVersionPayload{
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
		if err != nil {
			return err
		}
		return nil
	})
}

func (o *FraudModelSyncOrchestrator) rollbackShard(ctx context.Context, shardID int, versionID string) error {
	pool := o.svc.GetPool()
	_, err := pool.Exec(ctx, "UPDATE ml_shard_sync_state SET phase = 'ROLLBACK' WHERE shard_id = $1 AND model_version = $2", shardID, versionID)
	if err != nil {
		return fmt.Errorf("failed to update shard phase to ROLLBACK: %w", err)
	}

	var prevVersionID, prevHash string
	err = pool.QueryRow(ctx, "SELECT id, artifact_hash FROM ml_model_versions WHERE status = 'ACTIVE' LIMIT 1").Scan(&prevVersionID, &prevHash)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			rdb := o.svc.rdbs[shardID]
			if rdb != nil {
				rdb.Del(ctx, "ml:model:version", "ml:model:hash", "ml:model:applied_at")
			}
			return nil
		}
		return fmt.Errorf("failed to query previous active model version: %w", err)
	}

	rdb := o.svc.rdbs[shardID]
	if rdb != nil {
		rdb.Set(ctx, "ml:model:version", prevVersionID, 0)
		rdb.Set(ctx, "ml:model:hash", prevHash, 0)
		rdb.Set(ctx, "ml:model:applied_at", time.Now().Unix(), 0)
	}

	return nil
}

func (o *FraudModelSyncOrchestrator) runCanaryCheck(ctx context.Context, shardID int, versionID string) (bool, error) {
	if o.svc.chQuery == nil {
		return true, nil
	}

	query := `
		SELECT window_start, ip_address, campaign_id, events, clicks, spend_micro, budget_limit_micro, unique_users, unique_uas
		FROM ad_event_processor.ml_features_1m
		WHERE window_start >= now() - INTERVAL 1 HOUR
		LIMIT 1000`

	rows, err := o.svc.chQuery.Query(ctx, query)
	if err != nil {
		return false, fmt.Errorf("clickhouse query failed: %w", err)
	}
	defer rows.Close()

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

func (s *Service) CheckAndHandleStaleEpochs(ctx context.Context) error {
	if len(s.rdbs) == 0 {
		return nil
	}

	now := time.Now().Unix()
	var maxAppliedAt int64
	var foundStale = false

	for i, rdb := range s.rdbs {
		if rdb == nil {
			continue
		}
		val, err := rdb.Get(ctx, "ml:model:applied_at").Result()
		if err != nil {
			if errors.Is(err, redis.Nil) {
				continue
			}
			slog.Warn("failed to query ml:model:applied_at on shard", "shard_id", i, "error", err)
			continue
		}
		appliedAt, err := strconv.ParseInt(val, 10, 64)
		if err != nil {
			continue
		}
		if appliedAt > maxAppliedAt {
			maxAppliedAt = appliedAt
		}
	}

	if maxAppliedAt > 0 && now-maxAppliedAt > 600 {
		foundStale = true
	}

	if foundStale {
		slog.Warn("ML channel is STALE, tightening suspect rate limits", "lag_seconds", now-maxAppliedAt)
		var currentPctStr string
		err := s.pool.QueryRow(ctx, "SELECT value FROM system_settings WHERE key = 'fraud_rl_suspect_pct'").Scan(&currentPctStr)
		if err != nil {
			if errors.Is(err, pgx.ErrNoRows) {
				currentPctStr = "50"
			} else {
				return err
			}
		}

		currentPct, _ := strconv.Atoi(currentPctStr)
		newPct := currentPct / 2
		if newPct < 10 {
			newPct = 10
		}

		if newPct != currentPct {
			err = s.UpdateSettings(ctx, map[string]string{
				"fraud_rl_suspect_pct": strconv.Itoa(newPct),
			})
			if err != nil {
				return fmt.Errorf("failed to tighten suspect rate limit: %w", err)
			}
		}
	}

	return nil
}
