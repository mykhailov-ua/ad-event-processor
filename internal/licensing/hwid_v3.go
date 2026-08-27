package licensing

import "ad-event-processor/internal/config"

type HWIDTelemetryView struct {
	DMIUUID   string `json:"dmi_uuid,omitempty"`
	DiskID    string `json:"disk_id,omitempty"`
	MAC       string `json:"mac,omitempty"`
	CPUModel  string `json:"cpu_model,omitempty"`
	CPUCores  int    `json:"cpu_cores,omitempty"`
	MachineID string `json:"machine_id,omitempty"`
	V3Enabled bool   `json:"v3_enabled,omitempty"`
}

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

func HWIDV3Enabled() bool {
	return config.HWIDV3Enabled()
}
