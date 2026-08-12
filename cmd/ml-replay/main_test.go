package main

import (
	"bytes"
	"context"
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/bidshard/ad-event-processor/internal/fraud"

	"github.com/stretchr/testify/require"
)

func TestReplayFixturesCSV(t *testing.T) {
	root := repoRoot(t)
	modelPath := filepath.Join(root, "var", "fraudscore", "artifacts", "model.txt")
	if _, err := os.Stat(modelPath); err != nil {
		t.Skip("fraud model not found; run make fraudtrain-check locally")
	}
	fixturesDir := filepath.Join(root, "var", "fraudscore", "fixtures")
	if _, err := os.Stat(fixturesDir); err != nil {
		t.Skip("fixtures not found; run make fraudtrain-check locally")
	}
	opts := replayOptions{
		modelPath:   modelPath,
		fixturesDir: fixturesDir,
	}

	rows, err := loadFixtureRows(opts.fixturesDir)
	require.NoError(t, err)
	require.NotEmpty(t, rows)

	scorer, err := loadScorer(opts.modelPath)
	require.NoError(t, err)

	featureRows := make([]fraud.FeatureRow, len(rows))
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
		t.Skip("fraud model not found; run make fraudtrain-check locally")
	}
	fixturesDir := filepath.Join(root, "var", "fraudscore", "fixtures")
	if _, err := os.Stat(fixturesDir); err != nil {
		t.Skip("fixtures not found; run make fraudtrain-check locally")
	}
	opts := replayOptions{
		modelPath:   modelPath,
		fixturesDir: fixturesDir,
	}
	require.NoError(t, runReplay(context.Background(), opts))
}

func repoRoot(t *testing.T) string {
	t.Helper()
	_, file, _, ok := runtime.Caller(0)
	require.True(t, ok)
	return filepath.Clean(filepath.Join(filepath.Dir(file), "..", ".."))
}
