package verify_test

import (
	"testing"

	"ad-event-processor/internal/licensing/verify"
)

func BenchmarkHostHWID(b *testing.B) {
	restore := verify.SetHWIDCollectForTest(func() verify.HWIDTelemetry {
		return verify.HWIDTelemetry{
			DMIUUID:  "11111111-2222-3333-4444-555555555555",
			DiskID:   "disk-serial-abc",
			MAC:      "52:54:00:12:34:56",
			CPUModel: "QEMU Virtual CPU version 2.5+",
			CPUCores: 4,
		}
	})
	defer restore()

	for b.Loop() {
		_ = verify.HashHWIDFromTelemetry(verify.HWIDTelemetry{
			DMIUUID:  "11111111-2222-3333-4444-555555555555",
			DiskID:   "disk-serial-abc",
			MAC:      "52:54:00:12:34:56",
			CPUModel: "QEMU Virtual CPU version 2.5+",
			CPUCores: 4,
		})
	}
}
