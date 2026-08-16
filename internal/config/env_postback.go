package config

import (
	"os"
	"time"
)

func loadPostbackModules(cfg *Config) {
	if cfg == nil {
		return
	}
	cfg.Postback.PollIntervalMs = getEnvInt("POSTBACK_POLL_INTERVAL_MS", 5000)
	cfg.Postback.BatchSize = getEnvInt("POSTBACK_BATCH_SIZE", 50)
	cfg.Postback.StaleProcessingSec = getEnvInt("POSTBACK_STALE_PROCESSING_SEC", 120)
	cfg.Postback.MetricsAddr = os.Getenv("POSTBACK_METRICS_ADDR")
}

func (c *Config) PostbackPollInterval() time.Duration {
	if c == nil || c.Postback.PollIntervalMs <= 0 {
		return 5 * time.Second
	}
	return time.Duration(c.Postback.PollIntervalMs) * time.Millisecond
}

func (c *Config) PostbackBatchSize() int32 {
	if c == nil || c.Postback.BatchSize <= 0 {
		return 50
	}
	return int32(c.Postback.BatchSize)
}
