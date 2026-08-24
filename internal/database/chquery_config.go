package database

import (
	"time"

	"ad-event-processor/internal/config"
)

func CHQueryConfigFromApp(cfg *config.Config) CHQueryConfig {
	if cfg == nil {
		return CHQueryConfig{}
	}
	out := CHQueryConfig{
		MaxConcurrency: cfg.CHQueryMaxConcurrency,
	}
	if cfg.CHQueryTimeoutSec > 0 {
		out.QueryTimeout = time.Duration(cfg.CHQueryTimeoutSec) * time.Second
	}
	if cfg.CHQuerySlowLogMS > 0 {
		out.SlowQueryThreshold = time.Duration(cfg.CHQuerySlowLogMS) * time.Millisecond
	}
	if cfg.CHQueryMaxMemoryBytes > 0 {
		out.MaxMemoryBytes = cfg.CHQueryMaxMemoryBytes
	}
	if cfg.CHQueryMaxExecSec > 0 {
		out.MaxExecutionTimeSec = cfg.CHQueryMaxExecSec
	}
	return out
}
