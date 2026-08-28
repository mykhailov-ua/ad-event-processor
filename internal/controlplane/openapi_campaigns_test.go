package controlplane_test

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"ad-event-processor/internal/brand"
	"ad-event-processor/internal/campaign"

	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v3"
)

func TestOpenAPI_campaignSchemaKeys(t *testing.T) {
	root := repoRoot(t)
	schemaPath := filepath.Join(root, "api/openapi/components/schemas/campaign.yaml")
	raw, err := os.ReadFile(schemaPath)
	require.NoError(t, err)

	var schemas map[string]map[string]any
	require.NoError(t, yaml.Unmarshal(raw, &schemas))
	props, ok := schemas["Campaign"]["properties"].(map[string]any)
	require.True(t, ok)

	var dto campaign.CampaignDTO
	sample, err := json.Marshal(dto)
	require.NoError(t, err)
	var got map[string]any
	require.NoError(t, json.Unmarshal(sample, &got))
	for key := range got {
		_, inSpec := props[key]
		require.True(t, inSpec, "campaign.CampaignDTO json field %q missing from OpenAPI schema", key)
	}
}

func TestOpenAPI_patchCampaignRequestKeys(t *testing.T) {
	root := repoRoot(t)
	schemaPath := filepath.Join(root, "api/openapi/components/schemas/campaign.yaml")
	raw, err := os.ReadFile(schemaPath)
	require.NoError(t, err)

	var schemas map[string]map[string]any
	require.NoError(t, yaml.Unmarshal(raw, &schemas))
	props, ok := schemas["PatchCampaignRequest"]["properties"].(map[string]any)
	require.True(t, ok)

	var dto campaign.PatchCampaignRequest
	sample, err := json.Marshal(dto)
	require.NoError(t, err)
	var got map[string]any
	require.NoError(t, json.Unmarshal(sample, &got))
	for key := range got {
		_, inSpec := props[key]
		require.True(t, inSpec, "campaign.PatchCampaignRequest json field %q missing from OpenAPI schema", key)
	}
}

func TestOpenAPI_importValidateJobRequestKeys(t *testing.T) {
	root := repoRoot(t)
	schemaPath := filepath.Join(root, "api/openapi/components/schemas/campaign.yaml")
	raw, err := os.ReadFile(schemaPath)
	require.NoError(t, err)

	var schemas map[string]map[string]any
	require.NoError(t, yaml.Unmarshal(raw, &schemas))
	props, ok := schemas["ImportValidateJobRequest"]["properties"].(map[string]any)
	require.True(t, ok)

	var dto campaign.ImportValidateJobRequest
	sample, err := json.Marshal(dto)
	require.NoError(t, err)
	var got map[string]any
	require.NoError(t, json.Unmarshal(sample, &got))
	for key := range got {
		_, inSpec := props[key]
		require.True(t, inSpec, "campaign.ImportValidateJobRequest json field %q missing from OpenAPI schema", key)
	}
}

func TestOpenAPI_brandCreativeSchemaKeys(t *testing.T) {
	root := repoRoot(t)
	schemaPath := filepath.Join(root, "api/openapi/components/schemas/campaign.yaml")
	raw, err := os.ReadFile(schemaPath)
	require.NoError(t, err)

	var schemas map[string]map[string]any
	require.NoError(t, yaml.Unmarshal(raw, &schemas))
	props, ok := schemas["BrandCreative"]["properties"].(map[string]any)
	require.True(t, ok)

	var dto brand.CreativeDTO
	sample, err := json.Marshal(dto)
	require.NoError(t, err)
	var got map[string]any
	require.NoError(t, json.Unmarshal(sample, &got))
	for key := range got {
		_, inSpec := props[key]
		require.True(t, inSpec, "BrandCreativeDTO json field %q missing from OpenAPI schema", key)
	}
}
