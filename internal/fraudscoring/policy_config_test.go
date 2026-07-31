package fraudscoring

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLoadPolicyFromMetadata(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "metadata.json")
	err := os.WriteFile(path, []byte(`{
  "policy": {
    "tier_pass": 30,
    "tier_suspect": 60,
    "tier_ivt": 80,
    "tier_block": 100,
    "ml_threshold": 0.42,
    "residential_proxy_floor": 0.61,
    "residential_proxy_max_ml": 0.44,
    "fp_guard_cap": 0.78
  }
}`), 0o644)
	require.NoError(t, err)

	cfg, ok := LoadPolicyFromMetadata(path)
	require.True(t, ok)
	assert.InDelta(t, 0.42, cfg.MLThreshold, 1e-9)
	assert.InDelta(t, 0.61, cfg.ResidentialProxyFloor, 1e-9)
}

func TestResolvePolicyConfigAutoMergesEnv(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "metadata.json")
	err := os.WriteFile(path, []byte(`{
  "policy": {
    "tier_pass": 30,
    "tier_suspect": 60,
    "tier_ivt": 80,
    "tier_block": 100,
    "ml_threshold": 0.42,
    "fp_guard_cap": 0.78
  }
}`), 0o644)
	require.NoError(t, err)

	t.Setenv("FRAUD_POLICY_ML_THRESHOLD", "0.55")
	envCfg := PolicyConfigFromEnv()
	resolved := ResolvePolicyConfig(envCfg, path, "auto")
	assert.InDelta(t, 0.55, resolved.MLThreshold, 1e-9)
	assert.InDelta(t, 0.78, resolved.FPGuardCap, 1e-9)
}
