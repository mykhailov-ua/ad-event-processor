package controlplane

import (
	"math"

	"ad-event-processor/internal/licensing"
)

const defaultExportChunkMaxBytes = 10 * 1024 * 1024

func exportChunkMaxBytes() int {
	limits, state, ok := licenseDeploymentLimits()
	if !ok {
		return defaultExportChunkMaxBytes
	}
	if state == licensing.StateExpired || state == licensing.StateRevoked {
		return defaultExportChunkMaxBytes
	}
	if limits.MaxExportChunkBytes == 0 {
		return defaultExportChunkMaxBytes
	}
	if limits.MaxExportChunkBytes > uint64(math.MaxInt) {
		return math.MaxInt
	}
	return int(limits.MaxExportChunkBytes)
}
