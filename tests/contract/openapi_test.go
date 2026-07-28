package openapi_test

import (
	"encoding/json"
	"path/filepath"
	"runtime"
	"testing"

	"espx/internal/openapi"

	"github.com/stretchr/testify/require"
)

func repoRoot(t *testing.T) string {
	t.Helper()
	_, file, _, ok := runtime.Caller(0)
	require.True(t, ok)
	return filepath.Clean(filepath.Join(filepath.Dir(file), "..", ".."))
}

func TestDiscoverAPIV1Routes_nonEmpty(t *testing.T) {
	routes, err := openapi.DiscoverAPIV1Routes(repoRoot(t))
	require.NoError(t, err)
	require.Greater(t, len(routes), 40)
}

func TestOpenAPISpec_coversDocumentedRoutes(t *testing.T) {
	root := repoRoot(t)
	want, err := openapi.DocumentedRoutes(root)
	require.NoError(t, err)

	doc, err := openapi.LoadSpec(root)
	require.NoError(t, err)

	got := openapi.SpecRoutes(doc)
	gotSet := make(map[string]struct{}, len(got))
	for _, r := range got {
		gotSet[r.Key()] = struct{}{}
	}

	var missing []string
	for _, r := range want {
		key := r.Method + " " + openapi.OpenAPIPath(r.Path)
		if _, ok := gotSet[key]; !ok {
			missing = append(missing, key)
		}
	}
	require.Empty(t, missing, "openapi.yaml missing routes:\n%s", joinLines(missing))

	var extra []string
	wantSet := make(map[string]struct{}, len(want))
	for _, r := range want {
		wantSet[r.Method+" "+openapi.OpenAPIPath(r.Path)] = struct{}{}
	}
	for _, r := range got {
		key := r.Key()
		if _, ok := wantSet[key]; !ok {
			extra = append(extra, key)
		}
	}
	require.Empty(t, extra, "openapi.yaml documents routes not registered in code:\n%s", joinLines(extra))
}

func TestOpenAPISpec_noAdminHTMLRoutes(t *testing.T) {
	doc, err := openapi.LoadSpec(repoRoot(t))
	require.NoError(t, err)
	require.Empty(t, openapi.HasAdminHTMLPaths(doc))
}

func TestOpenAPISpec_securitySchemes(t *testing.T) {
	doc, err := openapi.LoadSpec(repoRoot(t))
	require.NoError(t, err)
	names := openapi.SecuritySchemeNames(doc)
	require.Contains(t, names, "AdminAPIKey")
	require.Contains(t, names, "SessionCookie")
	require.Contains(t, names, "ConsentHMAC")
}

func TestOpenAPISpec_sampleSchemas_snakeCase(t *testing.T) {
	root := repoRoot(t)
	doc, err := openapi.LoadSpec(root)
	require.NoError(t, err)

	components, ok := doc.Paths["/api/v1/customers/{id}/balance"].(map[string]any)
	require.True(t, ok)
	getOp, ok := components["get"].(map[string]any)
	require.True(t, ok)
	responses, ok := getOp["responses"].(map[string]any)
	require.True(t, ok)
	ok200, ok := responses["200"].(map[string]any)
	require.True(t, ok)
	content, ok := ok200["content"].(map[string]any)
	require.True(t, ok)
	appJSON, ok := content["application/json"].(map[string]any)
	require.True(t, ok)
	schema, ok := appJSON["schema"].(map[string]any)
	require.True(t, ok)
	ref, ok := schema["$ref"].(string)
	require.True(t, ok)
	require.Equal(t, "#/components/schemas/CustomerBalance", ref)
}

func TestOpenAPISpec_customerBalanceSampleJSON(t *testing.T) {
	sample := `{
		"customer_id": "018f3c2a-7b2a-7f3c-8b2a-7f3c8b2a7f3c",
		"balance": "50.000000",
		"currency": "USD",
		"ledger": [{
			"id": 1,
			"customer_id": "018f3c2a-7b2a-7f3c-8b2a-7f3c8b2a7f3c",
			"amount": "50.000000",
			"type": "PAYMENT_TOPUP",
			"created_at": "2026-01-15T12:00:00Z"
		}]
	}`
	var payload map[string]any
	require.NoError(t, json.Unmarshal([]byte(sample), &payload))
	require.Contains(t, payload, "customer_id")
	require.Contains(t, payload, "balance")
	require.Contains(t, payload, "currency")
	require.Contains(t, payload, "ledger")
	ledger, ok := payload["ledger"].([]any)
	require.True(t, ok)
	require.Len(t, ledger, 1)
	row, ok := ledger[0].(map[string]any)
	require.True(t, ok)
	require.Contains(t, row, "amount")
	require.Contains(t, row, "type")
}

func joinLines(lines []string) string {
	if len(lines) == 0 {
		return ""
	}
	out := lines[0]
	for i := 1; i < len(lines); i++ {
		out += "\n" + lines[i]
	}
	return out
}
