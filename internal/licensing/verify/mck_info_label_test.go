package verify_test

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"ad-event-processor/internal/licensing/verify"

	"github.com/stretchr/testify/require"
)

func TestMCKDerivation_releaseLabelMatchesGolden(t *testing.T) {
	path := filepath.Join("testdata", "mck_derivation.json")
	raw, err := os.ReadFile(path)
	require.NoError(t, err)
	var doc struct {
		MCKInfoLabel string `json:"mck_info_label"`
	}
	require.NoError(t, json.Unmarshal(raw, &doc))
	require.NotEmpty(t, doc.MCKInfoLabel)
	require.Equal(t, verify.DefaultMCKInfoLabel, doc.MCKInfoLabel)
	require.Equal(t, verify.MCKInfoLabel(), doc.MCKInfoLabel)
}

func TestMCKInfoLabel_ldflagsOverride(t *testing.T) {
	restore := verify.SetMCKInfoLabelForTest("license-mck-test-override")
	defer restore()
	require.Equal(t, "license-mck-test-override", verify.MCKInfoLabel())
}
