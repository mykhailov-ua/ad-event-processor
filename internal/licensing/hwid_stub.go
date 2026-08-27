//go:build !linux

package licensing

import "runtime"

func collectHWIDTelemetry() HWIDTelemetry {
	tel := HWIDTelemetry{
		DMIUUID:  readMachineID(),
		DiskID:   "",
		MAC:      "",
		CPUModel: runtime.GOARCH,
		CPUCores: runtime.NumCPU(),
	}
	if HWIDV3Enabled() {
		tel.MachineID = readMachineID()
	}
	return tel
}
