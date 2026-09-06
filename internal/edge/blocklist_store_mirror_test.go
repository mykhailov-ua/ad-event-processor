package edge

import (
	"os"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestBlocklistStore_hostDenyMirrorsLPMV4(t *testing.T) {
	maps := newTestBlocklistMapsV4Only(t)
	store := NewBlocklistStore()

	added, removed, err := store.ApplyDiff(maps, []string{"198.51.100.55"}, nil, nil)
	require.NoError(t, err)
	require.Equal(t, 1, added)
	require.Equal(t, 0, removed)

	key, ok := ParseHost("198.51.100.55")
	require.True(t, ok)
	addr := beToBPFAddr(key)

	var hostMarker uint8
	require.NoError(t, maps.V4Host.Lookup(addr, &hostMarker))
	require.Equal(t, blockedMarker, hostMarker)

	var lpmMarker uint8
	require.NoError(t, maps.V4Prefix.Lookup(key, &lpmMarker))
	require.Equal(t, blockedMarker, lpmMarker)

	added, removed, err = store.ApplyDiff(maps, nil, nil, nil)
	require.NoError(t, err)
	require.Equal(t, 0, added)
	require.Equal(t, 1, removed)

	err = maps.V4Host.Lookup(addr, &hostMarker)
	require.Error(t, err)
	err = maps.V4Prefix.Lookup(key, &lpmMarker)
	require.Error(t, err)
}

func TestBlocklistStore_hostDenyMirrorsLPMV6(t *testing.T) {
	maps := BlocklistMaps{
		V6Host:   newHostHashMapV6(t),
		V6Prefix: newLPMMapV6(t),
	}
	store := NewBlocklistStore()

	added, removed, err := store.ApplyDiff(maps, nil, nil, []string{"2001:db8::mirror"})
	require.NoError(t, err)
	require.Equal(t, 1, added)
	require.Equal(t, 0, removed)

	key, ok := ParseIPv6Host("2001:db8::mirror")
	require.True(t, ok)

	var hostMarker uint8
	require.NoError(t, maps.V6Host.Lookup(key.Addr, &hostMarker))
	require.Equal(t, blockedMarker, hostMarker)

	var lpmMarker uint8
	require.NoError(t, maps.V6Prefix.Lookup(key, &lpmMarker))
	require.Equal(t, blockedMarker, lpmMarker)
}

func TestBlocklistStore_hostLPMMirror_holdout(t *testing.T) {
	filter, err := os.ReadFile("../../deploy/edge/xdp/bpf/edge_filter.c")
	require.NoError(t, err)
	require.Contains(t, string(filter), "bpf_lpm_mirror")

	store, err := os.ReadFile("blocklist_store.go")
	require.NoError(t, err)
	require.Contains(t, string(store), "upsertHostDenyV4")
	require.Contains(t, string(store), "upsertHostDenyV6")
}
