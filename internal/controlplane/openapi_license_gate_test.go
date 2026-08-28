package controlplane_test

import (
	"ad-event-processor/internal/licensingadmin"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v3"
)

func TestOpenAPI_licenseFeatureRequiredSchemaKeys(t *testing.T) {
	root := repoRoot(t)
	schemaPath := filepath.Join(root, "api/openapi/components/schemas/errors.yaml")
	raw, err := os.ReadFile(schemaPath)
	require.NoError(t, err)

	var schemas map[string]map[string]any
	require.NoError(t, yaml.Unmarshal(raw, &schemas))
	props, ok := schemas["LicenseFeatureRequiredBody"]["properties"].(map[string]any)
	require.True(t, ok)

	var dto licensingadmin.LicenseFeatureRequiredBody
	sample, err := json.Marshal(dto)
	require.NoError(t, err)
	var got map[string]any
	require.NoError(t, json.Unmarshal(sample, &got))
	for key := range got {
		_, inSpec := props[key]
		require.True(t, inSpec, "licensingadmin.LicenseFeatureRequiredBody json field %q missing from OpenAPI schema", key)
	}
}
