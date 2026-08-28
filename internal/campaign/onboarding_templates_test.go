package campaign

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestListOnboardingTemplates_containsBundledKeys(t *testing.T) {
	resetOnboardingCatalogForTest()
	templates, err := ListOnboardingTemplates()
	require.NoError(t, err)
	keys := make(map[string]struct{}, len(templates))
	for _, row := range templates {
		keys[row.Key] = struct{}{}
		require.NotEmpty(t, row.Title)
		require.NotEmpty(t, row.TrafficFamily)
		require.NotEmpty(t, row.IntegrationSchemaRefs)
		require.NotEmpty(t, row.DefaultFlow.FlowName)
	}
	for _, want := range []string{"meta_social_funnel", "popunder_propeller", "push_house_funnel", "native_mgid_funnel"} {
		_, ok := keys[want]
		assert.True(t, ok, "missing template %s", want)
	}
}

func TestOnboardingTemplateKeys_matchOpenAPIEnum(t *testing.T) {
	resetOnboardingCatalogForTest()
	keys, err := OnboardingTemplateKeys()
	require.NoError(t, err)
	raw, err := os.ReadFile(onboardingOpenAPISchemaPath(t))
	require.NoError(t, err)
	for _, key := range keys {
		assert.Contains(t, string(raw), key)
	}
}

func TestApplyOnboardingTemplate_unknownKey_holdout(t *testing.T) {
	resetOnboardingCatalogForTest()
	_, err := ApplyOnboardingTemplate("missing_template")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "meta_social_funnel")
}

func onboardingOpenAPISchemaPath(t *testing.T) string {
	t.Helper()
	_, filename, _, ok := runtime.Caller(0)
	require.True(t, ok)
	root := filepath.Join(filepath.Dir(filename), "..", "..")
	path := filepath.Join(root, "api", "openapi", "components", "schemas", "campaign.yaml")
	if _, err := os.Stat(path); err != nil {
		t.Fatalf("openapi schema path: %v", err)
	}
	return path
}

func TestOnboardingTemplate_catalogHasNoSecrets(t *testing.T) {
	resetOnboardingCatalogForTest()
	templates, err := ListOnboardingTemplates()
	require.NoError(t, err)
	for _, row := range templates {
		for _, ref := range row.IntegrationSchemaRefs {
			lower := strings.ToLower(ref)
			assert.NotContains(t, lower, "password")
			assert.NotContains(t, lower, "token")
		}
	}
}
