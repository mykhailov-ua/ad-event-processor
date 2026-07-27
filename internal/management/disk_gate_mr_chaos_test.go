package management

import (
	"context"
	"testing"

	"espx/pkg/iogate"

	"github.com/stretchr/testify/require"
)

// TestChaos_DiskGate_DegradedShedsTierLow is CH-MR-04: degraded disk gate sheds TierLow appends.
func TestChaos_DiskGate_DegradedShedsTierLow(t *testing.T) {
	t.Parallel()

	gate := iogate.NewDiskWriteGate(iogate.Config{AppendCapacity: 4, GroupCommitRecords: 1})
	gate.SetDegraded(true)

	err := gate.AcquireAppend(context.Background(), iogate.TierLow)
	require.ErrorIs(t, err, iogate.ErrShed)

	require.NoError(t, gate.AcquireAppend(context.Background(), iogate.TierHigh))
	gate.ReleaseAppend(iogate.TierHigh)

	logChaosProof(t, "mr_disk_degraded", map[string]string{
		"subsystem":     "region_proxy_disk_gate",
		"disk_degraded": "true",
		"tier_low_shed": "true",
		"baseline_ok":   "true",
	})
}
