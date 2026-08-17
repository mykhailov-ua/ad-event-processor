package ingestion

import (
	"hash/crc32"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestTLSFingerprintTable_MatchJA3(t *testing.T) {
	ja3 := []byte("771,4865-4866-4867,0-23,29-23-24,0")
	h := crc32.ChecksumIEEE(ja3)
	table := NewTLSFingerprintTable()
	table.Publish(buildTLSFingerprintSnapshot([]uint32{h}, nil, 1))

	assert.True(t, table.MatchJA3(ja3))
	assert.False(t, table.MatchJA3([]byte("other-fingerprint")))
}

func TestTLSFingerprintTable_binarySearch(t *testing.T) {
	hashes := []uint32{10, 20, 30, 40}
	assert.True(t, tlsHashBlocked(hashes, 20))
	assert.False(t, tlsHashBlocked(hashes, 15))
}

func TestParseTLSFingerprintFeed(t *testing.T) {
	data := []byte("# comment\nja3:771,4865\nja4:abc,def\n")
	ja3, ja4 := parseTLSFingerprintFeed(data)
	require.Len(t, ja3, 1)
	require.Len(t, ja4, 1)
}
