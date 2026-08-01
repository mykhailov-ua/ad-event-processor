package config

import (
	"errors"
	"fmt"
	"os"
	"strconv"
)

func validateAndApplyDefaults(cfg *Config) error {
	if cfg.ServerPort == "" {
		return errors.New("SERVER_PORT is required")
	}
	if cfg.ProcessorPort == "" {
		cfg.ProcessorPort = "8186"
	}
	if cfg.ManagementPort == "" {
		cfg.ManagementPort = "8188"
	}
	if cfg.MetricsPort == "" {
		cfg.MetricsPort = "9090"
	}
	if cfg.DBDSN == "" {
		return errors.New("DB_DSN is required")
	}
	if cfg.PaymentDBDSN == "" {
		cfg.PaymentDBDSN = cfg.DBDSN
	}
	if len(cfg.RedisAddrs) == 0 {
		return errors.New("REDIS_ADDRS is required")
	}
	if cfg.Env == "production" && len(cfg.RedisAddrs) != ExpectedRedisShardCount {
		return fmt.Errorf("production requires exactly %d Redis shards (REDIS_ADDRS), got %d", ExpectedRedisShardCount, len(cfg.RedisAddrs))
	}
	if cfg.RedisSentinelEnabled() {
		if len(cfg.RedisMasterNames) > 0 && len(cfg.RedisMasterNames) != len(cfg.RedisAddrs) {
			return fmt.Errorf("REDIS_MASTER_NAMES count (%d) must match REDIS_ADDRS (%d)", len(cfg.RedisMasterNames), len(cfg.RedisAddrs))
		}
	}

	if cfg.RedisStreamName == "" {
		cfg.RedisStreamName = "ad:events:stream"
	}
	if cfg.FraudStreamName == "" {
		cfg.FraudStreamName = "ad:fraud:stream"
	}
	if cfg.RedisGroupName == "" {
		cfg.RedisGroupName = "ad:processor:group"
	}
	if cfg.RedisConsumerID == "" {
		hostname, _ := os.Hostname()
		if hostname == "" {
			hostname = "unknown"
		}
		cfg.RedisConsumerID = hostname + ":" + strconv.Itoa(os.Getpid())
	}

	applyControlplaneDefaults(cfg)
	if cfg.Env == "" {
		cfg.Env = "development"
	}
	if cfg.TokenSymmetricKey == "" {
		return errors.New("TOKEN_SYMMETRIC_KEY is required")
	}

	if cfg.FilterTimeoutMs <= 0 {
		cfg.FilterTimeoutMs = cfg.WriteTimeoutMs
	}
	if cfg.Env == "production" && cfg.FilterTimeoutMs > 100 {
		return fmt.Errorf("production FILTER_TIMEOUT_MS must be <= 100 (got %d)", cfg.FilterTimeoutMs)
	}
	if cfg.Env == "production" && cfg.TrackerPGFallback {
		return fmt.Errorf("production TRACKER_PG_FALLBACK must be 0")
	}

	return nil
}
