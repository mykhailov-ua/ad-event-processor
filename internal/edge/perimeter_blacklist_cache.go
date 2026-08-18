package edge

import (
	"context"
	"time"

	"github.com/redis/go-redis/v9"
)

const (
	defaultStaleSec = 30
)

type Phase1Outcome int

const (
	Phase1Pass Phase1Outcome = iota
	Phase1Blocked403
	Phase1Stale503
)

type Metrics struct {
	Phase1Pass     int64
	BlockedIP      int64
	BodyRead       int64
	BlacklistStale int64
}

type BlacklistCache struct {
	ver          int64
	syncTS       int64
	count        int64
	staleSec     int64
	blocked      map[string]int64
	asnWhitelist *ASNWhitelist
}

func NewBlacklistCache(staleSec int64) *BlacklistCache {
	if staleSec <= 0 {
		staleSec = defaultStaleSec
	}
	return &BlacklistCache{
		staleSec: staleSec,
		blocked:  make(map[string]int64),
	}
}

func (c *BlacklistCache) SyncFromRedis(ctx context.Context, rdb redis.Cmdable) error {
	manual, err := rdb.SMembers(ctx, redisKeyBlacklistManual).Result()
	if err != nil {
		return err
	}
	auto, err := rdb.SMembers(ctx, redisKeyBlacklistAuto).Result()
	if err != nil {
		return err
	}
	fraud, err := rdb.SMembers(ctx, redisKeyBlacklistFraud).Result()
	if err != nil {
		return err
	}

	newVer := c.ver + 1
	seen := make(map[string]struct{}, len(manual)+len(auto)+len(fraud))
	count := int64(0)

	stamp := func(ip string) {
		if ip == "" {
			return
		}
		if _, ok := seen[ip]; ok {
			return
		}
		seen[ip] = struct{}{}
		c.blocked[ip] = newVer
		count++
	}

	for _, ip := range manual {
		stamp(ip)
	}
	for _, ip := range auto {
		stamp(ip)
	}
	for _, ip := range fraud {
		stamp(ip)
	}

	c.ver = newVer
	c.syncTS = time.Now().Unix()
	c.count = count
	return nil
}

func (c *BlacklistCache) Phase1Check(clientIP string, nowUnix int64, m *Metrics) Phase1Outcome {
	return c.Phase1CheckASN(clientIP, "", nowUnix, m)
}

func (c *BlacklistCache) Phase1CheckASN(clientIP, clientASN string, nowUnix int64, m *Metrics) Phase1Outcome {
	if c.asnWhitelist != nil && c.asnWhitelist.IsWhitelisted(clientASN) {
		if m != nil {
			m.Phase1Pass++
		}
		return Phase1Pass
	}
	if c.ver == 0 || c.syncTS == 0 {
		if m != nil {
			m.BlacklistStale++
		}
		return Phase1Stale503
	}
	if nowUnix-c.syncTS > c.staleSec {
		if m != nil {
			m.BlacklistStale++
		}
		return Phase1Stale503
	}
	if ipVer, ok := c.blocked[clientIP]; ok && ipVer == c.ver {
		if m != nil {
			m.BlockedIP++
		}
		return Phase1Blocked403
	}
	if m != nil {
		m.Phase1Pass++
	}
	return Phase1Pass
}

func (c *BlacklistCache) SetASNWhitelist(w *ASNWhitelist) {
	c.asnWhitelist = w
}

func (c *BlacklistCache) Version() int64 { return c.ver }

func (c *BlacklistCache) SyncTimestamp() int64 { return c.syncTS }
