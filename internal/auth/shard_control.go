package auth

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
)

func CheckTokenRevocationShards(ctx context.Context, rdbs []redis.UniversalClient, payload *Payload) (bool, error) {
	if payload == nil || len(rdbs) == 0 {
		return false, nil
	}
	ctxRevoked, cancel := context.WithTimeout(ctx, revocationCheckTimeout)
	defer cancel()
	for i, rdb := range rdbs {
		if rdb == nil {
			continue
		}
		revoked, err := CheckTokenRevocation(ctxRevoked, rdb, payload)
		if err != nil {
			return false, fmt.Errorf("shard %d: %w", i, err)
		}
		if revoked {
			return true, nil
		}
	}
	return false, nil
}

func RevokeUserAccessShards(ctx context.Context, rdbs []redis.UniversalClient, userID uuid.UUID, ttl time.Duration) error {
	if len(rdbs) == 0 {
		return nil
	}
	for i, rdb := range rdbs {
		if rdb == nil {
			continue
		}
		if err := RevokeUserAccess(ctx, rdb, userID, ttl); err != nil {
			return fmt.Errorf("shard %d: %w", i, err)
		}
	}
	return nil
}

func ClearUserRevocationShards(ctx context.Context, rdbs []redis.UniversalClient, userID uuid.UUID) error {
	if len(rdbs) == 0 {
		return nil
	}
	for i, rdb := range rdbs {
		if rdb == nil {
			continue
		}
		if err := ClearUserRevocation(ctx, rdb, userID); err != nil {
			return fmt.Errorf("shard %d: %w", i, err)
		}
	}
	return nil
}

func PickAuthControlShard(rdbs []redis.UniversalClient) redis.UniversalClient {
	for _, rdb := range rdbs {
		if rdb != nil {
			return rdb
		}
	}
	return nil
}
