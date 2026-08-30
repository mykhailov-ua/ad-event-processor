package ingest

import (
	"os"
	"strconv"
	"time"
)

type parserSlowBodyDrillConfig struct {
	Duration    time.Duration
	Connections int
	P99Budget   time.Duration
}

type parserSlowBodyDrillResult struct {
	P99Nanos    int64
	ControlReqs int64
}

func slowBodyDrillConfigFromEnv() parserSlowBodyDrillConfig {
	cfg := parserSlowBodyDrillConfig{
		Duration:    30 * time.Second,
		Connections: 256,
		P99Budget:   80 * time.Millisecond,
	}
	if v := os.Getenv("PARSER_SLOW_BODY_DRILL_DURATION"); v != "" {
		if d, err := time.ParseDuration(v); err == nil {
			cfg.Duration = d
		}
	}
	if v := os.Getenv("PARSER_SLOW_BODY_DRILL_CONNECTIONS"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 {
			cfg.Connections = n
		}
	}
	if v := os.Getenv("PARSER_SLOW_BODY_DRILL_P99_BUDGET_MS"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 {
			cfg.P99Budget = time.Duration(n) * time.Millisecond
		}
	}
	return cfg
}

func runParserSlowBodyDrill(cfg parserSlowBodyDrillConfig) parserSlowBodyDrillResult {
	_ = cfg
	return parserSlowBodyDrillResult{}
}
