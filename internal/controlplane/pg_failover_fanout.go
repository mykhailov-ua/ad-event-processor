package controlplane

import (
	"context"
	"fmt"

	"ad-event-processor/pkg/pgfailover"

	"github.com/redis/go-redis/v9"
)

type pgFailoverShardReader struct {
	rdbs []redis.UniversalClient
}

func newPgFailoverShardReader(rdbs []redis.UniversalClient) *pgFailoverShardReader {
	return &pgFailoverShardReader{rdbs: rdbs}
}

func (reader *pgFailoverShardReader) activeDSN(ctx context.Context) (string, uint64, error) {
	var lastErr error
	for i, rdb := range reader.rdbs {
		if rdb == nil {
			continue
		}
		dsn, epoch, err := pgfailover.ActiveDSN(ctx, rdb)
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
