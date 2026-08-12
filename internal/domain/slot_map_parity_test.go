package domain

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestShardFromSlotTable_matchesStaticSlotSharder(t *testing.T) {
	sharder := NewStaticSlotSharder(4)
	table := buildSlotTable(4)
	sharder.SwapSnapshot(1, table, 0)

	slots := make([]uint16, SlotCount)
	for i := range slots {
		slots[i] = uint16(table[i])
	}

	const samples = 256
	for i := 0; i < samples; i++ {
		id := uuid.New()
		edgeShard, ok := ShardFromSlotTable(id, slots)
		require.True(t, ok)
		assert.Equal(t, sharder.GetShard(id), edgeShard)
	}
}

func TestCompareSlotMaps_detectsDrift(t *testing.T) {
	a := make([]uint16, SlotCount)
	b := make([]uint16, SlotCount)
	for i := range a {
		a[i] = uint16(i % 4)
		b[i] = uint16(i % 4)
	}
	b[42] = 99

	diffs, first := CompareSlotMaps(a, b)
	assert.Equal(t, 1, diffs)
	assert.Equal(t, 42, first)
	assert.True(t, SlotMapsEqual(a, a))
	assert.False(t, SlotMapsEqual(a, b))
}

func TestFetchOpsSlotMapHTTP_roundtrip(t *testing.T) {
	want := OpsSlotMapResponse{
		Version:       3,
		ActiveVersion: 3,
		RoutingEpoch:  7,
		Slots:         make([]uint16, SlotCount),
	}
	for i := range want.Slots {
		want.Slots[i] = uint16(i % 4)
	}

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/ops/shards/slot-map", r.URL.Path)
		require.NoError(t, json.NewEncoder(w).Encode(want))
	}))
	t.Cleanup(srv.Close)

	got, err := FetchOpsSlotMapHTTP(context.Background(), srv.Client(), srv.URL)
	require.NoError(t, err)
	assert.Equal(t, want.Version, got.Version)
	assert.True(t, SlotMapsEqual(want.Slots, got.Slots))
}
