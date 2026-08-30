package ingest

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestWorkerArena_acquireRelease(t *testing.T) {
	var a workerArena
	releases := make([]func(), 0, offloadArenaSlots)
	for i := range offloadArenaSlots {
		slot, buf, release, ok := a.Acquire(128)
		require.True(t, ok, "slot %d", i)
		require.Equal(t, i, slot)
		require.Len(t, buf, 128)
		releases = append(releases, release)
	}
	_, _, _, ok := a.Acquire(64)
	require.False(t, ok)
	for _, release := range releases {
		release()
	}
	_, _, release, ok := a.Acquire(64)
	require.True(t, ok)
	release()
}
