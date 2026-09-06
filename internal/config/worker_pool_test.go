package config

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestWorkerPoolConfig_defaults(t *testing.T) {
	cfg := &Config{MaxWorkers: 8, WorkerPoolQueueDepth: 4096}
	wp := cfg.WorkerPoolConfig()
	require.Equal(t, 8, wp.Workers)
	require.Equal(t, 4096, wp.QueueDepth)
	require.Equal(t, DefaultWorkerArenaSlots, wp.ArenaSlots)
	require.Equal(t, DefaultWorkerArenaSlotBytes, wp.ArenaSlotBytes)
	require.Equal(t, DefaultWorkerMaxPoolObjectBytes, wp.MaxPoolObjectBytes)
}

func TestWorkerPoolConfig_zeroWorkersUsesGOMAXPROCS(t *testing.T) {
	cfg := &Config{MaxWorkers: 0, WorkerPoolQueueDepth: 0}
	wp := cfg.WorkerPoolConfig()
	require.Greater(t, wp.Workers, 0)
	require.Equal(t, DefaultWorkerPoolQueueDepth, wp.QueueDepth)
}
