package config

import (
	"errors"
	"fmt"
	"os"
	"strconv"
	"strings"
)

func validateAndApplyDefaults(cfg *Config) error {
	if cfg.ServerPort == "" && cfg.TrackerUnixSocket == "" {
		return errors.New("SERVER_PORT is required when TRACKER_UNIX_SOCKET is unset")
	}
	if cfg.ServerPort == "" {
		cfg.ServerPort = "8181"
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

	if cfg.BrokerPrimaryCH() && strings.TrimSpace(cfg.Broker.URL) == "" {
		return errors.New("CH_INGEST_SOURCE=broker requires BROKER_URL")
	}

	if cfg.Postback.PollIntervalMs <= 0 {
		cfg.Postback.PollIntervalMs = 5000
	}
	if cfg.Postback.BatchSize <= 0 {
		cfg.Postback.BatchSize = 50
	}
	if cfg.Postback.StaleProcessingSec <= 0 {
		cfg.Postback.StaleProcessingSec = 120
	}

	if cfg.IVT.ScanIntervalMs <= 0 {
		cfg.IVT.ScanIntervalMs = 60000
	}

	if cfg.FraudScoring.ScanIntervalMs <= 0 {
		cfg.FraudScoring.ScanIntervalMs = 60000
	}
	if cfg.FraudScoring.BatchSize <= 0 {
		cfg.FraudScoring.BatchSize = 1000
	}
	if cfg.FraudScoring.MicrobatchFlushMs <= 0 {
		cfg.FraudScoring.MicrobatchFlushMs = 50
	}
	if cfg.FraudScoring.MicrobatchMaxLagSec <= 0 {
		cfg.FraudScoring.MicrobatchMaxLagSec = 30
	}
	if cfg.FraudScoring.BoostFullResyncSec <= 0 {
		cfg.FraudScoring.BoostFullResyncSec = 10
	}
	if cfg.FraudConsumerLagSec <= 0 {
		cfg.FraudConsumerLagSec = 60
	}

	if cfg.Billing.ExportFetchRows <= 0 {
		cfg.Billing.ExportFetchRows = 1000
	}
	if cfg.Billing.ExportJobTimeoutMin <= 0 {
		cfg.Billing.ExportJobTimeoutMin = 15
	}

	return nil
}
