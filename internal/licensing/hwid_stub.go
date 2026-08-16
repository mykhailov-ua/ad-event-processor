//go:build !linux

package licensing

import "runtime"

func collectHWIDTelemetry() HWIDTelemetry {
	return HWIDTelemetry{
		DMIUUID:  readMachineID(),
		DiskID:   "",
		MAC:      "",
		CPUModel: runtime.GOARCH,
		CPUCores: runtime.NumCPU(),
	}
}
