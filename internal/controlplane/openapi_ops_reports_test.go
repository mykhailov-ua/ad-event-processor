package controlplane_test

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"ad-event-processor/internal/controlplane"

	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v3"
)

func TestOpenAPI_dataFreshnessSchemaKeys(t *testing.T) {
	root := repoRoot(t)
	schemaPath := filepath.Join(root, "api/openapi/components/schemas/ops_reports.yaml")
	raw, err := os.ReadFile(schemaPath)
	require.NoError(t, err)

	var schemas map[string]map[string]any
	require.NoError(t, yaml.Unmarshal(raw, &schemas))
	props, ok := schemas["DataFreshness"]["properties"].(map[string]any)
	require.True(t, ok)

	var dto controlplane.DataFreshnessDTO
	sample, err := json.Marshal(dto)
	require.NoError(t, err)
	var got map[string]any
	require.NoError(t, json.Unmarshal(sample, &got))
	for key := range got {
		_, inSpec := props[key]
		require.True(t, inSpec, "DataFreshnessDTO json field %q missing from OpenAPI schema", key)
	}
}

func TestOpenAPI_placementReportRowSchemaKeys(t *testing.T) {
	root := repoRoot(t)
	schemaPath := filepath.Join(root, "api/openapi/components/schemas/ops_reports.yaml")
	raw, err := os.ReadFile(schemaPath)
	require.NoError(t, err)

	var schemas map[string]map[string]any
	require.NoError(t, yaml.Unmarshal(raw, &schemas))
	props, ok := schemas["PlacementReportRow"]["properties"].(map[string]any)
	require.True(t, ok)

	var dto controlplane.PlacementReportRowDTO
	sample, err := json.Marshal(dto)
	require.NoError(t, err)
	var got map[string]any
	require.NoError(t, json.Unmarshal(sample, &got))
	for key := range got {
		_, inSpec := props[key]
		require.True(t, inSpec, "PlacementReportRowDTO json field %q missing from OpenAPI schema", key)
	}
}

func TestOpenAPI_savedViewSchemaKeys(t *testing.T) {
	root := repoRoot(t)
	schemaPath := filepath.Join(root, "api/openapi/components/schemas/ops_reports.yaml")
	raw, err := os.ReadFile(schemaPath)
	require.NoError(t, err)

	var schemas map[string]map[string]any
	require.NoError(t, yaml.Unmarshal(raw, &schemas))
	props, ok := schemas["SavedView"]["properties"].(map[string]any)
	require.True(t, ok)

	var dto controlplane.SavedViewDTO
	sample, err := json.Marshal(dto)
	require.NoError(t, err)
	var got map[string]any
	require.NoError(t, json.Unmarshal(sample, &got))
	for key := range got {
		_, inSpec := props[key]
		require.True(t, inSpec, "SavedViewDTO json field %q missing from OpenAPI schema", key)
	}
}

func TestOpenAPI_doctorSummarySchemaKeys(t *testing.T) {
	root := repoRoot(t)
	schemaPath := filepath.Join(root, "api/openapi/components/schemas/ops_reports.yaml")
	raw, err := os.ReadFile(schemaPath)
	require.NoError(t, err)

	var schemas map[string]map[string]any
	require.NoError(t, yaml.Unmarshal(raw, &schemas))
	props, ok := schemas["DoctorSummary"]["properties"].(map[string]any)
	require.True(t, ok)

	var dto controlplane.DoctorResponseDTO
	sample, err := json.Marshal(dto)
	require.NoError(t, err)
	var got map[string]any
	require.NoError(t, json.Unmarshal(sample, &got))
	for key := range got {
		_, inSpec := props[key]
		require.True(t, inSpec, "DoctorResponseDTO json field %q missing from OpenAPI schema", key)
	}
}

func TestOpenAPI_reportJobStatusSchemaKeys(t *testing.T) {
	root := repoRoot(t)
	schemaPath := filepath.Join(root, "api/openapi/components/schemas/ops_reports.yaml")
	raw, err := os.ReadFile(schemaPath)
	require.NoError(t, err)

	var schemas map[string]map[string]any
	require.NoError(t, yaml.Unmarshal(raw, &schemas))
	props, ok := schemas["ReportJobStatus"]["properties"].(map[string]any)
	require.True(t, ok)

	var dto controlplane.ReportJobStatusDTO
	sample, err := json.Marshal(dto)
	require.NoError(t, err)
	var got map[string]any
	require.NoError(t, json.Unmarshal(sample, &got))
	for key := range got {
		_, inSpec := props[key]
		require.True(t, inSpec, "ReportJobStatusDTO json field %q missing from OpenAPI schema", key)
	}
}
