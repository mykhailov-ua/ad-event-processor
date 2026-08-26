package controlplane_test

import (
	"encoding/json"
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"ad-event-processor/internal/controlplane"

	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v3"
)

func repoRoot(t *testing.T) string {
	t.Helper()
	_, file, _, ok := runtime.Caller(0)
	require.True(t, ok)
	return filepath.Clean(filepath.Join(filepath.Dir(file), "../.."))
}

func TestOpenAPI_costSyncCredentialSchemaKeys(t *testing.T) {
	root := repoRoot(t)
	schemaPath := filepath.Join(root, "api/openapi/components/schemas/cost_sync.yaml")
	raw, err := os.ReadFile(schemaPath)
	require.NoError(t, err)

	var schemas map[string]map[string]any
	require.NoError(t, yaml.Unmarshal(raw, &schemas))
	credSchema, ok := schemas["CostSyncCredential"]
	require.True(t, ok)
	props, ok := credSchema["properties"].(map[string]any)
	require.True(t, ok)

	var dto controlplane.CostSyncCredentialDTO
	sample, err := json.Marshal(dto)
	require.NoError(t, err)
	var got map[string]any
	require.NoError(t, json.Unmarshal(sample, &got))

	for key := range got {
		if key == "token_expires_at" {
			continue
		}
		_, inSpec := props[key]
		require.True(t, inSpec, "CostSyncCredentialDTO json field %q missing from OpenAPI schema", key)
	}
}
