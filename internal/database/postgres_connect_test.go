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

func TestApplyPoolRuntimeParams_zeroDisabled(t *testing.T) {
	cfg, err := pgxpool.ParseConfig("postgres://localhost:5432/test")
	require.NoError(t, err)

	applyPoolRuntimeParams(cfg, PoolConfig{})
	require.Empty(t, cfg.ConnConfig.RuntimeParams)
}
