package identity

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
)

const revocationCheckTimeout = 100 * time.Millisecond

const defaultUserRevocationTTL = 24 * time.Hour

func CheckTokenRevocation(ctx context.Context, rdb redis.UniversalClient, payload *Payload) (revoked bool, err error) {
	if rdb == nil || payload == nil {
		return false, nil
	}

	ctxRevoked, cancel := context.WithTimeout(ctx, revocationCheckTimeout)
	defer cancel()

	cmds, errPipe := rdb.Pipelined(ctxRevoked, func(pipe redis.Pipeliner) error {
		pipe.Exists(ctxRevoked, "revoked:token:"+payload.ID.String())
		pipe.Exists(ctxRevoked, "revoked:session:"+payload.SessionID.String())
		pipe.Exists(ctxRevoked, "revoked:user:"+payload.UserID.String())
		return nil
	})
	if errPipe != nil {
		return false, errPipe
	}
	if len(cmds) != 3 {
		return false, fmt.Errorf("unexpected pipeline commands count: got %d want 3", len(cmds))
	}

	for _, cmd := range cmds {
		intCmd, ok := cmd.(*redis.IntCmd)
		if !ok {
			return false, fmt.Errorf("unexpected pipeline command type")
		}
		exists, errExists := intCmd.Result()
		if errExists != nil {
			return false, errExists
		}
		if exists > 0 {
			return true, nil
		}
	}
	return false, nil
}

func RevokeUserAccess(ctx context.Context, rdb redis.UniversalClient, userID uuid.UUID, ttl time.Duration) error {
	if rdb == nil {
		return nil
	}
	if ttl <= 0 {
		ttl = defaultUserRevocationTTL
	}
	return rdb.Set(ctx, "revoked:user:"+userID.String(), "1", ttl).Err()
}

func ClearUserRevocation(ctx context.Context, rdb redis.UniversalClient, userID uuid.UUID) error {
	if rdb == nil {
		return nil
	}
	return rdb.Del(ctx, "revoked:user:"+userID.String()).Err()
}

// RevokeTokenSession marks access token and session revoked on a Redis client.
func RevokeTokenSession(ctx context.Context, rdb redis.UniversalClient, tokenID, sessionID uuid.UUID, ttl time.Duration) error {
	if rdb == nil || ttl <= 0 {
		return nil
	}
	pipe := rdb.Pipeline()
	pipe.Set(ctx, "revoked:token:"+tokenID.String(), "true", ttl)
	pipe.Set(ctx, "revoked:session:"+sessionID.String(), "true", ttl)
	_, err := pipe.Exec(ctx)
	return err
}

// RevokeTokenSessionShards marks revocation on every control-plane Redis shard.
func RevokeTokenSessionShards(ctx context.Context, rdbs []redis.UniversalClient, tokenID, sessionID uuid.UUID, ttl time.Duration) error {
	if len(rdbs) == 0 || ttl <= 0 {
		return nil
	}
	for i, rdb := range rdbs {
		if rdb == nil {
			continue
		}
		if err := RevokeTokenSession(ctx, rdb, tokenID, sessionID, ttl); err != nil {
			return fmt.Errorf("shard %d: %w", i, err)
		}
	}
	return nil
}

func checkTokenRevocationShards(ctx context.Context, rdbs []redis.UniversalClient, payload *Payload) (bool, error) {
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

func revokeUserAccessShards(ctx context.Context, rdbs []redis.UniversalClient, userID uuid.UUID, ttl time.Duration) error {
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

func clearUserRevocationShards(ctx context.Context, rdbs []redis.UniversalClient, userID uuid.UUID) error {
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
