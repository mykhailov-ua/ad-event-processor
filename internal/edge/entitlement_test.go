package edge

import (
	"context"
	"testing"

	"github.com/alicebob/miniredis/v2"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestEbpfEdgeLicensed_missingKeyFailsOpen(t *testing.T) {
	mr, err := miniredis.Run()
	require.NoError(t, err)
	t.Cleanup(mr.Close)

	rdb := redis.NewClient(&redis.Options{Addr: mr.Addr()})
	t.Cleanup(func() { _ = rdb.Close() })

	assert.True(t, EbpfEdgeLicensed(context.Background(), rdb))
}

func TestEbpfEdgeLicensed_enabled(t *testing.T) {
	mr, err := miniredis.Run()
	require.NoError(t, err)
	t.Cleanup(mr.Close)

	mr.HSet(entitlementDeploymentKey, entitlementEbpfXDPEdge, "1")

	rdb := redis.NewClient(&redis.Options{Addr: mr.Addr()})
	t.Cleanup(func() { _ = rdb.Close() })

	assert.True(t, EbpfEdgeLicensed(context.Background(), rdb))
}

func TestEbpfEdgeLicensed_deniedWhenZero(t *testing.T) {
	mr, err := miniredis.Run()
	require.NoError(t, err)
	t.Cleanup(mr.Close)

	mr.HSet(entitlementDeploymentKey, entitlementEbpfXDPEdge, "0")

	rdb := redis.NewClient(&redis.Options{Addr: mr.Addr()})
	t.Cleanup(func() { _ = rdb.Close() })

	assert.False(t, EbpfEdgeLicensed(context.Background(), rdb))
}

func TestEbpfEdgeLicensed_nilClientFailsOpen(t *testing.T) {
	assert.True(t, EbpfEdgeLicensed(context.Background(), nil))
}
