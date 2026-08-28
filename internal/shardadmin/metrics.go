package shardadmin

import (
	"context"
	"strconv"
	"strings"

	"github.com/redis/go-redis/v9"
)

type ShardMetricsProvider interface {
	GetMetrics(ctx context.Context, shardID int16, redisClient redis.UniversalClient) (ShardMetrics, error)
}

type RealShardMetricsProvider struct{}

func (p *RealShardMetricsProvider) GetMetrics(ctx context.Context, shardID int16, redisClient redis.UniversalClient) (ShardMetrics, error) {
	shardMetrics := ShardMetrics{ShardID: shardID}

	memInfo, err := redisClient.Info(ctx, "memory").Result()
	if err == nil {
		used := parseInfoInt64(memInfo, "used_memory")
		maxmem := parseInfoInt64(memInfo, "maxmemory")
		if maxmem > 0 {
			shardMetrics.MemoryPct = (float64(used) / float64(maxmem)) * 100.0
		} else {
			shardMetrics.MemoryPct = (float64(used) / (1024 * 1024 * 1024)) * 100.0
		}
	}

	statsInfo, err := redisClient.Info(ctx, "stats").Result()
	if err == nil {
		shardMetrics.OpsPerSec = parseInfoInt64(statsInfo, "instantaneous_ops_per_sec")
	}

	cpuInfo, err := redisClient.Info(ctx, "cpu").Result()
	if err == nil {
		sys := parseInfoFloat64(cpuInfo, "used_cpu_sys")
		user := parseInfoFloat64(cpuInfo, "used_cpu_user")
		shardMetrics.CPUUsage = (sys + user) * 10.0
		if shardMetrics.CPUUsage > 100.0 {
			shardMetrics.CPUUsage = 100.0
		}
	}

	return shardMetrics, nil
}

func parseInfoInt64(info, key string) int64 {
	lines := strings.Split(info, "\r\n")
	for _, line := range lines {
		if strings.HasPrefix(line, key+":") {
			parts := strings.Split(line, ":")
			if len(parts) == 2 {
				val, err := strconv.ParseInt(parts[1], 10, 64)
				if err == nil {
					return val
				}
			}
		}
	}
	return 0
}

func parseInfoFloat64(info, key string) float64 {
	lines := strings.Split(info, "\r\n")
	for _, line := range lines {
		if strings.HasPrefix(line, key+":") {
			parts := strings.Split(line, ":")
			if len(parts) == 2 {
				val, err := strconv.ParseFloat(parts[1], 64)
				if err == nil {
					return val
				}
			}
		}
	}
	return 0.0
}
