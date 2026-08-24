package licensing

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func TestGenHWIDVectorArtifacts(t *testing.T) {
	if os.Getenv("WRITE_HWID_VECTORS") == "" {
		t.Skip("set WRITE_HWID_VECTORS=1 to regenerate argon2id_hwid.json")
	}
	fixtures := []struct {
		Name  string
		Input HWIDTelemetry
	}{
		{"pilot_vm", HWIDTelemetry{DMIUUID: "11111111-2222-3333-4444-555555555555", DiskID: "disk-serial-abc", MAC: "52:54:00:12:34:56", CPUModel: "QEMU Virtual CPU version 2.5+", CPUCores: 4}},
		{"missing_dmi", HWIDTelemetry{DiskID: "/dev/vda1", MAC: "52:54:00:ab:cd:ef", CPUModel: "Intel(R) Xeon(R)", CPUCores: 2}},
		{"bare_metal", HWIDTelemetry{DMIUUID: "7c43435b-1234-5678-9abc-def012345678", DiskID: "S3YNNX0K123456", MAC: "00:1a:2b:3c:4d:5e", CPUModel: "AMD EPYC 7763 64-Core Processor", CPUCores: 32}},
	}
	out := struct {
		Fixtures []struct {
			Name     string        `json:"name"`
			Input    HWIDTelemetry `json:"input"`
			HWIDHash string        `json:"hwid_hash"`
		} `json:"fixtures"`
	}{}
	for _, fx := range fixtures {
		out.Fixtures = append(out.Fixtures, struct {
			Name     string        `json:"name"`
			Input    HWIDTelemetry `json:"input"`
			HWIDHash string        `json:"hwid_hash"`
		}{fx.Name, fx.Input, HashHWIDFromTelemetry(fx.Input)})
	}
	raw, err := json.MarshalIndent(out, "", " ")
	if err != nil {
		t.Fatal(err)
	}
	path := filepath.Join("testdata", "argon2id_hwid.json")
	if err := os.WriteFile(path, append(raw, '\n'), 0o644); err != nil {
		t.Fatal(err)
	}
}
