package licensing_test

import (
	"testing"

	"ad-event-processor/internal/licensing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestHWID_V3OffMatchesGoldenVectors(t *testing.T) {
	t.Setenv("AD_EVENT_PROCESSOR_LICENSE_HWID_V3", "0")
	tel := licensing.HWIDTelemetry{
		DMIUUID:   "11111111-2222-3333-4444-555555555555",
		DiskID:    "disk-serial-abc",
		MAC:       "52:54:00:12:34:56",
		CPUModel:  "QEMU Virtual CPU version 2.5+",
		CPUCores:  4,
		MachineID: "should-not-affect-v2",
	}
	base := licensing.HashHWIDFromTelemetry(licensing.HWIDTelemetry{
		DMIUUID:  tel.DMIUUID,
		DiskID:   tel.DiskID,
		MAC:      tel.MAC,
		CPUModel: tel.CPUModel,
		CPUCores: tel.CPUCores,
	})
	require.Equal(t, base, licensing.HashHWIDFromTelemetry(tel))
}

func TestHWID_V3MachineIDChangesHash(t *testing.T) {
	t.Setenv("AD_EVENT_PROCESSOR_LICENSE_HWID_V3", "1")
	tel := licensing.HWIDTelemetry{
		DMIUUID:  "11111111-2222-3333-4444-555555555555",
		DiskID:   "disk-serial-abc",
		MAC:      "52:54:00:12:34:56",
		CPUModel: "QEMU Virtual CPU version 2.5+",
		CPUCores: 4,
	}
	withoutMachine := licensing.HashHWIDFromTelemetry(tel)
	tel.MachineID = "a1b2c3d4e5f6478990abcdef01234567"
	withMachine := licensing.HashHWIDFromTelemetry(tel)
	require.NotEqual(t, withoutMachine, withMachine)
	require.Len(t, withMachine, 64)
}

func TestSnapshotHWIDTelemetry_omitsMachineIDWhenV3Off(t *testing.T) {
	t.Setenv("AD_EVENT_PROCESSOR_LICENSE_HWID_V3", "0")
	restore := licensing.SetHWIDCollectForTest(func() licensing.HWIDTelemetry {
		return licensing.HWIDTelemetry{
			DMIUUID:   "dmi",
			DiskID:    "disk",
			MAC:       "mac",
			CPUModel:  "cpu",
			CPUCores:  2,
			MachineID: "secret-machine-id",
		}
	})
	defer restore()

	view := licensing.SnapshotHWIDTelemetry()
	assert.False(t, view.V3Enabled)
	assert.Empty(t, view.MachineID)
	assert.Equal(t, "dmi", view.DMIUUID)
}

func TestSnapshotHWIDTelemetry_includesMachineIDWhenV3On(t *testing.T) {
	t.Setenv("AD_EVENT_PROCESSOR_LICENSE_HWID_V3", "1")
	restore := licensing.SetHWIDCollectForTest(func() licensing.HWIDTelemetry {
		return licensing.HWIDTelemetry{
			DMIUUID:   "dmi",
			MachineID: "machine-id-value",
		}
	})
	defer restore()

	view := licensing.SnapshotHWIDTelemetry()
	assert.True(t, view.V3Enabled)
	assert.Equal(t, "machine-id-value", view.MachineID)
}
