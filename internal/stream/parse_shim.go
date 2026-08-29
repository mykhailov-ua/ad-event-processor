package stream

import (
	"ad-event-processor/internal/filter"

	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
)

func ParseUUID(b []byte, dst *uuid.UUID) bool {
	return filter.ParseUUID(b, dst)
}

func firstConnectedRedisShard(redisShards []redis.UniversalClient) redis.UniversalClient {
	for _, redisClient := range redisShards {
		if redisClient != nil {
			return redisClient
		}
	}
	return nil
}
