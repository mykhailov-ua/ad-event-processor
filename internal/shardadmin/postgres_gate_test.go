package shardadmin

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestPostgresGate_LowRejectedWhenBudgetExhausted(t *testing.T) {
	gate := NewPostgresGate(3)
	ctx := context.Background()

	require.NoError(t, gate.AcquireLow(ctx))
	require.ErrorIs(t, gate.AcquireLow(ctx), ErrPostgresGateRejected)
	gate.ReleaseLow()
}

func TestPostgresGate_HighUsesReservedSlot(t *testing.T) {
	gate := NewPostgresGate(3)
	ctx := context.Background()

	require.NoError(t, gate.AcquireLow(ctx))
	require.NoError(t, gate.AcquireHigh(ctx))
	gate.ReleaseHigh()
	gate.ReleaseLow()
}
