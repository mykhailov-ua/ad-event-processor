package config

import (
	"fmt"
	"os"
	"strconv"
)

type Config struct {
	ServerPort       string
	DBDSN            string
	RedisAddr        string
	EventBatchSize   int
	EventFlushMs     int
	StatsFlushMs     int
	MaxWorkers       int
	LogRetentionDays int
}

func getEnvInt(key string, fallback int) int {
	if value, ok := os.LookupEnv(key); ok {
		if intVal, err := strconv.Atoi(value); err == nil {
			return intVal
		}
	}
	return fallback
}

func Load() (*Config, error) {
	cfg := &Config{
		ServerPort:       os.Getenv("SERVER_PORT"),
		DBDSN:            os.Getenv("DB_DSN"),
		RedisAddr:        os.Getenv("REDIS_ADDR"),
		EventBatchSize:   getEnvInt("EVENT_BATCH_SIZE", 1000),
		EventFlushMs:     getEnvInt("EVENT_FLUSH_MS", 500),
		StatsFlushMs:     getEnvInt("STATS_FLUSH_MS", 5000),
		MaxWorkers:       getEnvInt("MAX_WORKERS", 10),
		LogRetentionDays: getEnvInt("LOG_RETENTION_DAYS", 7),
	}

	if cfg.ServerPort == "" {
		return nil, fmt.Errorf("SERVER_PORT is required")
	}
	if cfg.DBDSN == "" {
		return nil, fmt.Errorf("DB_DSN is required")
	}
	if cfg.RedisAddr == "" {
		return nil, fmt.Errorf("REDIS_ADDR is required")
	}

	return cfg, nil
}
