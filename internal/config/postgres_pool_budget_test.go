package config

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestValidatePostgresPoolBudget_ok(t *testing.T) {
	cfg := &Config{
		DBTrackerMaxConns:          4,
		PostgresMaxConnections:     100,
		PostgresPoolConnHeadroom:   20,
		PostgresPoolSettleMaxConns: 10,
	}
	require.NoError(t, cfg.ValidatePostgresPoolBudget())
}

func TestValidatePostgresPoolBudget_exceeded(t *testing.T) {
	cfg := &Config{
		DBTrackerMaxConns:          40,
		PostgresMaxConnections:     50,
		PostgresPoolConnHeadroom:   20,
		PostgresPoolSettleMaxConns: 10,
	}
	err := cfg.ValidatePostgresPoolBudget()
	require.Error(t, err)
	require.Contains(t, err.Error(), "postgres pool budget exceeded")
}

func TestValidatePostgresPoolBudget_skipsWhenUnset(t *testing.T) {
	cfg := &Config{DBTrackerMaxConns: 1000}
	require.NoError(t, cfg.ValidatePostgresPoolBudget())
}
