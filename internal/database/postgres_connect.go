package database

import (
	"context"
	"fmt"
	"strconv"
	"sync"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

// PoolConfig optional pgxpool tuning applied in Connect.
type PoolConfig struct {
	StatementTimeout time.Duration
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
	applyPoolRuntimeParams(config, opts)

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
