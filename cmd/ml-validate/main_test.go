package main

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestValidateModelAndFixtures(t *testing.T) {
	root := repoRoot(t)
	modelPath := filepath.Join(root, "var", "fraudscore", "artifacts", "model.txt")
	fixturesDir := filepath.Join(root, "testdata", "ml")

	if _, err := os.Stat(modelPath); err != nil {
		t.Skip("fraud model not found; run make fraud-modeling-check locally")
	}

	require.NoError(t, checkModel(modelPath))

	require.NoError(t, validateFixtures(fixturesDir))
}

func repoRoot(t *testing.T) string {
	t.Helper()
	_, file, _, ok := runtime.Caller(0)
	require.True(t, ok)
	return filepath.Clean(filepath.Join(filepath.Dir(file), "..", ".."))
}
