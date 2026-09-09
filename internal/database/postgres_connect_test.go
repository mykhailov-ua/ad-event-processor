package database

import (
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/stretchr/testify/require"
)

func TestApplyPoolRuntimeParams_statementTimeout(t *testing.T) {
	cfg, err := pgxpool.ParseConfig("postgres://localhost:5432/test")
	require.NoError(t, err)

	applyPoolRuntimeParams(cfg, PoolConfig{StatementTimeout: 30 * time.Second})
	require.Equal(t, "30000", cfg.ConnConfig.RuntimeParams["statement_timeout"])
}

func TestApplyPoolRuntimeParams_settlementStatementTimeout(t *testing.T) {
	cfg, err := pgxpool.ParseConfig("postgres://localhost:5432/test")
	require.NoError(t, err)

	applyPoolRuntimeParams(cfg, PoolConfig{StatementTimeout: 60 * time.Second})
	require.Equal(t, "60000", cfg.ConnConfig.RuntimeParams["statement_timeout"])
}

func TestApplyPoolRuntimeParams_zeroDisabled(t *testing.T) {
	cfg, err := pgxpool.ParseConfig("postgres://localhost:5432/test")
	require.NoError(t, err)

	applyPoolRuntimeParams(cfg, PoolConfig{})
	require.Empty(t, cfg.ConnConfig.RuntimeParams)
}

func TestApplyPoolConfig_defaults(t *testing.T) {
	cfg, err := pgxpool.ParseConfig("postgres://localhost:5432/test")
	require.NoError(t, err)

	applyPoolConfig(cfg, PoolConfig{})
	require.Equal(t, defaultPostgresConnectTimeout, cfg.ConnConfig.ConnectTimeout)
	require.Equal(t, defaultPostgresMaxConnLifetime, cfg.MaxConnLifetime)
}

func TestApplyPoolConfig_overrides(t *testing.T) {
	cfg, err := pgxpool.ParseConfig("postgres://localhost:5432/test")
	require.NoError(t, err)

	applyPoolConfig(cfg, PoolConfig{
		ConnectTimeout:  3 * time.Second,
		MaxConnLifetime: 2 * time.Hour,
	})
	require.Equal(t, 3*time.Second, cfg.ConnConfig.ConnectTimeout)
	require.Equal(t, 2*time.Hour, cfg.MaxConnLifetime)
}
