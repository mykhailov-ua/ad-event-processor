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

func CheckTokenRevocation(ctx context.Context, redisClient redis.UniversalClient, payload *Payload) (revoked bool, err error) {
	if redisClient == nil || payload == nil {
		return false, nil
	}

	ctxRevoked, cancel := context.WithTimeout(ctx, revocationCheckTimeout)
	defer cancel()

	cmds, errPipe := redisClient.Pipelined(ctxRevoked, func(pipe redis.Pipeliner) error {
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

func RevokeUserAccess(ctx context.Context, redisClient redis.UniversalClient, userID uuid.UUID, ttl time.Duration) error {
	if redisClient == nil {
		return nil
	}
	if ttl <= 0 {
		ttl = defaultUserRevocationTTL
	}
	return redisClient.Set(ctx, "revoked:user:"+userID.String(), "1", ttl).Err()
}

func ClearUserRevocation(ctx context.Context, redisClient redis.UniversalClient, userID uuid.UUID) error {
	if redisClient == nil {
		return nil
	}
	return redisClient.Del(ctx, "revoked:user:"+userID.String()).Err()
}

func RevokeTokenSession(ctx context.Context, redisClient redis.UniversalClient, tokenID, sessionID uuid.UUID, ttl time.Duration) error {
	if redisClient == nil || ttl <= 0 {
		return nil
	}
	pipe := redisClient.Pipeline()
	pipe.Set(ctx, "revoked:token:"+tokenID.String(), "true", ttl)
	pipe.Set(ctx, "revoked:session:"+sessionID.String(), "true", ttl)
	_, err := pipe.Exec(ctx)
	return err
}

func RevokeTokenSessionShards(ctx context.Context, redisShards []redis.UniversalClient, tokenID, sessionID uuid.UUID, ttl time.Duration) error {
	if len(redisShards) == 0 || ttl <= 0 {
		return nil
	}
	for i, redisClient := range redisShards {
		if redisClient == nil {
			continue
		}
		if err := RevokeTokenSession(ctx, redisClient, tokenID, sessionID, ttl); err != nil {
			return fmt.Errorf("shard %d: %w", i, err)
		}
	}
	return nil
}

func checkTokenRevocationShards(ctx context.Context, redisShards []redis.UniversalClient, payload *Payload) (bool, error) {
	if payload == nil || len(redisShards) == 0 {
		return false, nil
	}
	ctxRevoked, cancel := context.WithTimeout(ctx, revocationCheckTimeout)
	defer cancel()
	for i, redisClient := range redisShards {
		if redisClient == nil {
			continue
		}
		revoked, err := CheckTokenRevocation(ctxRevoked, redisClient, payload)
		if err != nil {
			return false, fmt.Errorf("shard %d: %w", i, err)
		}
		if revoked {
			return true, nil
		}
	}
	return false, nil
}

func revokeUserAccessShards(ctx context.Context, redisShards []redis.UniversalClient, userID uuid.UUID, ttl time.Duration) error {
	if len(redisShards) == 0 {
		return nil
	}
	for i, redisClient := range redisShards {
		if redisClient == nil {
			continue
		}
		if err := RevokeUserAccess(ctx, redisClient, userID, ttl); err != nil {
			return fmt.Errorf("shard %d: %w", i, err)
		}
	}
	return nil
}

func clearUserRevocationShards(ctx context.Context, redisShards []redis.UniversalClient, userID uuid.UUID) error {
	if len(redisShards) == 0 {
		return nil
	}
	for i, redisClient := range redisShards {
		if redisClient == nil {
			continue
		}
		if err := ClearUserRevocation(ctx, redisClient, userID); err != nil {
			return fmt.Errorf("shard %d: %w", i, err)
		}
	}
	return nil
}
