package controlplane

import (
	"context"
	"fmt"

	"ad-event-processor/pkg/pgfailover"

	"github.com/redis/go-redis/v9"
)

type pgFailoverShardReader struct {
	redisShards []redis.UniversalClient
}

func newPgFailoverShardReader(redisShards []redis.UniversalClient) *pgFailoverShardReader {
	return &pgFailoverShardReader{redisShards: redisShards}
}

func (reader *pgFailoverShardReader) activeDSN(ctx context.Context) (string, uint64, error) {
	var lastErr error
	for i, redisClient := range reader.redisShards {
		if redisClient == nil {
			continue
		}
		dsn, epoch, err := pgfailover.ActiveDSN(ctx, redisClient)
		if err == nil && dsn != "" {
			return dsn, epoch, nil
		}
		if err != nil {
			lastErr = fmt.Errorf("shard %d: %w", i, err)
		}
	}
	if lastErr != nil {
		return "", 0, lastErr
	}
	return "", 0, nil
}
