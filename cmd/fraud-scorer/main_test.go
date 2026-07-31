package main

import (
	"context"
	"encoding/json"
	"os"
	"testing"

	"espx/internal/database"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestScanAndRegister(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test in short mode")
	}

	pool, cleanup := database.SetupTestDB(t)
	defer cleanup()

	ctx := context.Background()

	tmpDir, err := os.MkdirTemp("", "ml-artifacts-test")
	require.NoError(t, err)
	defer os.RemoveAll(tmpDir)

	origWd, err := os.Getwd()
	require.NoError(t, err)
	defer func() { _ = os.Chdir(origWd) }()

	err = os.Chdir(tmpDir)
	require.NoError(t, err)

	err = os.MkdirAll("var/fraudscore/artifacts", 0755)
	require.NoError(t, err)

	modelContent := "test model content"
	err = os.WriteFile("var/fraudscore/artifacts/model.txt", []byte(modelContent), 0644)
	require.NoError(t, err)

	meta := struct {
		Version string             `json:"version"`
		Metrics map[string]float64 `json:"metrics"`
	}{
		Version: "vTest123",
		Metrics: map[string]float64{
			"accuracy": 0.99,
		},
	}

	metaBytes, err := json.Marshal(meta)
	require.NoError(t, err)
	err = os.WriteFile("var/fraudscore/artifacts/metadata.json", metaBytes, 0644)
	require.NoError(t, err)

	err = scanAndRegister(ctx, pool)
	require.NoError(t, err)

	var id, status, artifactHash string
	var metadata map[string]interface{}
	err = pool.QueryRow(ctx, "SELECT id, status, artifact_hash, metrics_json FROM ml_model_versions WHERE id = 'vTest123'").Scan(&id, &status, &artifactHash, &metadata)
	require.NoError(t, err)

	assert.Equal(t, "vTest123", id)
	assert.Equal(t, "SYNCING", status)
	assert.NotEmpty(t, artifactHash)
	assert.Equal(t, "vTest123", metadata["version"])
	nestedMetrics, ok := metadata["metrics"].(map[string]interface{})
	require.True(t, ok)
	assert.Equal(t, 0.99, nestedMetrics["accuracy"])

	err = scanAndRegister(ctx, pool)
	require.NoError(t, err)
}
