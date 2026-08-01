package openapi_test

import (
	"path/filepath"
	"runtime"
	"testing"

	"espx/internal/openapi"

	"github.com/stretchr/testify/require"
)

func TestDocumentedRoutes_excludesStubs(t *testing.T) {
	root := repoRoot(t)
	all, err := openapi.DiscoverAPIV1Routes(root)
	require.NoError(t, err)
	doc, err := openapi.DocumentedRoutes(root)
	require.NoError(t, err)
	require.LessOrEqual(t, len(doc), len(all))
	for _, r := range doc {
		require.False(t, openapi.IsStub(r.Method, r.Path), r.Key())
	}
}

func repoRoot(t *testing.T) string {
	t.Helper()
	_, file, _, ok := runtime.Caller(0)
	require.True(t, ok)
	return filepath.Clean(filepath.Join(filepath.Dir(file), "..", ".."))
}
