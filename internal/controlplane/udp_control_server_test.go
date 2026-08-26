package controlplane

import (
	"context"
	"testing"
	"time"

	"ad-event-processor/internal/config"

	"github.com/stretchr/testify/require"
)

func TestUDPControlServer_buildLimits_defaultsWithoutPool(t *testing.T) {
	t.Parallel()
	cfg := &config.Config{UDPDefaultShardRPS: 12_345}
	s := NewUDPControlServer(cfg, nil, nil, 4)
	limits := s.buildLimits(context.Background())
	require.Equal(t, uint8(4), limits.NumShards)
	require.Equal(t, uint64(12_345), limits.Limits[0])
	require.Equal(t, uint64(12_345), limits.Limits[3])
}

func TestUDPControlServer_buildLimits_usesDefaultsOnCanceledCtx(t *testing.T) {
	t.Parallel()
	cfg := &config.Config{UDPDefaultShardRPS: 9_999, MultiRegionEnabled: true}
	s := NewUDPControlServer(cfg, nil, nil, 2)
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	limits := s.buildLimits(ctx)
	require.Equal(t, uint8(2), limits.NumShards)
	require.Equal(t, uint64(9_999), limits.Limits[0])
	require.Zero(t, limits.MaxRPD)
}

func TestUDPControlServer_buildLimits_queryTimeoutConstant(t *testing.T) {
	t.Parallel()
	require.Equal(t, 5*time.Second, udpLimitsQueryTimeout)
}
