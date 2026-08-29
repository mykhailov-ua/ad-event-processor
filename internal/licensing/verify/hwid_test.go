package verify_test

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"ad-event-processor/internal/licensing/verify"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestHWID_ArgonParamsDocumented(t *testing.T) {
	require.Equal(t, uint32(3), verify.HWIDArgonTime())
	require.Equal(t, uint32(65536), verify.HWIDArgonMemoryKiB())
	require.Equal(t, uint8(4), verify.HWIDArgonThreads())
	require.Equal(t, uint32(32), verify.HWIDArgonKeyLen())
}

func TestHWID_LabCollectPrint(t *testing.T) {
	if os.Getenv("HWID_LAB_COLLECT") == "" {
		t.Skip("set HWID_LAB_COLLECT=1 for lab collection")
	}
	tel, hash := verify.LabCollectHWID()
	t.Logf("dmi_uuid=%q disk_id=%q mac=%q cpu_model=%q cpu_cores=%d hwid_v2=%s",
		tel.DMIUUID, tel.DiskID, tel.MAC, tel.CPUModel, tel.CPUCores, hash)
	require.Len(t, hash, 64)
}

func TestHWID_Deterministic(t *testing.T) {
	tel := verify.HWIDTelemetry{
		DMIUUID:  "11111111-2222-3333-4444-555555555555",
		DiskID:   "disk-serial-abc",
		MAC:      "52:54:00:12:34:56",
		CPUModel: "QEMU Virtual CPU version 2.5+",
		CPUCores: 4,
	}
	first := verify.HashHWIDFromTelemetry(tel)
	for i := range 100 {
		got := verify.HashHWIDFromTelemetry(tel)
		require.Equal(t, first, got, "iteration %d", i)
	}
	require.Len(t, first, 64)
}

func TestHWID_VMFallback(t *testing.T) {
	tel := verify.HWIDTelemetry{
		DMIUUID:  "",
		DiskID:   "/dev/vda1",
		MAC:      "52:54:00:ab:cd:ef",
		CPUModel: "Intel(R) Xeon(R)",
		CPUCores: 2,
	}
	hash := verify.HashHWIDFromTelemetry(tel)
	require.NotEmpty(t, hash)
	require.Len(t, hash, 64)

	emptyDMI := verify.HWIDTelemetry{
		DiskID:   tel.DiskID,
		MAC:      tel.MAC,
		CPUModel: tel.CPUModel,
		CPUCores: tel.CPUCores,
	}
	require.Equal(t, hash, verify.HashHWIDFromTelemetry(emptyDMI))
}

func TestHWID_GoldenVectors(t *testing.T) {
	path := filepath.Join("testdata", "argon2id_hwid.json")
	raw, err := os.ReadFile(path)
	require.NoError(t, err)

	var doc struct {
		Fixtures []struct {
			Name     string               `json:"name"`
			Input    verify.HWIDTelemetry `json:"input"`
			HWIDHash string               `json:"hwid_hash"`
		} `json:"fixtures"`
	}
	require.NoError(t, json.Unmarshal(raw, &doc))
	require.NotEmpty(t, doc.Fixtures)

	for _, fx := range doc.Fixtures {
		t.Run(fx.Name, func(t *testing.T) {
			got := verify.HashHWIDFromTelemetry(fx.Input)
			assert.Equal(t, fx.HWIDHash, got)
		})
	}
}

func TestVerifyDeploymentBind_hwidMismatch(t *testing.T) {
	claims := &verify.LicenseClaims{}
	claims.Bind.Mode = "hard"
	claims.HWIDHash = "deadbeef"
	err := verify.VerifyDeploymentBind(claims, "")
	require.ErrorIs(t, err, verify.ErrFingerprintMismatch)
}

func TestVerifyDeploymentBind_hwidMatch(t *testing.T) {
	tel := verify.HWIDTelemetry{
		DMIUUID:  "fixture-dmi",
		DiskID:   "fixture-disk",
		MAC:      "aa:bb:cc:dd:ee:ff",
		CPUModel: "Fixture CPU",
		CPUCores: 8,
	}
	expected := verify.HashHWIDFromTelemetry(tel)

	restore := setHWIDTelemetryForTest(tel)
	defer restore()

	claims := &verify.LicenseClaims{}
	claims.Bind.Mode = "hard"
	claims.HWIDHash = expected
	require.NoError(t, verify.VerifyDeploymentBind(claims, ""))
}

func setHWIDTelemetryForTest(tel verify.HWIDTelemetry) func() {
	return verify.SetHWIDCollectForTest(func() verify.HWIDTelemetry { return tel })
}
