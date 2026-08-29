package conn

import (
	"bufio"
	"context"
	"encoding/json"
	"log/slog"
	"net/netip"
	"os"
	"path/filepath"
	"strings"
	"sync/atomic"
	"time"

	"ad-event-processor/internal/config"
	"ad-event-processor/internal/filter"
	"ad-event-processor/internal/metrics"

	"github.com/redis/go-redis/v9"
)

const (
	residentialIntelFeedFileName = "external_residential.txt"
	residentialIntelRedisPrefix  = "intel:residential:"
)

type residentialIntelRedisEntry struct {
	ResidentialProxy bool `json:"residential_proxy"`
	VPN              bool `json:"vpn"`
	Proxy            bool `json:"proxy"`
}

func (e residentialIntelRedisEntry) isResidentialFarm() bool {
	return e.ResidentialProxy || (e.Proxy && e.VPN)
}

type residentialIntelFeedLoader struct {
	dir     string
	refresh time.Duration
	table   *filter.ResidentialIntelTable
	redis   redis.Cmdable
	gen     atomic.Uint64
}

type ResidentialIntelFeedLoader struct {
	loader *residentialIntelFeedLoader
}

func NewResidentialIntelFeedLoader(cfg *config.Config, table *filter.ResidentialIntelTable, redisClient redis.Cmdable) *ResidentialIntelFeedLoader {
	if cfg == nil || !cfg.ResidentialIntelHotReadEnabled || table == nil {
		return nil
	}
	dir := cfg.ResidentialIntelFeedDir
	if dir == "" {
		dir = cfg.ProxyVPNFeedDir
	}
	if dir == "" {
		dir = "/var/lib/ad-event-processor/proxy-vpn"
	}
	refresh := cfg.ResidentialIntelFeedRefresh
	if refresh <= 0 {
		refresh = cfg.ProxyVPNFeedRefresh
	}
	if refresh <= 0 {
		refresh = 24 * time.Hour
	}
	return &ResidentialIntelFeedLoader{loader: &residentialIntelFeedLoader{
		dir:     dir,
		refresh: refresh,
		table:   table,
		redis:   redisClient,
	}}
}

func (l *ResidentialIntelFeedLoader) Start(ctx context.Context) {
	if l == nil || l.loader == nil {
		return
	}
	l.loader.Start(ctx)
}

func (l *residentialIntelFeedLoader) Start(ctx context.Context) {
	if l == nil {
		return
	}
	l.reloadOnce(ctx)
	ticker := time.NewTicker(l.refresh)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			l.reloadOnce(ctx)
		}
	}
}

func (l *residentialIntelFeedLoader) reloadOnce(ctx context.Context) {
	prefixes := l.scanFeedFile(filepath.Join(l.dir, residentialIntelFeedFileName))
	redisPrefixes, redisErr := l.scanRedis(ctx)
	if redisErr != nil {
		metrics.ResidentialIntelFeedRefreshErrorsTotal.Inc()
		slog.Warn("residential intel redis snapshot failed", "error", redisErr)
	}
	prefixes = append(prefixes, redisPrefixes...)
	if len(prefixes) == 0 {
		if !l.table.Ready() {
			metrics.ResidentialIntelLPMUninitialized.Set(1)
			metrics.ResidentialIntelCacheStaleTotal.Inc()
		}
		return
	}
	gen := l.gen.Add(1)
	l.table.PublishPrefixes(prefixes, gen)
	metrics.ResidentialIntelFeedRefreshTotal.Inc()
	metrics.ResidentialIntelLPMPrefixes.Set(float64(len(prefixes)))
	metrics.ResidentialIntelLPMUninitialized.Set(0)
	metrics.ResidentialIntelFeedLastSuccess.Set(float64(time.Now().Unix()))
	slog.Info("residential intel snapshot published", "prefixes", len(prefixes), "gen", gen)
}

func (l *residentialIntelFeedLoader) scanFeedFile(path string) []netip.Prefix {
	f, err := os.Open(path)
	if err != nil {
		if !os.IsNotExist(err) {
			metrics.ResidentialIntelFeedRefreshErrorsTotal.Inc()
			slog.Warn("residential intel feed open failed", "path", path, "error", err)
		}
		return nil
	}
	defer func() { _ = f.Close() }()

	var out []netip.Prefix
	sc := bufio.NewScanner(f)
	for sc.Scan() {
		line := strings.TrimSpace(sc.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		prefix, _, _, ok := filter.ParseProxyVPNFeedLine(line)
		if !ok {
			continue
		}
		out = append(out, prefix)
	}
	if err := sc.Err(); err != nil {
		metrics.ResidentialIntelFeedRefreshErrorsTotal.Inc()
		slog.Warn("residential intel feed scan failed", "error", err)
		return nil
	}
	return out
}

func (l *residentialIntelFeedLoader) scanRedis(ctx context.Context) ([]netip.Prefix, error) {
	if l == nil || l.redis == nil {
		return nil, nil
	}
	var out []netip.Prefix
	iter := l.redis.Scan(ctx, 0, residentialIntelRedisPrefix+"*", 256).Iterator()
	for iter.Next(ctx) {
		key := iter.Val()
		ip := strings.TrimPrefix(key, residentialIntelRedisPrefix)
		if ip == "" || ip == key {
			continue
		}
		raw, err := l.redis.Get(ctx, key).Bytes()
		if err != nil {
			continue
		}
		var entry residentialIntelRedisEntry
		if err := json.Unmarshal(raw, &entry); err != nil {
			continue
		}
		if !entry.isResidentialFarm() {
			continue
		}
		addr, err := netip.ParseAddr(strings.TrimSpace(ip))
		if err != nil || !addr.IsValid() {
			continue
		}
		prefix := netip.PrefixFrom(addr, addr.BitLen()).Masked()
		if prefix.IsValid() {
			out = append(out, prefix)
		}
	}
	if err := iter.Err(); err != nil {
		return out, err
	}
	return out, nil
}
