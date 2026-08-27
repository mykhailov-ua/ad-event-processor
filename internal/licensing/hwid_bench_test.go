package licensing_test

import (
	"testing"

	"ad-event-processor/internal/licensing"
)

func BenchmarkHostHWID(b *testing.B) {
	restore := licensing.SetHWIDCollectForTest(func() licensing.HWIDTelemetry {
		return licensing.HWIDTelemetry{
			DMIUUID:  "11111111-2222-3333-4444-555555555555",
			DiskID:   "disk-serial-abc",
			MAC:      "52:54:00:12:34:56",
			CPUModel: "QEMU Virtual CPU version 2.5+",
			CPUCores: 4,
		}
	})
	defer restore()

	for b.Loop() {
		_ = licensing.HashHWIDFromTelemetry(licensing.HWIDTelemetry{
			DMIUUID:  "11111111-2222-3333-4444-555555555555",
			DiskID:   "disk-serial-abc",
			MAC:      "52:54:00:12:34:56",
			CPUModel: "QEMU Virtual CPU version 2.5+",
			CPUCores: 4,
		})
	}
}
