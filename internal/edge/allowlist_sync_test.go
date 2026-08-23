package edge

import (
	"context"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSyncAllowlistFromRedis_cidr(t *testing.T) {
	ctx := context.Background()
	rdb := &redisStub{sets: map[string][]string{
		redisKeyAllowlistPartners: {"10.0.0.0/8", "203.0.113.5"},
	}}
	m := newLPMMap(t)
	store := NewAllowlistStore()

	added, removed, err := SyncAllowlistFromRedis(ctx, rdb, m, nil, store)
	require.NoError(t, err)
	assert.Equal(t, 2, added)
	assert.Equal(t, 0, removed)

	var val uint8
	p8, ok := ParsePrefix("10.0.0.0/8")
	require.True(t, ok)
	require.NoError(t, m.Lookup(p8, &val))
	p32, ok := ParsePrefix("203.0.113.5")
	require.True(t, ok)
	require.NoError(t, m.Lookup(p32, &val))
	require.NoError(t, m.Lookup(IPv4Key{PrefixLen: 32, Addr: ToBPFAddr(0x0a090807)}, &val))
}

func TestAllowlistApplyDiff_cidrRemoval(t *testing.T) {
	m := newLPMMap(t)
	store := NewAllowlistStore()

	_, _, err := store.ApplyDiff(m, nil, []string{"198.51.100.0/24"})
	require.NoError(t, err)

	_, _, err = store.ApplyDiff(m, nil, nil)
	require.NoError(t, err)
	assert.Equal(t, 0, store.Len())

	var val uint8
	err = m.Lookup(IPv4Key{PrefixLen: 24, Addr: ToBPFAddr(0xc6336400)}, &val)
	assert.Error(t, err)
}

func TestIsProtected(t *testing.T) {
	os.Setenv("INSTALL_LAN_CIDR", "192.168.1.0/24")
	defer os.Unsetenv("INSTALL_LAN_CIDR")

	ResetProtectedForTest()

	assert.True(t, IsProtected("8.8.8.8"))
	assert.True(t, IsProtected("1.1.1.1"))
	assert.True(t, IsProtected("127.0.0.1"))
	assert.True(t, IsProtected("192.168.1.50"))

	assert.False(t, IsProtected("8.8.8.9"))
	assert.False(t, IsProtected("192.168.2.50"))
	assert.False(t, IsProtected("invalid-ip"))
}
