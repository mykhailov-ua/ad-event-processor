package quorum_test

import (
	"context"
	"testing"

	"github.com/bidshard/ad-event-processor/internal/database"
	"github.com/bidshard/ad-event-processor/pkg/regionproxy/quorum"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

func TestBook_2of3Quorum(t *testing.T) {
	ctx := context.Background()
	rdb, cleanup := database.SetupTestRedis(t)
	t.Cleanup(cleanup)

	opID := uuid.New()
	nodes := []string{"proxy-a", "proxy-b", "proxy-c"}

	st, err := quorum.Book(ctx, rdb, toBytes(opID), nodes, "proxy-a")
	require.NoError(t, err)
	require.False(t, st.QuorumMet)
	require.Equal(t, int32(1), st.AckCount)

	st, err = quorum.AckBook(ctx, rdb, toBytes(opID), len(nodes), "proxy-b")
	require.NoError(t, err)
	require.True(t, st.QuorumMet)
	require.Equal(t, int32(2), st.AckCount)
	require.Equal(t, quorum.StateBooked, st.State)
}

func TestTransition_bookedExecutingCompleted(t *testing.T) {
	ctx := context.Background()
	rdb, cleanup := database.SetupTestRedis(t)
	t.Cleanup(cleanup)

	opID := uuid.New()
	nodes := []string{"proxy-a", "proxy-b", "proxy-c"}
	_, err := quorum.Book(ctx, rdb, toBytes(opID), nodes, "proxy-a")
	require.NoError(t, err)
	_, err = quorum.AckBook(ctx, rdb, toBytes(opID), len(nodes), "proxy-b")
	require.NoError(t, err)

	require.NoError(t, quorum.Transition(ctx, rdb, toBytes(opID), quorum.StateBooked, quorum.StateExecuting))
	require.NoError(t, quorum.Transition(ctx, rdb, toBytes(opID), quorum.StateExecuting, quorum.StateCompleted))

	st, err := quorum.ReadStatus(ctx, rdb, toBytes(opID), len(nodes))
	require.NoError(t, err)
	require.Equal(t, quorum.StateCompleted, st.State)
}

func toBytes(id uuid.UUID) [16]byte {
	var out [16]byte
	copy(out[:], id[:])
	return out
}
