package moderatorcorpus

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestModeratorCorpus_MatchJA3Only(t *testing.T) {
	entry, err := EntryFromTuple(Tuple{JA3: "771,4865-4866"})
	require.NoError(t, err)
	snap := BuildSnapshot([]Entry{entry}, 1)

	ja3 := []byte("771,4865-4866")
	assert.True(t, MatchSnapshot(snap, ja3, nil, 0, 0, nil, 0))
	assert.False(t, MatchSnapshot(snap, []byte("other"), nil, 0, 0, nil, 0))
}

func TestModeratorCorpus_MatchJA3JA4(t *testing.T) {
	entry, err := EntryFromTuple(Tuple{JA3: "771,4865", JA4: "t13d1516h2"})
	require.NoError(t, err)
	snap := BuildSnapshot([]Entry{entry}, 1)

	assert.True(t, MatchSnapshot(snap, []byte("771,4865"), []byte("t13d1516h2"), 0, 0, nil, 0))
	assert.False(t, MatchSnapshot(snap, []byte("771,4865"), []byte("other"), 0, 0, nil, 0))
}

func TestModeratorCorpus_EmptySnapshotFailOpen(t *testing.T) {
	assert.False(t, MatchSnapshot(nil, []byte("771,4865"), nil, 0, 0, nil, 0))
	assert.False(t, MatchSnapshot(&Snapshot{}, []byte("771,4865"), nil, 0, 0, nil, 0))
}

func TestModeratorCorpus_ParseFeedLine(t *testing.T) {
	tuple, err := ParseFeedLine("ja3:771,4865|ja4:t13d1516h2|tcp:00eaf022|desync:2")
	require.NoError(t, err)
	assert.Equal(t, "771,4865", tuple.JA3)
	assert.Equal(t, "t13d1516h2", tuple.JA4)
	assert.Equal(t, "00eaf022", tuple.TCPSig)
	assert.Equal(t, uint8(2), tuple.LayerDesyncCount)
}

func TestModeratorCorpus_FormatFeedRoundTrip(t *testing.T) {
	entries, err := ParseFeed([]byte("ja3:771,4865|ja4:t13d1516h2\n"))
	require.NoError(t, err)
	require.Len(t, entries, 1)
	formatted := FormatFeed(entries)
	again, err := ParseFeed(formatted)
	require.NoError(t, err)
	require.Len(t, again, 1)
	assert.Equal(t, entries[0].JA3Hash, again[0].JA3Hash)
	assert.Equal(t, entries[0].JA4Hash, again[0].JA4Hash)
}
