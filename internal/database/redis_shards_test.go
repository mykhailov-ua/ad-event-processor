package database

import (
	"testing"

	"ad-event-processor/internal/config"
)

func TestShardUniversalOptions_direct(t *testing.T) {
	cfg := &config.Config{
		RedisAddrs:    []string{"127.0.0.1:6479", "127.0.0.1:6480"},
		RedisPassword: "secret",
	}
	opts := shardUniversalOptions(cfg, 1, cfg.ResolveRedisMasterNames(), RedisShardOptions{PoolSize: 8, FilterTimeoutMs: 12})
	if opts.MasterName != "" {
		t.Fatalf("direct mode must not set MasterName, got %q", opts.MasterName)
	}
	if len(opts.Addrs) != 1 || opts.Addrs[0] != "127.0.0.1:6480" {
		t.Fatalf("addrs=%v", opts.Addrs)
	}
	if opts.PoolSize != 8 {
		t.Fatalf("pool=%d", opts.PoolSize)
	}
	if opts.MaxActiveConns != 8 {
		t.Fatalf("max_active=%d", opts.MaxActiveConns)
	}
	if opts.ReadTimeout.Milliseconds() != 12 || opts.WriteTimeout.Milliseconds() != 12 {
		t.Fatalf("timeouts read=%s write=%s", opts.ReadTimeout, opts.WriteTimeout)
	}
}

func TestShardUniversalOptions_sentinel(t *testing.T) {
	cfg := &config.Config{
		RedisAddrs:         []string{"127.0.0.1:6479", "127.0.0.1:6480"},
		RedisSentinelAddrs: []string{"127.0.0.1:26379", "127.0.0.1:26380"},
		RedisMasterNames:   []string{"ad-event-processor-shard-0", "ad-event-processor-shard-1"},
		RedisPassword:      "secret",
	}
	opts := shardUniversalOptions(cfg, 0, cfg.RedisMasterNames, RedisShardOptions{})
	if opts.MasterName != "ad-event-processor-shard-0" {
		t.Fatalf("master=%q", opts.MasterName)
	}
	if len(opts.Addrs) != 2 {
		t.Fatalf("sentinel addrs=%v", opts.Addrs)
	}
}

func TestShardUniversalOptions_stickyPinReserve(t *testing.T) {
	cfg := &config.Config{
		RedisAddrs:    []string{"127.0.0.1:6479"},
		RedisPassword: "secret",
	}
	opts := shardUniversalOptions(cfg, 0, cfg.ResolveRedisMasterNames(), RedisShardOptions{
		PoolSize:         8,
		StickyPinWorkers: 16,
	})
	if opts.PoolSize != 24 {
		t.Fatalf("pool=%d want 24", opts.PoolSize)
	}
	if opts.MaxActiveConns != 24 {
		t.Fatalf("max_active=%d want 24", opts.MaxActiveConns)
	}
}

func TestShardUniversalOptions_UnixDomainSocket(t *testing.T) {
	tests := []struct {
		name       string
		addr       string
		wantDialer bool
	}{
		{
			name:       "TCP IPv4",
			addr:       "127.0.0.1:6379",
			wantDialer: false,
		},
		{
			name:       "Unix socket path",
			addr:       "/var/run/redis/redis-shard0.sock",
			wantDialer: true,
		},
		{
			name:       "Unix socket relative filename",
			addr:       "redis.sock",
			wantDialer: true,
		},
		{
			name:       "Unix socket in temp directory",
			addr:       "/tmp/redis_test.sock",
			wantDialer: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := &config.Config{
				RedisAddrs: []string{tt.addr},
			}
			opts := shardUniversalOptions(cfg, 0, nil, RedisShardOptions{})
			if (opts.Dialer != nil) != tt.wantDialer {
				t.Fatalf("opts.Dialer != nil is %v; want %v", opts.Dialer != nil, tt.wantDialer)
			}
			if len(opts.Addrs) != 1 || opts.Addrs[0] != tt.addr {
				t.Fatalf("opts.Addrs = %v; want [%s]", opts.Addrs, tt.addr)
			}
		})
	}
}

func BenchmarkShardUniversalOptions_UDSParsing(b *testing.B) {
	cfgUDS := &config.Config{
		RedisAddrs: []string{"/var/run/redis/redis-shard0.sock"},
	}
	cfgTCP := &config.Config{
		RedisAddrs: []string{"127.0.0.1:6379"},
	}

	b.Run("UDS_Network", func(b *testing.B) {
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			_ = shardUniversalOptions(cfgUDS, 0, nil, RedisShardOptions{})
		}
	})

	b.Run("TCP_Network", func(b *testing.B) {
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			_ = shardUniversalOptions(cfgTCP, 0, nil, RedisShardOptions{})
		}
	})
}

func TestShardUniversalOptions_MaxActiveConnsBacklogSync(t *testing.T) {
	cfg := &config.Config{
		RedisAddrs:          []string{"127.0.0.1:6379"},
		RedisMaxActiveConns: 2048,
	}
	opts := shardUniversalOptions(cfg, 0, nil, RedisShardOptions{PoolSize: 64})
	if opts.PoolSize != 64 {
		t.Fatalf("opts.PoolSize = %d; want 64", opts.PoolSize)
	}
	if opts.MaxActiveConns != 2048 {
		t.Fatalf("opts.MaxActiveConns = %d; want 2048", opts.MaxActiveConns)
	}
}
