package openapi_test

import (
	"path/filepath"
	"runtime"
	"testing"

	"ad-event-processor/internal/openapi"

	"github.com/stretchr/testify/require"
)

func repoRoot(t *testing.T) string {
	t.Helper()
	_, file, _, ok := runtime.Caller(0)
	require.True(t, ok)
	return filepath.Clean(filepath.Join(filepath.Dir(file), "../.."))
}

func TestAssertCatalogParity(t *testing.T) {
	root := repoRoot(t)
	require.NoError(t, openapi.AssertCatalogParity(root))
}

func TestDocumentedRoutes_inCatalog(t *testing.T) {
	root := repoRoot(t)
	specRoutes, err := openapi.RouteKeysFromSpecUnion(root)
	require.NoError(t, err)
	for key := range openapi.DocumentedRoutes {
		_, ok := specRoutes[key]
		require.True(t, ok, "documented route missing from spec union: %s", key)
	}
}

func TestExport_idempotent(t *testing.T) {
	root := repoRoot(t)
	require.NoError(t, openapi.Export(root))
	require.NoError(t, openapi.Export(root))
	require.NoError(t, openapi.AssertCatalogParity(root))
}
