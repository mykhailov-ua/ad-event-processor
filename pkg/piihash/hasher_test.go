package piihash_test

import (
	"testing"

	"ad-event-processor/pkg/piihash"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestHasher_deterministicAndDistinctDomains(t *testing.T) {
	h := piihash.TestHasher()

	ip1 := h.HashIP("203.0.113.10")
	ip2 := h.HashIP("203.0.113.10")
	assert.Equal(t, ip1, ip2)

	ua := h.HashUA("Mozilla/5.0")
	assert.NotEqual(t, ip1, ua)

	uid := h.HashUserID("user-42")
	assert.NotEqual(t, ip1, uid)
	assert.NotEqual(t, ua, uid)
}

func TestHasher_emptyInputZero(t *testing.T) {
	h := piihash.TestHasher()
	assert.Equal(t, [16]byte{}, h.HashIP(""))
	assert.Equal(t, [16]byte{}, h.HashUA(""))
}

func TestNewFromSalt_derivesFromToken(t *testing.T) {
	h, err := piihash.NewFromSalt(2, "", "test-secret")
	require.NoError(t, err)
	assert.Equal(t, uint8(2), h.Version())
	assert.NotEqual(t, [16]byte{}, h.HashIP("1.2.3.4"))
}

func TestNewFromSalt_explicitSaltHex(t *testing.T) {
	h, err := piihash.NewFromSalt(3, "0102030405060708090a0b0c0d0e0f101112131415161718191a1b1c1d1e1f20", "")
	require.NoError(t, err)
	assert.Equal(t, uint8(3), h.Version())
}
