package broker

import (
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestConsumerOffsetTracker_CommitAndRead(t *testing.T) {
	dir := t.TempDir()
	tracker, err := NewConsumerOffsetTracker(dir)
	require.NoError(t, err)

	topic := "ad-events"
	partition := uint16(0)
	group := "clickhouse_processor"

	off, err := tracker.GetCommittedOffset(topic, partition, group)
	require.NoError(t, err)
	assert.Equal(t, uint64(0), off)

	require.NoError(t, tracker.CommitOffset(topic, partition, group, 50000))

	off, err = tracker.GetCommittedOffset(topic, partition, group)
	require.NoError(t, err)
	assert.Equal(t, uint64(50000), off)

	require.NoError(t, tracker.CommitOffset(topic, partition, group, 25000))

	off, err = tracker.GetCommittedOffset(topic, partition, group)
	require.NoError(t, err)
	assert.Equal(t, uint64(50000), off)
}

func TestConsumerOffsetTracker_PersistenceAcrossRestarts(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "offsets")
	tracker1, err := NewConsumerOffsetTracker(dir)
	require.NoError(t, err)

	require.NoError(t, tracker1.CommitOffset("ad-events", 0, "ch_group", 125000))
	require.NoError(t, tracker1.CommitOffset("ad-events", 1, "ch_group", 80000))

	tracker2, err := NewConsumerOffsetTracker(dir)
	require.NoError(t, err)

	off0, err := tracker2.GetCommittedOffset("ad-events", 0, "ch_group")
	require.NoError(t, err)
	assert.Equal(t, uint64(125000), off0)

	off1, err := tracker2.GetCommittedOffset("ad-events", 1, "ch_group")
	require.NoError(t, err)
	assert.Equal(t, uint64(80000), off1)
}
