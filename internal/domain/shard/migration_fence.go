package shard

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	redis "github.com/redis/go-redis/v9"
)

const migrationFenceTTL = 24 * time.Hour

func BumpMigrationFences(
	ctx context.Context,
	pool *pgxpool.Pool,
	src redis.Cmdable,
	campaignIDs []uuid.UUID,
) error {
	if pool == nil || src == nil || len(campaignIDs) == 0 {
		return nil
	}
	pgIDs := make([]uuid.UUID, len(campaignIDs))
	copy(pgIDs, campaignIDs)

	rows, err := pool.Query(ctx, `
		UPDATE campaigns
		SET migration_gen = migration_gen + 1
		WHERE id = ANY($1::uuid[])
		RETURNING id, migration_gen`,
		pgIDs,
	)
	if err != nil {
		return fmt.Errorf("bump migration_gen: %w", err)
	}
	defer rows.Close()

	type fenceRow struct {
		id  uuid.UUID
		gen int64
	}
	var bumped []fenceRow
	for rows.Next() {
		var r fenceRow
		if err := rows.Scan(&r.id, &r.gen); err != nil {
			return err
		}
		bumped = append(bumped, r)
	}
	if err := rows.Err(); err != nil {
		return err
	}

	pipe := src.Pipeline()
	for _, r := range bumped {
		key := MigrationFenceRedisKey(r.id)
		pipe.Set(ctx, key, r.gen, migrationFenceTTL)
	}
	if _, err := pipe.Exec(ctx); err != nil {
		return fmt.Errorf("set migration fence keys: %w", err)
	}
	return nil
}

func SetBudgetFrozen(ctx context.Context, redisClient redis.Cmdable, campaignID uuid.UUID) error {
	if redisClient == nil {
		return fmt.Errorf("nil redis client")
	}
	return redisClient.Set(ctx, BudgetFrozenRedisKey(campaignID), "1", 0).Err()
}

func ClearBudgetFrozen(ctx context.Context, redisClient redis.Cmdable, campaignID uuid.UUID) error {
	if redisClient == nil {
		return fmt.Errorf("nil redis client")
	}
	return redisClient.Del(ctx, BudgetFrozenRedisKey(campaignID)).Err()
}
