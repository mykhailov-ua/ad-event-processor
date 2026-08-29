package features

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/redis/go-redis/v9"
)

const residentialIntelRedisPrefix = "intel:residential:"

type ResidentialIntelCache struct {
	redisClient redis.Cmdable
	ttl         time.Duration
}

func NewResidentialIntelCache(redisClient redis.Cmdable, ttl time.Duration) *ResidentialIntelCache {
	if ttl <= 0 {
		ttl = 24 * time.Hour
	}
	return &ResidentialIntelCache{redisClient: redisClient, ttl: ttl}
}

func (c *ResidentialIntelCache) Get(ctx context.Context, ip string) (ResidentialIntelResult, bool, error) {
	if c == nil || c.redisClient == nil {
		return ResidentialIntelResult{}, false, nil
	}
	ip = strings.TrimSpace(ip)
	if ip == "" {
		return ResidentialIntelResult{}, false, ErrInvalidIP
	}
	raw, err := c.redisClient.Get(ctx, residentialIntelRedisPrefix+ip).Result()
	if errors.Is(err, redis.Nil) {
		return ResidentialIntelResult{}, false, nil
	}
	if err != nil {
		return ResidentialIntelResult{}, false, fmt.Errorf("redis get residential intel: %w", err)
	}
	var entry ResidentialIntelResult
	if err := json.Unmarshal([]byte(raw), &entry); err != nil {
		return ResidentialIntelResult{}, false, fmt.Errorf("decode residential intel cache: %w", err)
	}
	return entry, true, nil
}

func (c *ResidentialIntelCache) Set(ctx context.Context, ip string, result ResidentialIntelResult) error {
	if c == nil || c.redisClient == nil {
		return nil
	}
	ip = strings.TrimSpace(ip)
	if ip == "" {
		return ErrInvalidIP
	}
	payload, err := json.Marshal(result)
	if err != nil {
		return fmt.Errorf("marshal residential intel cache: %w", err)
	}
	if err := c.redisClient.Set(ctx, residentialIntelRedisPrefix+ip, payload, c.ttl).Err(); err != nil {
		return fmt.Errorf("redis set residential intel: %w", err)
	}
	return nil
}
