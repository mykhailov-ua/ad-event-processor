package ingest

import (
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestSafePageParityDrill_holdoutPolicyBreach(t *testing.T) {
	root := repoRootForParityDrill(t)
	script := filepath.Join(root, "scripts", "test", "edge", "safe_page_parity_drill.sh")
	require.FileExists(t, script)

	cmd := exec.Command("bash", script, "--holdout")
	cmd.Dir = root
	out, err := cmd.CombinedOutput()
	require.NoError(t, err, "holdout drill failed: %s", out)
	require.Contains(t, string(out), "fault_proof fault=safe_zone_parity")
}

func repoRootForParityDrill(t *testing.T) string {
	t.Helper()
	_, file, _, ok := runtime.Caller(0)
	require.True(t, ok)
	dir := filepath.Dir(file)
	for i := 0; i < 6; i++ {
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			return dir
		}
		dir = filepath.Dir(dir)
	}
	t.Fatal("go.mod not found from test file")
	return ""
}
