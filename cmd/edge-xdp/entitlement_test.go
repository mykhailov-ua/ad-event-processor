package main

import (
	"context"
	"testing"

	"ad-event-processor/internal/edge"
	"github.com/alicebob/miniredis/v2"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestEbpfEdgeAttachAllowed_deniedWhenEntitlementZero(t *testing.T) {
	mr, err := miniredis.Run()
	require.NoError(t, err)
	t.Cleanup(mr.Close)

	mr.HSet("entitlement:deployment", "ebpf_xdp_edge", "0")

	rdb := redis.NewClient(&redis.Options{Addr: mr.Addr()})
	t.Cleanup(func() { _ = rdb.Close() })

	assert.False(t, edge.EbpfEdgeLicensed(context.Background(), rdb))
}
