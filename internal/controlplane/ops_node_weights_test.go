package controlplane

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"espx/internal/config"
	"espx/internal/database"

	"github.com/stretchr/testify/require"
)

func TestOpsNodeWeights_ReturnsTrackerPeers(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test")
	}

	pool, cleanup := database.SetupTestDB(t)
	t.Cleanup(cleanup)

	ctx := context.Background()
	_, err := pool.Exec(ctx, `
		INSERT INTO node_capacity_scores (node_id, region_code, role, score, weight, provenance, epoch_id)
		VALUES
			('tracker-1', 1, 'tracker', 0.91, 0.25, 'own_window', 10),
			('tracker-2', 1, 'tracker', 0.78, 0.75, 'own_window', 10)
		ON CONFLICT (node_id, region_code, role) DO UPDATE SET
			score = EXCLUDED.score,
			weight = EXCLUDED.weight,
			provenance = EXCLUDED.provenance,
			epoch_id = EXCLUDED.epoch_id`)
	require.NoError(t, err)

	mux := http.NewServeMux()
	cfg := &config.Config{RegionCode: 1, MultiRegionEnabled: true}
	RegisterOpsRoutes(mux, pool, nil, cfg)

	req := httptest.NewRequest(http.MethodGet, "/ops/node-weights", nil)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)
	require.Equal(t, http.StatusOK, rec.Code)

	var resp OpsNodeWeightsResponse
	require.NoError(t, json.NewDecoder(rec.Body).Decode(&resp))
	require.Len(t, resp.Nodes, 2)
	require.Equal(t, 0, resp.Nodes[0].PeerIndex)
	require.Equal(t, 1, resp.Nodes[1].PeerIndex)
	require.InDelta(t, 0.25, resp.Nodes[0].Weight, 1e-9)
	require.InDelta(t, 0.75, resp.Nodes[1].Weight, 1e-9)
}
