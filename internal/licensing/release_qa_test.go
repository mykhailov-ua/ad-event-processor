package licensing

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

const (
	HarnessReleaseQAFuzzSmoke      = "release_qa_fuzz_smoke"
	HarnessReleaseQAGarbledAlloc   = "release_qa_garbled_alloc"
	HarnessReleaseQAGarbledRedTeam = "release_qa_garbled_red_team"
)

func repoRoot(t *testing.T) string {
	t.Helper()
	_, file, _, ok := runtime.Caller(0)
	require.True(t, ok)
	return filepath.Clean(filepath.Join(filepath.Dir(file), "..", ".."))
}

func TestReleaseQA_nightlyFuzzWorkflowDocumentsTargets(t *testing.T) {
	t.Parallel()
	path := filepath.Join(repoRoot(t), ".github", "workflows", "license-fuzz-nightly.yaml")
	raw, err := os.ReadFile(path)
	require.NoError(t, err, "missing nightly workflow")
	body := string(raw)
	for _, target := range []string{
		"FuzzVerifyJWT",
		"FuzzDecodeUnverified",
		"FuzzJSONClaims",
	} {
		require.Contains(t, body, target, "nightly workflow must fuzz %s", target)
	}
	require.Contains(t, body, "fuzztime=10m")
	require.Contains(t, body, "fuzztime=5m")
	require.True(t, strings.Contains(body, "go-version: '1.25") || strings.Contains(body, "go-version: \"1.25"),
		"nightly workflow must pin Go 1.25.x")
}

func TestReleaseQA_harnessLabels_registered(t *testing.T) {
	t.Parallel()
	for _, label := range []string{
		HarnessReleaseQAFuzzSmoke,
		HarnessReleaseQAGarbledAlloc,
		HarnessReleaseQAGarbledRedTeam,
	} {
		require.NotEmpty(t, label)
		require.NotContains(t, label, " ")
	}
}
