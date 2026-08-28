package controlplane

import (
	"context"
	"encoding/json"
	"testing"

	"ad-event-processor/internal/database"
	"ad-event-processor/internal/domain/db"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCampaignPublishGate_resumeBlockedWithoutFlow_holdout(t *testing.T) {
	if testing.Short() {
		t.Skip("integration: publish gate blocks resume without flow")
	}
	pool, cleanupDB := database.SetupTestDB(t)
	defer cleanupDB()
	redisClient, cleanupRedis := database.SetupTestRedis(t)
	defer cleanupRedis()

	svc := NewService(context.Background(), pool, []redis.UniversalClient{redisClient}, nil, nil)
	defer svc.Close()

	ctx := context.Background()
	custID := uuid.New()
	require.NoError(t, svc.CreateCustomer(ctx, custID, "Publish Gate Customer", 500_000_000, "USD"))

	campID, err := svc.CreateCampaign(ctx, testCampaignSpec(custID, "Gate Camp", 50_000_000, "publish-gate-idem"))
	require.NoError(t, err)

	require.NoError(t, svc.PauseCampaign(ctx, campID, "manual"))
	err = svc.ResumeCampaign(ctx, campID, "manual")
	require.Error(t, err)
	var blocked *CampaignPublishBlockedError
	require.ErrorAs(t, err, &blocked)
	assert.Contains(t, blocked.FieldErrors, "flow_id")

	attachPublishableFlowFixture(t, ctx, pool, svc, campID)
	require.NoError(t, svc.ResumeCampaign(ctx, campID, "manual"))

	camp, err := svc.GetCampaignRow(ctx, campID)
	require.NoError(t, err)
	assert.Equal(t, db.CampaignStatusTypeACTIVE, camp.Status)
}

func TestCampaignPublishGate_patchActiveBlockedWithoutFlow_holdout(t *testing.T) {
	if testing.Short() {
		t.Skip("integration: publish gate blocks patch to active without flow")
	}
	pool, cleanupDB := database.SetupTestDB(t)
	defer cleanupDB()
	redisClient, cleanupRedis := database.SetupTestRedis(t)
	defer cleanupRedis()

	svc := NewService(context.Background(), pool, []redis.UniversalClient{redisClient}, nil, nil)
	defer svc.Close()

	ctx := context.Background()
	custID := uuid.New()
	require.NoError(t, svc.CreateCustomer(ctx, custID, "Patch Gate Customer", 500_000_000, "USD"))

	campID, err := svc.CreateCampaign(ctx, testCampaignSpec(custID, "Patch Gate Camp", 50_000_000, "publish-gate-patch-idem"))
	require.NoError(t, err)
	require.NoError(t, svc.PauseCampaign(ctx, campID, "manual"))

	active := "ACTIVE"
	_, err = svc.PatchCampaign(ctx, campID, PatchCampaignRequest{Status: &active})
	require.Error(t, err)
	var blocked *CampaignPublishBlockedError
	require.ErrorAs(t, err, &blocked)
	assert.Contains(t, blocked.FieldErrors, "flow_id")

	attachPublishableFlowFixture(t, ctx, pool, svc, campID)
	updated, err := svc.PatchCampaign(ctx, campID, PatchCampaignRequest{Status: &active})
	require.NoError(t, err)
	assert.Equal(t, "ACTIVE", updated.Status)
}

func attachPublishableFlowFixture(t *testing.T, ctx context.Context, pool *pgxpool.Pool, svc *Service, campID uuid.UUID) {
	t.Helper()
	landerID := uuid.New()
	offerID := uuid.New()
	_, err := pool.Exec(ctx, `INSERT INTO landers (id, name, url) VALUES ($1, 'L1', 'https://lander.example/')`, landerID)
	require.NoError(t, err)
	_, err = pool.Exec(ctx, `INSERT INTO offers (id, name, url) VALUES ($1, 'O1', 'https://offer.example/')`, offerID)
	require.NoError(t, err)

	flowID := uuid.New()
	paths, err := json.Marshal([]FlowPathDTO{{
		Weight:  100,
		Landers: []FlowPathLanderRef{{LanderID: landerID, Weight: 100}},
		Offers:  []FlowPathOfferRef{{OfferID: offerID, Weight: 100}},
	}})
	require.NoError(t, err)
	_, err = pool.Exec(ctx, `INSERT INTO flows (id, name, paths) VALUES ($1, 'publish-flow', $2)`, flowID, paths)
	require.NoError(t, err)
	require.NoError(t, svc.AssignCampaignFlow(ctx, campID, flowID))

	_, err = pool.Exec(ctx, `
		UPDATE campaigns
		SET target_url = 'https://offer.example/click?cid={{campaign.id}}&sub1={{sub1}}'
		WHERE id = $1`, campID)
	require.NoError(t, err)
}
