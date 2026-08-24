package opkey

import (
	"context"
	"testing"

	"ad-event-processor/internal/database"
	"ad-event-processor/pkg/regionproxy/quorum"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestBatchCommitter_quorumBeforeForward(t *testing.T) {
	ctx := context.Background()
	rdb, cleanup := database.SetupTestRedis(t)
	t.Cleanup(cleanup)

	replicas := []string{"proxy-a", "proxy-b", "proxy-c"}
	committerA := NewBatchCommitter(rdb, "proxy-a", replicas)
	committerB := NewBatchCommitter(rdb, "proxy-b", replicas)

	var slot Slot
	slot.Seq = 42
	slot.setDerived()
	gen := newIDGen("proxy-a")
	gen.next(&slot.OpID)

	ready, err := committerA.PrepareForward(ctx, &slot)
	require.NoError(t, err)
	assert.False(t, ready)

	ready, err = committerB.PrepareForward(ctx, &slot)
	require.NoError(t, err)
	assert.True(t, ready)
	assert.True(t, slot.Has(OpKeyFlagExecuting))

	committerB.Complete(ctx, &slot)
	st, err := quorum.ReadStatus(ctx, rdb, slot.OpID, len(replicas))
	require.NoError(t, err)
	assert.Equal(t, quorum.StateCompleted, st.State)
}
