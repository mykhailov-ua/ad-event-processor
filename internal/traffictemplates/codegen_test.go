package traffictemplates

import (
	"os"
	"path/filepath"
	"testing"

	"ad-event-processor/internal/integrationschema"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGenerate_coversCatalogAndCuratedMeta(t *testing.T) {
	root := findRepoRoot(t)
	templates, err := Generate(
		integrationschema.SchemaRootDir(),
		filepath.Join(root, "deploy", "vendor", "traffic_source_ui.yaml"),
	)
	require.NoError(t, err)
	require.GreaterOrEqual(t, len(templates), CountBundledTrafficSchemas()+1)

	var meta *Template
	slugSeen := map[string]struct{}{}
	for i := range templates {
		if templates[i].ID == "meta-facebook" {
			meta = &templates[i]
		}
		if slug := templates[i].BundledSlug; slug != "" {
			slugSeen[slug] = struct{}{}
		}
	}
	require.NotNil(t, meta)
	assert.Equal(t, "meta", meta.CostSync)
	assert.Equal(t, "{{campaign.id}}", paramValue(meta.Params, "sub2"))

	assert.GreaterOrEqual(t, len(slugSeen), CountBundledTrafficSchemas())
}

func paramValue(params []Param, key string) string {
	for _, row := range params {
		if row.Key == key {
			return row.Value
		}
	}
	return ""
}

func findRepoRoot(t *testing.T) string {
	t.Helper()
	dir, err := os.Getwd()
	require.NoError(t, err)
	for {
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			return dir
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			t.Fatal("repo root not found")
		}
		dir = parent
	}
}
