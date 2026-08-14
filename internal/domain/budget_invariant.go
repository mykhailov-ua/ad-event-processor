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

func ReadBudgetInvariant(ctx context.Context, pool *pgxpool.Pool, rdb redis.Cmdable, campaignID uuid.UUID) (BudgetInvariantSnapshot, error) {
	var snap BudgetInvariantSnapshot
	snap.CampaignID = campaignID

	err := pool.QueryRow(ctx,
		`SELECT budget_limit, current_spend FROM campaigns WHERE id = $1`, ToUUID(campaignID),
	).Scan(&snap.BudgetLimit, &snap.PGCurrentSpend)
	if err != nil {
		return snap, fmt.Errorf("read campaign spend from postgres: %w", err)
	}

	budgetKey := budgetCampaignKey(campaignID)
	syncKey := campaignSyncKey(campaignID)

	syncDelta, err := rdb.Get(ctx, syncKey).Int64()
	if errors.Is(err, redis.Nil) {
		syncDelta = 0
	} else if err != nil {
		return snap, fmt.Errorf("read %s: %w", syncKey, err)
	}
	snap.SyncDelta = syncDelta

	remaining, err := rdb.Get(ctx, budgetKey).Int64()
	if errors.Is(err, redis.Nil) {
		remaining = snap.BudgetLimit - snap.PGCurrentSpend - snap.SyncDelta
		if remaining < 0 {
			remaining = 0
		}
	} else if err != nil {
		return snap, fmt.Errorf("read %s: %w", budgetKey, err)
	}
	snap.RedisRemaining = remaining

	return snap, nil
}

func VerifyBudgetInvariant(ctx context.Context, pool *pgxpool.Pool, rdb redis.Cmdable, campaignID uuid.UUID) error {
	snap, err := ReadBudgetInvariant(ctx, pool, rdb, campaignID)
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

func AssertBudgetInvariant(t testing.TB, ctx context.Context, pool *pgxpool.Pool, rdb redis.Cmdable, campaignID uuid.UUID) {
	t.Helper()

	if err := VerifyBudgetInvariant(ctx, pool, rdb, campaignID); err != nil {
		t.Fatal(err)
	}
}
