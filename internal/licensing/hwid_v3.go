package licensing

import "ad-event-processor/internal/config"

// HWIDTelemetryView documents live telemetry fields shown on license status.
type HWIDTelemetryView struct {
	DMIUUID   string `json:"dmi_uuid,omitempty"`
	DiskID    string `json:"disk_id,omitempty"`
	MAC       string `json:"mac,omitempty"`
	CPUModel  string `json:"cpu_model,omitempty"`
	CPUCores  int    `json:"cpu_cores,omitempty"`
	MachineID string `json:"machine_id,omitempty"`
	V3Enabled bool   `json:"v3_enabled,omitempty"`
}

// SnapshotHWIDTelemetry returns the telemetry inputs used for hwid_v2 on this host.
func SnapshotHWIDTelemetry() HWIDTelemetryView {
	tel := hwidCollectFn()
	view := HWIDTelemetryView{
		DMIUUID:   tel.DMIUUID,
		DiskID:    tel.DiskID,
		MAC:       tel.MAC,
		CPUModel:  tel.CPUModel,
		CPUCores:  tel.CPUCores,
		V3Enabled: HWIDV3Enabled(),
	}
	if view.V3Enabled {
		view.MachineID = tel.MachineID
	}
	return view
}

// HWIDV3Enabled reports whether systemd machine-id is mixed into HWID Argon2 input.
func HWIDV3Enabled() bool {
	return config.HWIDV3Enabled()
}
