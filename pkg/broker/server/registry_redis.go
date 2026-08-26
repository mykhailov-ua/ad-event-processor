package server

import (
	"context"
	"fmt"
	"strconv"
	"time"

	"ad-event-processor/pkg/broker/protocol"

	"github.com/redis/go-redis/v9"
)

const (
	redisTopicsHashKey = "ad_event_processor:broker:topics"
	redisTopicsNextKey = "ad_event_processor:broker:topics:next_id"
)

var registerTopicScript = redis.NewScript(`
local existing = redis.call('HGET', KEYS[1], ARGV[1])
if existing then
 return existing
end
local id = redis.call('INCR', KEYS[2])
redis.call('HSET', KEYS[1], ARGV[1], id)
return id
`)

type RedisTopicStore struct {
	redisClient redis.UniversalClient
}

func NewRedisTopicStore(redisClient redis.UniversalClient) *RedisTopicStore {
	return &RedisTopicStore{redisClient: redisClient}
}

func (s *RedisTopicStore) Load(ctx context.Context) (protocol.RegistrySnapshot, error) {
	snap := protocol.RegistrySnapshot{
		Version: 1,
		Topics:  make(map[string]uint16),
	}
	if s == nil || s.redisClient == nil {
		return snap, nil
	}

	raw, err := s.redisClient.HGetAll(ctx, redisTopicsHashKey).Result()
	if err != nil {
		return snap, err
	}
	var maxID uint16
	for name, idStr := range raw {
		id64, convErr := strconv.ParseUint(idStr, 10, 16)
		if convErr != nil || id64 == 0 {
			continue
		}
		id := uint16(id64)
		snap.Topics[name] = id
		if id > maxID {
			maxID = id
		}
	}

	nextStr, err := s.redisClient.Get(ctx, redisTopicsNextKey).Result()
	if err == nil {
		if next64, convErr := strconv.ParseUint(nextStr, 10, 32); convErr == nil {
			snap.NextID = uint32(next64)
		}
	}
	if snap.NextID <= uint32(maxID) {
		snap.NextID = uint32(maxID) + 1
	}
	return snap, nil
}

func (s *RedisTopicStore) Register(ctx context.Context, name string) (uint16, error) {
	if s == nil || s.redisClient == nil {
		return 0, fmt.Errorf("redis topic store is not configured")
	}
	if err := protocol.ValidateTopicNameForStore(name); err != nil {
		return 0, err
	}

	res, err := registerTopicScript.Run(ctx, s.redisClient, []string{redisTopicsHashKey, redisTopicsNextKey}, name).Result()
	if err != nil {
		return 0, err
	}
	switch v := res.(type) {
	case int64:
		if v <= 0 || v > 65535 {
			return 0, fmt.Errorf("invalid topic id from redis: %d", v)
		}
		return uint16(v), nil
	case string:
		id64, convErr := strconv.ParseUint(v, 10, 16)
		if convErr != nil || id64 == 0 {
			return 0, fmt.Errorf("invalid topic id from redis: %q", v)
		}
		return uint16(id64), nil
	default:
		return 0, fmt.Errorf("unexpected redis topic id type %T", res)
	}
}

func MergeTimeout() time.Duration {
	return 3 * time.Second
}
