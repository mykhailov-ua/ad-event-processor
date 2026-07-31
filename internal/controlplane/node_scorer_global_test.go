package controlplane

import (
	"context"
	"testing"
	"time"

	"espx/internal/config"
	"espx/internal/database"
	db "espx/internal/domain/db"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestComputeRegionDialResults_weightsTrackScores(t *testing.T) {
	cfg := DefaultScorerConfig()
	results := ComputeRegionDialResults([]RegionDialInput{
		{
			RegionCode: 1,
			Nodes: []db.NodeCapacityScore{
				{NodeID: "us-1", Score: 0.90, Weight: 1, Provenance: ProvenanceOwnWindow},
			},
			PrevWeight: 0.5,
		},
		{
			RegionCode: 2,
			Nodes: []db.NodeCapacityScore{
				{NodeID: "eu-1", Score: 0.40, Weight: 1, Provenance: ProvenanceOwnWindow},
			},
			PrevWeight: 0.5,
		},
	}, cfg)
	require.Len(t, results, 2)
	assert.Greater(t, results[0].Weight, results[1].Weight)
	var sum float64
	for _, r := range results {
		sum += r.Weight
	}
	assert.InDelta(t, 1.0, sum, 1e-6)
}

func TestAggregateRegionDialScore_weightedMean(t *testing.T) {
	score, prov, ok := AggregateRegionDialScore([]db.NodeCapacityScore{
		{Score: 0.8, Weight: 0.75, Provenance: ProvenanceOwnWindow},
		{Score: 0.4, Weight: 0.25, Provenance: ProvenanceNeighborMedian},
	})
	require.True(t, ok)
	assert.InDelta(t, 0.7, score, 1e-6)
	assert.Equal(t, ProvenanceNeighborMedian, prov)
}

func TestGlobalRegionTrafficScorer_doesNotAlterForeignNodeScores(t *testing.T) {
	t.Parallel()
	if testing.Short() {
		t.Skip("integration test")
	}

	ctx := context.Background()
	pool, cleanupDB := database.SetupTestDB(t)
	defer cleanupDB()

	_, err := pool.Exec(ctx, `
		INSERT INTO regions (code, name, active) VALUES
			(1, 'us-east', TRUE),
			(2, 'eu-west', TRUE)`)
	require.NoError(t, err)

	q := db.New(pool)
	require.NoError(t, q.UpsertNodeCapacityScore(ctx, db.UpsertNodeCapacityScoreParams{
		NodeID: "us-tracker-1", RegionCode: 1, Role: RoleTracker,
		Score: 0.92, Weight: 1.0, Provenance: ProvenanceOwnWindow, EpochID: 1,
	}))
	require.NoError(t, q.UpsertNodeCapacityScore(ctx, db.UpsertNodeCapacityScoreParams{
		NodeID: "eu-tracker-1", RegionCode: 2, Role: RoleTracker,
		Score: 0.55, Weight: 1.0, Provenance: ProvenanceOwnWindow, EpochID: 1,
	}))

	usBefore, err := q.ListNodeCapacityScoresByRegionRole(ctx, db.ListNodeCapacityScoresByRegionRoleParams{
		RegionCode: 1, Role: RoleTracker,
	})
	require.NoError(t, err)
	require.Len(t, usBefore, 1)

	cfg := &config.Config{MultiRegionEnabled: true, RegionCode: 0}
	svc := newBareService(t, pool, nil, cfg)
	global := NewGlobalRegionTrafficScorer(svc)
	require.NoError(t, global.Tick(ctx, testGlobalScorerNow()))

	euDial, err := q.GetRegionTrafficDial(ctx, 2)
	require.NoError(t, err)
	assert.Greater(t, euDial.Weight, 0.0)

	require.NoError(t, q.UpsertNodeCapacityScore(ctx, db.UpsertNodeCapacityScoreParams{
		NodeID: "eu-tracker-1", RegionCode: 2, Role: RoleTracker,
		Score: 0.15, Weight: 1.0, Provenance: ProvenanceOwnWindow, EpochID: 2,
	}))
	require.NoError(t, global.Tick(ctx, testGlobalScorerNow()))

	usAfter, err := q.ListNodeCapacityScoresByRegionRole(ctx, db.ListNodeCapacityScoresByRegionRoleParams{
		RegionCode: 1, Role: RoleTracker,
	})
	require.NoError(t, err)
	require.Len(t, usAfter, 1)
	assert.Equal(t, usBefore[0].Score, usAfter[0].Score)
	assert.Equal(t, usBefore[0].Weight, usAfter[0].Weight)
	assert.Equal(t, usBefore[0].Provenance, usAfter[0].Provenance)
	assert.Equal(t, usBefore[0].EpochID, usAfter[0].EpochID)

	euDialAfter, err := q.GetRegionTrafficDial(ctx, 2)
	require.NoError(t, err)
	assert.Less(t, euDialAfter.Weight, euDial.Weight)

	dials, err := q.ListRegionTrafficDial(ctx)
	require.NoError(t, err)
	require.Len(t, dials, 2)
}

func testGlobalScorerNow() time.Time {
	return time.Date(2026, 7, 27, 16, 0, 0, 0, time.UTC)
}
