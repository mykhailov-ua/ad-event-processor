package shardadmin

import (
	"context"
	"fmt"

	"ad-event-processor/internal/domain"
	db "ad-event-processor/internal/domain/db"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/redis/go-redis/v9"
)

func mapRepoUpdateSlotState(ctx context.Context, pool *pgxpool.Pool, version int32, slot, shard int16, state db.RedisSlotState) error {
	if pool == nil {
		return fmt.Errorf("nil pool")
	}
	return db.New(pool).UpdateSlotMapEntry(ctx, db.UpdateSlotMapEntryParams{
		Version: version,
		Slot:    slot,
		ShardID: shard,
		State:   state,
	})
}

func verifyActivationKeysPostCopy(
	ctx context.Context,
	src, dst redis.Cmdable,
	catalog *domain.CampaignRedisKeyCatalog,
	campaignIDs []uuid.UUID,
) error {
	if catalog == nil || len(campaignIDs) == 0 {
		return nil
	}
	keys := make([]string, 0, len(campaignIDs))
	for _, id := range campaignIDs {
		keys = append(keys, catalog.ActivationRequiredKeys(id)...)
	}
	if len(keys) == 0 {
		return nil
	}

	srcPipe := src.Pipeline()
	srcCmds := make([]*redis.IntCmd, len(keys))
	for i, key := range keys {
		srcCmds[i] = srcPipe.Exists(ctx, key)
	}
	if _, err := srcPipe.Exec(ctx); err != nil {
		return fmt.Errorf("post-copy exists src pipeline: %w", err)
	}

	needDst := make([]string, 0, len(keys))
	for i, cmd := range srcCmds {
		n, err := cmd.Result()
		if err != nil {
			return fmt.Errorf("post-copy exists src %q: %w", keys[i], err)
		}
		if n > 0 {
			needDst = append(needDst, keys[i])
		}
	}
	if len(needDst) == 0 {
		return nil
	}

	dstPipe := dst.Pipeline()
	dstCmds := make([]*redis.IntCmd, len(needDst))
	for i, key := range needDst {
		dstCmds[i] = dstPipe.Exists(ctx, key)
	}
	if _, err := dstPipe.Exec(ctx); err != nil {
		return fmt.Errorf("post-copy exists dst pipeline: %w", err)
	}
	for i, cmd := range dstCmds {
		n, err := cmd.Result()
		if err != nil {
			return fmt.Errorf("post-copy exists dst %q: %w", needDst[i], err)
		}
		if n == 0 {
			return fmt.Errorf("post-copy verify: %q missing on target shard", needDst[i])
		}
	}
	return nil
}
