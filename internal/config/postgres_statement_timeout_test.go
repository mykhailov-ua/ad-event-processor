package config

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestSettlementPGStatementTimeout_defaults(t *testing.T) {
	require.Equal(t, 60*time.Second, (&Config{}).SettlementPGStatementTimeout())
	require.Equal(t, 45*time.Second, (&Config{SettlementPGStatementTimeoutMs: 45000}).SettlementPGStatementTimeout())
}

func TestAdminPGStatementTimeout_defaults(t *testing.T) {
	require.Equal(t, 30*time.Second, (&Config{}).AdminPGStatementTimeout())
	require.Equal(t, 15*time.Second, (&Config{AdminPGStatementTimeoutMs: 15000}).AdminPGStatementTimeout())
}
