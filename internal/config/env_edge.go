package config

import (
	"os"
	"strings"
)

func loadEdgeModules(cfg *Config) {
	cfg.ElasticShardingEnabled = getEnvBool("ELASTIC_SHARDING_ENABLED", false)
	cfg.ShardOrchestratorEnabled = getEnvBool("SHARD_ORCHESTRATOR_ENABLED", false)
	cfg.ShardOrchestratorIntervalMs = getEnvInt("SHARD_ORCHESTRATOR_INTERVAL_MS", 10000)
	cfg.TCPControlEnabled = getEnvBool("TCP_CONTROL_ENABLED", false)
	cfg.TCPControlHMACSecret = Secret(os.Getenv("TCP_CONTROL_HMAC_SECRET"))
	cfg.TCPControlBindAddr = os.Getenv("TCP_CONTROL_BIND_ADDR")
	if cfg.TCPControlBindAddr == "" {
		cfg.TCPControlBindAddr = os.Getenv("TCP_MGMT_BIND_ADDR")
	}
	if cfg.TCPControlBindAddr == "" {
		cfg.TCPControlBindAddr = ":8192"
	}
	cfg.TCPControlAddr = os.Getenv("TCP_CONTROL_ADDR")
	if cfg.TCPControlAddr == "" {
		cfg.TCPControlAddr = os.Getenv("TCP_MGMT_ADDR")
	}
	if cfg.TCPControlAddr == "" {
		cfg.TCPControlAddr = "127.0.0.1:8192"
	}
	if addrs := os.Getenv("TCP_TRACKER_ADDRS"); addrs != "" {
		cfg.TCPTrackerAddrs = strings.Split(addrs, ",")
	}
	cfg.LuaFastPathEnabled = getEnvBool("LUA_FAST_PATH_ENABLED", true)
	cfg.UDPControlEnabled = getEnvBool("UDP_CONTROL_ENABLED", false)
	cfg.UDPFailClosed = getEnvBool("UDP_FAIL_CLOSED", true)
	cfg.UDPControlBindAddr = os.Getenv("UDP_CONTROL_BIND_ADDR")
	if cfg.UDPControlBindAddr == "" {
		cfg.UDPControlBindAddr = os.Getenv("UDP_MGMT_BIND_ADDR")
	}
	if cfg.UDPControlBindAddr == "" {
		cfg.UDPControlBindAddr = ":8190"
	}
	cfg.UDPTrackerBindAddr = os.Getenv("UDP_TRACKER_BIND_ADDR")
	if cfg.UDPTrackerBindAddr == "" {
		cfg.UDPTrackerBindAddr = ":8191"
	}
	cfg.UDPControlAddr = os.Getenv("UDP_CONTROL_ADDR")
	if cfg.UDPControlAddr == "" {
		cfg.UDPControlAddr = os.Getenv("UDP_MGMT_ADDR")
	}
	if cfg.UDPControlAddr == "" {
		cfg.UDPControlAddr = "127.0.0.1:8190"
	}
	if addrs := os.Getenv("UDP_TRACKER_ADDRS"); addrs != "" {
		cfg.UDPTrackerAddrs = strings.Split(addrs, ",")
	}
	cfg.UDPTrackerID = uint32(getEnvInt("UDP_TRACKER_ID", 1))
	cfg.UDPSyncIntervalMs = getEnvInt("UDP_SYNC_INTERVAL_MS", 10000)
	cfg.UDPDefaultShardRPS = uint64(getEnvInt64("UDP_DEFAULT_SHARD_RPS", 50_000))
	cfg.RegionCode = uint8(RegionCodeFromEnv())
	cfg.MultiRegionEnabled = getEnvBool("MULTI_REGION_ENABLED", false)
	cfg.NodeID = os.Getenv("NODE_ID")
	cfg.NodeRole = os.Getenv("NODE_ROLE")
	cfg.NodeScoreWindowMin = getEnvInt("NODE_SCORE_WINDOW_MIN", 15)
	cfg.NodeScoreMinSamples = getEnvInt("NODE_SCORE_MIN_SAMPLES", 30)
	cfg.NodeWarmupSec = getEnvInt("NODE_WARMUP_SEC", 300)
	cfg.ScoringWeightsJSON = os.Getenv("SCORING_WEIGHTS_JSON")
	cfg.OpLeaseTimeoutSec = getEnvInt("OP_LEASE_TIMEOUT_SEC", 30)
	cfg.OpLeaseMaxRenewals = getEnvInt("OP_LEASE_MAX_RENEWALS", 3)
	cfg.OpLeaseFencingDir = os.Getenv("OP_LEASE_FENCING_DIR")
	cfg.GlobalSpendBatchMin = getEnvInt("GLOBAL_SPEND_BATCH_MIN", 100)
	cfg.GlobalSpendFlushIntervalMs = getEnvInt("GLOBAL_SPEND_FLUSH_INTERVAL_MS", 500)
	cfg.GlobalSpendMaxConcurrency = getEnvInt("GLOBAL_SPEND_MAX_CONCURRENCY", 8)
	cfg.RegionProxyAddr = os.Getenv("REGION_PROXY_ADDR")
	if cfg.RegionProxyRedisURL == "" {
		cfg.RegionProxyRedisURL = os.Getenv("REGION_PROXY_REDIS_URL")
	}
	if cfg.RegionProxyRedisURL == "" && len(cfg.RedisAddrs) > 0 {
		cfg.RegionProxyRedisURL = "redis://" + cfg.RedisAddrs[0] + "/0"
	}
}
