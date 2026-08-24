package controlplane

import (
	"context"
	"encoding/json"
	"testing"

	"ad-event-processor/internal/config"
	"ad-event-processor/internal/database"
	"ad-event-processor/internal/domain"
	"ad-event-processor/internal/testutil"

	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/require"
)

func TestCohortConfig_FanoutAndRegistryReload(t *testing.T) {
	if testing.Short() {
		t.Skip("integration: run make test-integration (Docker testcontainers)")
	}

	ctx := context.Background()
	pool, cleanupDB := database.SetupTestDB(t)
	defer cleanupDB()
	rdb, cleanupRedis := database.SetupTestRedis(t)
	defer cleanupRedis()

	_, err := pool.Exec(ctx, `
		INSERT INTO regions (code, name, active) VALUES (1, 'us-east', TRUE)
		ON CONFLICT (code) DO NOTHING`)
	require.NoError(t, err)

	experimentID := uuid.New()
	cfg := &config.Config{MultiRegionEnabled: true, RegionCode: 1}
	svc := newBareService(t, pool, []redis.UniversalClient{rdb}, cfg)

	require.NoError(t, svc.UpsertExperimentCohort(ctx, ExperimentCohortSpec{
		ID:     experimentID,
		Name:   "homepage-cta",
		Active: true,
		Salt:   "salt-2026",
		Variants: []CohortVariantSpec{
			{ID: "control", Weight: 50, Flags: map[string]string{"btn": "blue"}},
			{ID: "treatment", Weight: 50, Flags: map[string]string{"btn": "green"}},
		},
	}))

	var eventID int64
	require.NoError(t, pool.QueryRow(ctx, `
		SELECT id FROM outbox_events WHERE event_type = 'UPDATE_COHORT_SNAPSHOT' ORDER BY id DESC LIMIT 1`).Scan(&eventID))

	var deliveryStatus string
	require.NoError(t, pool.QueryRow(ctx, `
		SELECT status FROM outbox_region_delivery WHERE outbox_event_id = $1 AND region_code = 1`, eventID).Scan(&deliveryStatus))
	require.Equal(t, "PENDING", deliveryStatus)

	relay := NewRegionOutboxRelay(svc)
	require.NoError(t, relay.ProcessPending(ctx))

	require.NoError(t, pool.QueryRow(ctx, `
		SELECT status FROM outbox_region_delivery WHERE outbox_event_id = $1 AND region_code = 1`, eventID).Scan(&deliveryStatus))
	require.Equal(t, "DELIVERED", deliveryStatus)

	reg := testutil.NewCohortRegistry(pool)
	require.NoError(t, reg.SyncCohorts(ctx))
	require.Equal(t, 1, reg.ExperimentCount())

	variant, flags, ok := reg.AssignExperiment(experimentID, "user-42")
	require.True(t, ok)
	require.NotEmpty(t, variant)
	require.Contains(t, []string{"control", "treatment"}, variant)

	variant2, _, ok2 := reg.AssignExperiment(experimentID, "user-42")
	require.True(t, ok2)
	require.Equal(t, variant, variant2)
	if flags != nil {
		require.NotEmpty(t, flags["btn"])
	}

	variantsJSON, _ := json.Marshal([]CohortVariantSpec{
		{ID: "control", Weight: 50},
		{ID: "treatment", Weight: 50},
	})
	_, err = pool.Exec(ctx, `
		INSERT INTO experiment_cohorts (id, name, active, salt, variants)
		VALUES ($1, 'direct', TRUE, 's', $2)
		ON CONFLICT (id) DO NOTHING`, domain.ToUUID(uuid.New()), variantsJSON)
	require.NoError(t, err)
}
