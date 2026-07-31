package controlplane

import (
	"espx/pkg/faultproof"

	"context"
	"testing"

	"espx/pkg/iogate"

	"github.com/stretchr/testify/require"
)

func TestFault_DiskGate_DegradedShedsTierLow(t *testing.T) {
	t.Parallel()

	gate := iogate.NewDiskWriteGate(iogate.Config{AppendCapacity: 4, GroupCommitRecords: 1})
	gate.SetDegraded(true)

	err := gate.AcquireAppend(context.Background(), iogate.TierLow)
	require.ErrorIs(t, err, iogate.ErrShed)

	require.NoError(t, gate.AcquireAppend(context.Background(), iogate.TierHigh))
	gate.ReleaseAppend(iogate.TierHigh)

	faultproof.Log(t, "mr_disk_degraded", map[string]string{
		"subsystem":     "region_proxy_disk_gate",
		"disk_degraded": "true",
		"tier_low_shed": "true",
		"baseline_ok":   "true",
	})
}
