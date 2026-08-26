package openapi_test

import (
	"bytes"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

const oasdiffModule = "github.com/oasdiff/oasdiff@v1.29.1"

func runOasdiffBreaking(t *testing.T, base, revision string, extraArgs ...string) (string, error) {
	t.Helper()
	root := repoRoot(t)
	args := []string{"run", oasdiffModule, "breaking", "--fail-on", "ERR"}
	args = append(args, extraArgs...)
	args = append(args, base, revision)
	cmd := exec.Command("go", args...)
	cmd.Dir = root
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	err := cmd.Run()
	out := strings.TrimSpace(stdout.String() + "\n" + stderr.String())
	return out, err
}

func TestBreakingChangeGate_fixtureDetectsRemovedProperty(t *testing.T) {
	if testing.Short() {
		t.Skip("integration: breaking gate fixture runs oasdiff via go run")
	}
	root := repoRoot(t)
	fixtureDir := filepath.Join(root, "internal/openapi/testdata/breaking")
	base := filepath.Join(fixtureDir, "base.yaml")
	head := filepath.Join(fixtureDir, "head_removed_field.yaml")

	out, err := runOasdiffBreaking(t, base, head)
	require.Error(t, err, "expected breaking diff output: %s", out)
	require.Contains(t, strings.ToLower(out), "removed")
}

func TestBreakingChangeGate_fixtureIdenticalSpecsPass(t *testing.T) {
	if testing.Short() {
		t.Skip("integration: breaking gate fixture runs oasdiff via go run")
	}
	root := repoRoot(t)
	base := filepath.Join(root, "internal/openapi/testdata/breaking/base.yaml")

	_, err := runOasdiffBreaking(t, base, base)
	require.NoError(t, err)
}

func TestBreakingChangeGate_errIgnoreFilePresent(t *testing.T) {
	root := repoRoot(t)
	path := filepath.Join(root, "api/openapi/breaking_err_ignore.txt")
	_, err := os.Stat(path)
	require.NoError(t, err)
}
