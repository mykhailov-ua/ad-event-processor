// In-process generational blacklist cache mirroring L7 access-check.lua / edge-blacklist-sync.lua.
//
// Not used on the BPF hot path. Production L7: ngx.shared blacklist_cache (_bl_ver, b:{ip}).
// Production L4: BlocklistStore + pinned maps via edge-bpf-sync (blocklist_sync.go).
//
// Memory Model Rules (L7 generational semantics):
//
//	SyncFromRedis: SMEMBERS manual|auto|fraud -> bump ver, stamp blocked[ip]=newVer (full invalidation
//	  of prior generation: PerimeterCheck requires ipVer == c.ver).
//	PerimeterCheck: fail-closed stale when ver==0, syncTS==0, or now-syncTS > staleSec (503 analog).
//	Block match: blocked[clientIP] == c.ver -> 403 analog.
//
// Cache invalidation patterns:
//   - l7_generational_full: SyncFromRedis increments ver; old generation entries no longer match.
//   - Unlike BPF path, no explicit delete of stale IP keys; generational mismatch passes.
//   - Unlike ngx incremental quarantine, this Go helper always bumps ver on sync (no bump_version=false).
//   - Production _bl_count gauge: deduped on incremental stamp_ips; see edge-blacklist-sync.lua header.
package edge

import (
	"context"
	"time"

	"github.com/redis/go-redis/v9"
)

const (
	defaultStaleSec = 30
)

type PerimeterOutcome int

const (
	PerimeterPass PerimeterOutcome = iota
	PerimeterBlocked403
	PerimeterStale503
)

type Metrics struct {
	PerimeterPass  int64
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

func (c *BlacklistCache) SyncFromRedis(ctx context.Context, redisClient redis.Cmdable) error {
	manual, err := redisClient.SMembers(ctx, redisKeyBlacklistManual).Result()
	if err != nil {
		return err
	}
	auto, err := redisClient.SMembers(ctx, redisKeyBlacklistAuto).Result()
	if err != nil {
		return err
	}
	fraud, err := redisClient.SMembers(ctx, redisKeyBlacklistFraud).Result()
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

func (c *BlacklistCache) PerimeterCheck(clientIP string, nowUnix int64, m *Metrics) PerimeterOutcome {
	return c.PerimeterCheckASN(clientIP, "", nowUnix, m)
}

func (c *BlacklistCache) PerimeterCheckASN(clientIP, clientASN string, nowUnix int64, m *Metrics) PerimeterOutcome {
	if c.asnWhitelist != nil && c.asnWhitelist.IsWhitelisted(clientASN) {
		if m != nil {
			m.PerimeterPass++
		}
		return PerimeterPass
	}
	if c.ver == 0 || c.syncTS == 0 {
		if m != nil {
			m.BlacklistStale++
		}
		return PerimeterStale503
	}
	if nowUnix-c.syncTS > c.staleSec {
		if m != nil {
			m.BlacklistStale++
		}
		return PerimeterStale503
	}
	if ipVer, ok := c.blocked[clientIP]; ok && ipVer == c.ver {
		if m != nil {
			m.BlockedIP++
		}
		return PerimeterBlocked403
	}
	if m != nil {
		m.PerimeterPass++
	}
	return PerimeterPass
}

func (c *BlacklistCache) SetASNWhitelist(w *ASNWhitelist) {
	c.asnWhitelist = w
}

func (c *BlacklistCache) Version() int64 { return c.ver }

func (c *BlacklistCache) SyncTimestamp() int64 { return c.syncTS }
