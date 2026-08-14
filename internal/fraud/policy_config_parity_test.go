package fraud

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type policyConfigParityFile struct {
	Cases []policyConfigParityCase `json:"cases"`
}

type policyConfigParityCase struct {
	ID     string             `json:"id"`
	Check  string             `json:"check"`
	Policy map[string]float64 `json:"policy"`
	Env    map[string]string  `json:"env"`
	Want   map[string]float64 `json:"want"`
}

func TestPolicyConfigParityFixtures(t *testing.T) {
	fixturesDir := fraudMLFixtureDir(t)
	path := filepath.Join(fixturesDir, "policy_config_parity.json")
	require.FileExists(t, path)

	data, err := os.ReadFile(path)
	require.NoError(t, err)

	var file policyConfigParityFile
	require.NoError(t, json.Unmarshal(data, &file))
	require.NotEmpty(t, file.Cases)

	for _, tc := range file.Cases {
		t.Run(tc.ID, func(t *testing.T) {
			dir := t.TempDir()
			metaPath := filepath.Join(dir, "metadata.json")
			metaBytes, err := json.Marshal(map[string]any{"policy": tc.Policy})
			require.NoError(t, err)
			require.NoError(t, os.WriteFile(metaPath, metaBytes, 0o644))

			switch tc.Check {
			case "load_metadata":
				cfg, ok := LoadPolicyFromMetadata(metaPath)
				require.True(t, ok)
				assert.InDelta(t, tc.Want["ml_threshold"], cfg.MLThreshold, 1e-9)
				assert.InDelta(t, tc.Want["residential_proxy_floor"], cfg.ResidentialProxyFloor, 1e-9)
			case "resolve_auto":
				for key, value := range tc.Env {
					t.Setenv(key, value)
				}
				envCfg := PolicyConfigFromEnv()
				resolved := ResolvePolicyConfig(envCfg, metaPath, "auto")
				assert.InDelta(t, tc.Want["ml_threshold"], resolved.MLThreshold, 1e-9)
				assert.InDelta(t, tc.Want["fp_guard_cap"], resolved.FPGuardCap, 1e-9)
			default:
				t.Fatalf("unknown check %q", tc.Check)
			}
		})
	}
}
