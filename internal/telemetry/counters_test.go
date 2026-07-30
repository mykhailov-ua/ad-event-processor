package telemetry

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestCountersSnapshotAndReset(t *testing.T) {
	ResetForTest()
	RecordAccepted()
	RecordAccepted()
	RecordRejected()
	RecordTrack()
	RecordTrack()
	RecordTrack()

	snap := SnapshotAndReset()
	assert.Equal(t, uint64(2), snap.AcceptedEvents)
	assert.Equal(t, uint64(1), snap.RejectedEvents)
	assert.GreaterOrEqual(t, snap.PeakRPS, uint64(1))

	empty := SnapshotAndReset()
	assert.Equal(t, uint64(0), empty.AcceptedEvents)
	assert.Equal(t, uint64(0), empty.RejectedEvents)
}
