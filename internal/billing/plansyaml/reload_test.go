package plansyaml_test

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"espx/internal/billing/plansyaml"

	"github.com/stretchr/testify/require"
)

func TestLoadPlansYAML(t *testing.T) {
	path := filepath.Join("..", "..", "..", "deploy", "operator", "plans.yaml")
	doc, err := plansyaml.Load(path)
	require.NoError(t, err)
	require.NotEmpty(t, doc.Plans)
	require.Equal(t, "network_pro", doc.Plans[0].Code)
}

func TestReloadDryRun(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "plans.yaml")
	require.NoError(t, os.WriteFile(path, []byte(`plans:
  - code: smoke_plan
    display_name: Smoke
    base_fee_micro: 0
    limits:
      max_rps: 100
    features:
      rtb_live: false
assignments: []
`), 0o600))

	report, err := plansyaml.Reload(context.Background(), nil, path, true, nil)
	require.NoError(t, err)
	require.True(t, report.DryRun)
	require.Equal(t, 1, report.PlansUpserted)
}
