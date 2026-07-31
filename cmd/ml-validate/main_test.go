package main

import (
	"path/filepath"
	"runtime"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestValidateModelAndFixtures(t *testing.T) {
	root := repoRoot(t)
	modelPath := filepath.Join(root, "internal", "fraudscoring", "testdata", "model.txt")
	fixturesDir := filepath.Join(root, "testdata", "ml")

	scorer, err := validateModel(modelPath)
	require.NoError(t, err)
	require.Equal(t, 7, scorer.Dims())

	require.NoError(t, validateFixtures(scorer, fixturesDir))
}

func repoRoot(t *testing.T) string {
	t.Helper()
	_, file, _, ok := runtime.Caller(0)
	require.True(t, ok)
	return filepath.Clean(filepath.Join(filepath.Dir(file), "..", ".."))
}
