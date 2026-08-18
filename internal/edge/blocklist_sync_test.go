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
	rdb := &redisStub{sets: map[string][]string{
		redisKeyBlacklistFraud: {"198.51.100.9"},
	}}
	m := newLPMMap(t)
	store := NewBlocklistStore()

	added, removed, err := SyncBlocklistFromRedis(ctx, rdb, m, store)
	require.NoError(t, err)
	assert.Equal(t, 1, added)
	assert.Equal(t, 0, removed)
	assert.Equal(t, 1, store.Len())

	key := KeyFromHost(198, 51, 100, 9)
	var val uint8
	require.NoError(t, m.Lookup(key, &val))
	assert.Equal(t, blockedMarker, val)
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
	m := newLPMMap(t)
	store := NewBlocklistStore()

	added, removed, err := store.ApplyDiff(m, nil, nil, []string{"198.51.100.1"})
	require.NoError(t, err)
	assert.Equal(t, 1, added)
	assert.Equal(t, 0, removed)

	added, removed, err = store.ApplyDiff(m, nil, nil, nil)
	require.NoError(t, err)
	assert.Equal(t, 0, added)
	assert.Equal(t, 1, removed)
	assert.Equal(t, 0, store.Len())

	key := KeyFromHost(198, 51, 100, 1)
	var val uint8
	err = m.Lookup(key, &val)
	assert.Error(t, err)
}

func TestBlocklistApplyDiff_skipsProtected(t *testing.T) {
	os.Setenv("INSTALL_LAN_CIDR", "192.168.1.0/24")
	defer os.Unsetenv("INSTALL_LAN_CIDR")
	ResetProtectedForTest()

	m := newLPMMap(t)
	store := NewBlocklistStore()

	added, removed, err := store.ApplyDiff(m, []string{"8.8.8.8", "192.168.1.10", "198.51.100.1"}, nil, nil)
	require.NoError(t, err)
	assert.Equal(t, 1, added)
	assert.Equal(t, 0, removed)

	var val uint8
	err = m.Lookup(KeyFromHost(198, 51, 100, 1), &val)
	require.NoError(t, err)

	err = m.Lookup(KeyFromHost(8, 8, 8, 8), &val)
	assert.Error(t, err)

	err = m.Lookup(KeyFromHost(192, 168, 1, 10), &val)
	assert.Error(t, err)
}
