package edge

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPinnedMapPathJoins(t *testing.T) {
	got := PinnedMapPath("/sys/fs/bpf/ad-event-processor", MapBlocklistV4)
	assert.Equal(t, "/sys/fs/bpf/ad-event-processor/blocklist_v4", got)
}

func TestResolvePinnedMapPathsDefault(t *testing.T) {
	t.Setenv("BPF_PIN_DIR", "")
	t.Setenv("BPF_BLOCKLIST_MAP", "")
	t.Setenv("BPF_ALLOWLIST_MAP", "")
	t.Setenv("BPF_STATS_MAP", "")
	t.Setenv("BPF_VIOLATIONS_MAP", "")
	t.Setenv("BPF_FINGERPRINTS_MAP", "")

	paths := ResolvePinnedMapPaths()
	assert.Equal(t, DefaultBlocklistMapPath, paths.Blocklist)
	assert.Equal(t, DefaultAllowlistMapPath, paths.Allowlist)
	assert.Equal(t, DefaultStatsMapPath, paths.Stats)
	assert.Equal(t, DefaultViolationsMapPath, paths.Violations)
	assert.Equal(t, DefaultFingerprintsMapPath, paths.Fingerprints)
}

func TestResolvePinnedMapPathsCustomPinDir(t *testing.T) {
	custom := "/sys/fs/bpf/custom-edge"
	t.Setenv("BPF_PIN_DIR", custom)
	t.Setenv("BPF_BLOCKLIST_MAP", "")
	t.Setenv("BPF_ALLOWLIST_MAP", "")
	t.Setenv("BPF_STATS_MAP", "")
	t.Setenv("BPF_VIOLATIONS_MAP", "")
	t.Setenv("BPF_FINGERPRINTS_MAP", "")

	paths := ResolvePinnedMapPaths()
	assert.Equal(t, filepath.Join(custom, MapBlocklistV4), paths.Blocklist)
	assert.Equal(t, filepath.Join(custom, MapAllowV4), paths.Allowlist)
}

func TestResolvePinnedMapPathsExplicitMapOverride(t *testing.T) {
	override := "/var/run/edge/blocklist_v4"
	t.Setenv("BPF_PIN_DIR", "/sys/fs/bpf/ignored")
	t.Setenv("BPF_BLOCKLIST_MAP", override)

	paths := ResolvePinnedMapPaths()
	assert.Equal(t, override, paths.Blocklist)
}

func TestEntrypointUsesCanonicalPinDir(t *testing.T) {
	root, err := os.Getwd()
	require.NoError(t, err)
	for root != "/" && !fileExists(filepath.Join(root, "go.mod")) {
		root = filepath.Dir(root)
	}
	entrypoint := filepath.Join(root, "deploy", "edge", "xdp", "entrypoint.sh")
	require.FileExists(t, entrypoint)

	data, err := os.ReadFile(entrypoint)
	require.NoError(t, err)
	body := string(data)
	assert.Contains(t, body, "BPF_PIN_DIR")
	assert.Contains(t, body, DefaultBPFPinDir)
	assert.NotContains(t, body, "/sys/fs/bpf/ad-event-processor")
}

func fileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}
