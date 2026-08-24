package domain

import (
	"context"
	"errors"
	"fmt"
	"testing"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/redis/go-redis/v9"
)

const budgetInvariantToleranceMicro = int64(1)

type BudgetInvariantSnapshot struct {
	CampaignID     uuid.UUID
	BudgetLimit    int64
	RedisRemaining int64
	SyncDelta      int64
	PGCurrentSpend int64
}

func ReadBudgetInvariant(ctx context.Context, pool *pgxpool.Pool, redisClient redis.Cmdable, campaignID uuid.UUID) (BudgetInvariantSnapshot, error) {
	snaps, err := ReadBudgetInvariants(ctx, pool, redisClient, []uuid.UUID{campaignID})
	if err != nil {
		return BudgetInvariantSnapshot{CampaignID: campaignID}, err
	}
	snap, ok := snaps[campaignID]
	if !ok {
		return BudgetInvariantSnapshot{CampaignID: campaignID}, fmt.Errorf("read campaign spend from postgres: campaign %s not found", campaignID)
	}
	return snap, nil
}

func ReadBudgetInvariants(ctx context.Context, pool *pgxpool.Pool, redisClient redis.Cmdable, campaignIDs []uuid.UUID) (map[uuid.UUID]BudgetInvariantSnapshot, error) {
	out := make(map[uuid.UUID]BudgetInvariantSnapshot, len(campaignIDs))
	if pool == nil || len(campaignIDs) == 0 {
		return out, nil
	}

	pgIDs := make([]uuid.UUID, len(campaignIDs))
	copy(pgIDs, campaignIDs)

	rows, err := pool.Query(ctx,
		`SELECT id, budget_limit, current_spend FROM campaigns WHERE id = ANY($1::uuid[])`,
		pgIDs,
	)
	if err != nil {
		return nil, fmt.Errorf("read campaign spends from postgres: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		var snap BudgetInvariantSnapshot
		if err := rows.Scan(&snap.CampaignID, &snap.BudgetLimit, &snap.PGCurrentSpend); err != nil {
			return nil, fmt.Errorf("scan campaign spend: %w", err)
		}
		out[snap.CampaignID] = snap
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("read campaign spends from postgres: %w", err)
	}

	if redisClient == nil {
		return out, nil
	}

	pipe := redisClient.Pipeline()
	budgetCmds := make([]*redis.StringCmd, len(campaignIDs))
	syncCmds := make([]*redis.StringCmd, len(campaignIDs))
	for i, id := range campaignIDs {
		budgetCmds[i] = pipe.Get(ctx, budgetCampaignKey(id))
		syncCmds[i] = pipe.Get(ctx, campaignSyncKey(id))
	}
	if _, err := pipe.Exec(ctx); err != nil && !errors.Is(err, redis.Nil) {
		return nil, fmt.Errorf("read budget keys from redis: %w", err)
	}

	for i, id := range campaignIDs {
		snap, ok := out[id]
		if !ok {
			continue
		}
		syncDelta, err := syncCmds[i].Int64()
		if errors.Is(err, redis.Nil) {
			syncDelta = 0
		} else if err != nil {
			return nil, fmt.Errorf("read %s: %w", campaignSyncKey(id), err)
		}
		snap.SyncDelta = syncDelta

		remaining, err := budgetCmds[i].Int64()
		if errors.Is(err, redis.Nil) {
			remaining = snap.BudgetLimit - snap.PGCurrentSpend - snap.SyncDelta
			if remaining < 0 {
				remaining = 0
			}
		} else if err != nil {
			return nil, fmt.Errorf("read %s: %w", budgetCampaignKey(id), err)
		}
		snap.RedisRemaining = remaining
		out[id] = snap
	}

	return out, nil
}

func VerifyBudgetInvariant(ctx context.Context, pool *pgxpool.Pool, redisClient redis.Cmdable, campaignID uuid.UUID) error {
	snap, err := ReadBudgetInvariant(ctx, pool, redisClient, campaignID)
	if err != nil {
		return err
	}
	redisSpend := snap.BudgetLimit - snap.RedisRemaining
	pgPlusSync := snap.PGCurrentSpend + snap.SyncDelta
	diff := redisSpend - pgPlusSync
	if diff < -budgetInvariantToleranceMicro || diff > budgetInvariantToleranceMicro {
		return fmt.Errorf(
			"budget invariant violated for campaign %s: diff=%d tolerance=%d",
			campaignID, diff, budgetInvariantToleranceMicro,
		)
	}
	return nil
}

func AssertBudgetInvariant(t testing.TB, ctx context.Context, pool *pgxpool.Pool, redisClient redis.Cmdable, campaignID uuid.UUID) {
	t.Helper()

	if err := VerifyBudgetInvariant(ctx, pool, redisClient, campaignID); err != nil {
		t.Fatal(err)
	}
}
