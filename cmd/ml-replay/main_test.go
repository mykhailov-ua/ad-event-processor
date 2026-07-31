package main

import (
	"bytes"
	"context"
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"espx/internal/fraudscoring"

	"github.com/stretchr/testify/require"
)

func TestReplayFixturesCSV(t *testing.T) {
	root := repoRoot(t)
	modelPath := filepath.Join(root, "var", "fraudscore", "artifacts", "model.txt")
	if _, err := os.Stat(modelPath); err != nil {
		t.Skip("fraud model not found; run make fraud-modeling-check locally")
	}
	opts := replayOptions{
		modelPath:   modelPath,
		fixturesDir: filepath.Join(root, "testdata", "ml"),
	}

	rows, err := loadFixtureRows(opts.fixturesDir)
	require.NoError(t, err)
	require.NotEmpty(t, rows)

	scorer, err := loadScorer(opts.modelPath)
	require.NoError(t, err)

	featureRows := make([]fraudscoring.FeatureRow, len(rows))
	for i := range rows {
		featureRows[i] = rows[i].FeatureRow
	}
	scores, err := scorer.ScoreBatch(context.Background(), featureRows)
	require.NoError(t, err)
	require.Len(t, scores, len(rows))

	var buf bytes.Buffer
	require.NoError(t, writeReplayCSV(&buf, rows, scores))
	require.Contains(t, buf.String(), "ml_score")
	require.Contains(t, buf.String(), "tier")
}

func TestRunReplayFixtures(t *testing.T) {
	root := repoRoot(t)
	modelPath := filepath.Join(root, "var", "fraudscore", "artifacts", "model.txt")
	if _, err := os.Stat(modelPath); err != nil {
		t.Skip("fraud model not found; run make fraud-modeling-check locally")
	}
	opts := replayOptions{
		modelPath:   modelPath,
		fixturesDir: filepath.Join(root, "testdata", "ml"),
	}
	require.NoError(t, runReplay(context.Background(), opts))
}

func repoRoot(t *testing.T) string {
	t.Helper()
	_, file, _, ok := runtime.Caller(0)
	require.True(t, ok)
	return filepath.Clean(filepath.Join(filepath.Dir(file), "..", ".."))
}
