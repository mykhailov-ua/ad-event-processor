package database

import (
	"context"
	"fmt"
	"strconv"
	"sync"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

const (
	defaultPostgresConnectTimeout  = 5 * time.Second
	defaultPostgresMaxConnLifetime = time.Hour
)

// PoolConfig optional pgxpool tuning applied in Connect.
type PoolConfig struct {
	StatementTimeout time.Duration
	ConnectTimeout   time.Duration
	MaxConnLifetime  time.Duration
}

// Connect warms minConns with parallel Ping so first /track burst does not pay pgx dial latency.
func Connect(ctx context.Context, dsn string, maxConns, minConns int, poolCfg ...PoolConfig) (*pgxpool.Pool, error) {
	config, err := pgxpool.ParseConfig(dsn)
	if err != nil {
		return nil, fmt.Errorf("failed to parse config: %w", err)
	}

	var opts PoolConfig
	if len(poolCfg) > 0 {
		opts = poolCfg[0]
	}
	applyPoolConfig(config, opts)

	config.MaxConns = int32(maxConns)
	config.MinConns = int32(minConns)
	config.MaxConnIdleTime = 30 * time.Minute

	pool, err := pgxpool.NewWithConfig(ctx, config)
	if err != nil {
		return nil, fmt.Errorf("failed to create pool: %w", err)
	}

	if err := pool.Ping(ctx); err != nil {
		pool.Close()
		return nil, fmt.Errorf("failed to ping db: %w", err)
	}

	if minConns > 1 {
		var wg sync.WaitGroup
		for i := 1; i < minConns; i++ {
			wg.Add(1)
			go func() {
				defer wg.Done()
				_ = pool.Ping(ctx)
			}()
		}
		wg.Wait()
	}

	return pool, nil
}

func applyPoolConfig(config *pgxpool.Config, opts PoolConfig) {
	if config == nil {
		return
	}
	connectTimeout := opts.ConnectTimeout
	if connectTimeout <= 0 {
		connectTimeout = defaultPostgresConnectTimeout
	}
	config.ConnConfig.ConnectTimeout = connectTimeout

	// Avoid stale TCP behind L4 idle timeout (mirror clickhouse_connect.go ConnMaxLifetime).
	maxLifetime := opts.MaxConnLifetime
	if maxLifetime <= 0 {
		maxLifetime = defaultPostgresMaxConnLifetime
	}
	config.MaxConnLifetime = maxLifetime

	applyPoolRuntimeParams(config, opts)
}

func applyPoolRuntimeParams(config *pgxpool.Config, opts PoolConfig) {
	if config == nil || opts.StatementTimeout <= 0 {
		return
	}
	ms := opts.StatementTimeout.Milliseconds()
	if ms < 1 {
		ms = 1
	}
	if config.ConnConfig.RuntimeParams == nil {
		config.ConnConfig.RuntimeParams = make(map[string]string)
	}
	config.ConnConfig.RuntimeParams["statement_timeout"] = strconv.FormatInt(ms, 10)
}
