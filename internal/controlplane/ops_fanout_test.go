package controlplane

import (
	"context"
	"errors"
	"sync/atomic"
	"testing"
	"time"

	"ad-event-processor/internal/opsadmin"

	"github.com/stretchr/testify/assert"

	"github.com/stretchr/testify/require"
)

func TestCollectFanOut_allSourcesOK(t *testing.T) {
	t.Parallel()
	collector := opsadmin.NewFanOutCollector(nil, "test_ok")
	sources := []opsadmin.FanOutSource[int]{
		{ID: "0", Poll: func(ctx context.Context) ([]int, error) { return []int{1}, nil }},
		{ID: "1", Poll: func(ctx context.Context) ([]int, error) { return []int{2}, nil }},
	}
	result := opsadmin.CollectFanOut(context.Background(), collector, sources)
	require.False(t, result.Partial)
	require.Empty(t, result.Errors)
	require.Len(t, result.Items, 2)
}

func TestCollectFanOut_partialFailure(t *testing.T) {
	t.Parallel()
	collector := opsadmin.NewFanOutCollector(nil, "test_partial")
	sources := []opsadmin.FanOutSource[int]{
		{ID: "0", Poll: func(ctx context.Context) ([]int, error) { return []int{10}, nil }},
		{ID: "1", Poll: func(ctx context.Context) ([]int, error) { return nil, errors.New("down") }},
		{ID: "2", Poll: func(ctx context.Context) ([]int, error) { return []int{30}, nil }},
		{ID: "3", Poll: func(ctx context.Context) ([]int, error) { return []int{40}, nil }},
	}
	result := opsadmin.CollectFanOut(context.Background(), collector, sources)
	require.True(t, result.Partial)
	require.Len(t, result.Errors, 1)
	require.Equal(t, "1", result.Errors[0].Source)
	require.Len(t, result.Items, 3)
}

func TestCollectFanOut_respectsConcurrencyCap(t *testing.T) {
	t.Parallel()
	collector := &opsadmin.FanOutCollector{MaxConcurrency: 2, PerSourceTO: time.Second, Route: "cap"}
	var peak atomic.Int32
	var current atomic.Int32
	sources := make([]opsadmin.FanOutSource[int], 0, 6)
	for range 6 {
		sources = append(sources, opsadmin.FanOutSource[int]{
			ID: "s",
			Poll: func(ctx context.Context) ([]int, error) {
				cur := current.Add(1)
				for {
					p := peak.Load()
					if cur > p {
						if peak.CompareAndSwap(p, cur) {
							break
						}
						continue
					}
					break
				}
				time.Sleep(20 * time.Millisecond)
				current.Add(-1)
				return []int{1}, nil
			},
		})
	}
	_ = opsadmin.CollectFanOut(context.Background(), collector, sources)
	assert.LessOrEqual(t, peak.Load(), int32(2))
}

func TestFanOutCursor_roundTrip(t *testing.T) {
	t.Parallel()
	state := map[string]string{"0": "1234-0", "pg": "42"}
	encoded, err := opsadmin.EncodeFanOutCursor(state)
	require.NoError(t, err)
	decoded, err := opsadmin.DecodeFanOutCursor(encoded)
	require.NoError(t, err)
	assert.Equal(t, state["0"], decoded["0"])
	assert.Equal(t, state["pg"], decoded["pg"])
}

func TestParseDLQRouteID(t *testing.T) {
	t.Parallel()
	assert.Equal(t, 2, opsadmin.ParseDLQShardFromRoute("shard-2-1700000000000-0"))
	assert.Equal(t, "1700000000000-0", opsadmin.ParseDLQEntryIDFromRoute("shard-2-1700000000000-0"))
}
