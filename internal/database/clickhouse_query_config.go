package database

import (
	"time"

	"ad-event-processor/internal/config"
)

func ClickHouseQueryConfigFromApp(cfg *config.Config) ClickHouseQueryConfig {
	if cfg == nil {
		return ClickHouseQueryConfig{}
	}
	out := ClickHouseQueryConfig{
		MaxConcurrency: cfg.ClickHouseQueryMaxConcurrency,
	}
	if cfg.ClickHouseQueryTimeoutSec > 0 {
		out.QueryTimeout = time.Duration(cfg.ClickHouseQueryTimeoutSec) * time.Second
	}
	if cfg.ClickHouseQuerySlowLogMS > 0 {
		out.SlowQueryThreshold = time.Duration(cfg.ClickHouseQuerySlowLogMS) * time.Millisecond
	}
	if cfg.ClickHouseQueryMaxMemoryBytes > 0 {
		out.MaxMemoryBytes = cfg.ClickHouseQueryMaxMemoryBytes
	}
	if cfg.ClickHouseQueryMaxExecSec > 0 {
		out.MaxExecutionTimeSec = cfg.ClickHouseQueryMaxExecSec
	}
	return out
}
