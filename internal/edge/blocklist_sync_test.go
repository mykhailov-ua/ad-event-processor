package edge

import (
	"context"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSyncBlocklistFromRedis_fraudOnly(t *testing.T) {
	ctx := context.Background()
	redisClient := &redisStub{sets: map[string][]string{
		redisKeyBlacklistFraud: {"198.51.100.9"},
	}}
	maps := newTestBlocklistMapsV4Only(t)
	store := NewBlocklistStore()

	added, removed, err := SyncBlocklistFromRedis(ctx, redisClient, maps, store)
	require.NoError(t, err)
	assert.Equal(t, 1, added)
	assert.Equal(t, 0, removed)
	assert.Equal(t, 1, store.Len())

	hostAddr := HostKey(198, 51, 100, 9).Addr
	var val uint8
	require.NoError(t, maps.V4Host.Lookup(hostAddr, &val))
	assert.Equal(t, blockedMarker, val)
}

func TestBlocklistStore_hostHashMap_holdout(t *testing.T) {
	maps := newTestBlocklistMapsV4Only(t)
	store := NewBlocklistStore()

	host := "198.51.100.42"
	cidr := "10.0.0.0/8"
	added, _, err := store.ApplyDiff(maps, nil, nil, []string{host, cidr})
	require.NoError(t, err)
	require.Equal(t, 2, added)

	hostAddr := HostKey(198, 51, 100, 42).Addr
	var val uint8
	require.NoError(t, maps.V4Host.Lookup(hostAddr, &val))
	assert.Equal(t, blockedMarker, val)

	hostLPM := HostKey(198, 51, 100, 42)
	err = maps.V4Prefix.Lookup(hostLPM, &val)
	assert.Error(t, err, "host /32 must not land in LPM trie")

	prefix, ok := ParsePrefix(cidr)
	require.True(t, ok)
	require.NoError(t, maps.V4Prefix.Lookup(prefix, &val))
	assert.Equal(t, blockedMarker, val)

	err = maps.V4Host.Lookup(prefix.Addr, &val)
	assert.Error(t, err, "CIDR prefix must not land in host HASH map")
}

func TestMergeDenyIPs_allSources(t *testing.T) {
	manual := []string{"203.0.113.1", "203.0.113.2"}
	auto := []string{"203.0.113.2", "203.0.113.3"}
	fraud := []string{"203.0.113.3", "203.0.113.4", "not-an-ip"}

	got := MergeDenyIPs(manual, auto, fraud)
	assert.Len(t, got, 4)
	for _, host := range []struct{ a, b, c, d byte }{
		{203, 0, 113, 1},
		{203, 0, 113, 2},
		{203, 0, 113, 3},
		{203, 0, 113, 4},
	} {
		_, ok := got[HostKey(host.a, host.b, host.c, host.d).Addr]
		assert.True(t, ok)
	}
}

func TestBlocklistApplyDiff_fraudRemoval(t *testing.T) {
	maps := newTestBlocklistMapsV4Only(t)
	store := NewBlocklistStore()

	added, removed, err := store.ApplyDiff(maps, nil, nil, []string{"198.51.100.1"})
	require.NoError(t, err)
	assert.Equal(t, 1, added)
	assert.Equal(t, 0, removed)

	added, removed, err = store.ApplyDiff(maps, nil, nil, nil)
	require.NoError(t, err)
	assert.Equal(t, 0, added)
	assert.Equal(t, 1, removed)
	assert.Equal(t, 0, store.Len())

	hostAddr := HostKey(198, 51, 100, 1).Addr
	var val uint8
	err = maps.V4Host.Lookup(hostAddr, &val)
	assert.Error(t, err)
}

func TestBlocklistApplyDiff_skipsProtected(t *testing.T) {
	os.Setenv("INSTALL_LAN_CIDR", "192.168.1.0/24")
	defer os.Unsetenv("INSTALL_LAN_CIDR")
	ResetProtectedForTest()

	maps := newTestBlocklistMapsV4Only(t)
	store := NewBlocklistStore()

	added, removed, err := store.ApplyDiff(maps, nil, []string{"8.8.8.8", "192.168.1.10", "198.51.100.1"}, nil)
	require.NoError(t, err)
	assert.Equal(t, 1, added)
	assert.Equal(t, 0, removed)

	var val uint8
	err = maps.V4Host.Lookup(HostKey(198, 51, 100, 1).Addr, &val)
	require.NoError(t, err)

	err = maps.V4Host.Lookup(HostKey(8, 8, 8, 8).Addr, &val)
	assert.Error(t, err)

	err = maps.V4Host.Lookup(HostKey(192, 168, 1, 10).Addr, &val)
	assert.Error(t, err)
}
