package controlplane

import (
	"context"
	"encoding/json"
	"testing"

	"ad-event-processor/internal/campaign"
	"ad-event-processor/internal/database"
	"ad-event-processor/internal/domain"

	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCloneCampaignName_holdout(t *testing.T) {
	t.Parallel()
	assert.Equal(t, "Spring (copy)", campaign.CloneCampaignName("Spring", "", ""))
	assert.Equal(t, "Copy of Spring", campaign.CloneCampaignName("Spring", "Copy of ", ""))
	assert.Equal(t, "Spring - test", campaign.CloneCampaignName("Spring", "", " - test"))
}

func TestCloneCampaign_holdout(t *testing.T) {
	if testing.Short() {
		t.Skip("integration: campaign clone duplicates flow and postback config")
	}
	pool, cleanupDB := database.SetupTestDB(t)
	defer cleanupDB()
	redisClient, cleanupRedis := database.SetupTestRedis(t)
	defer cleanupRedis()

	svc := NewService(context.Background(), pool, []redis.UniversalClient{redisClient}, nil, nil)
	defer svc.Close()

	ctx := context.Background()
	custID := uuid.New()
	require.NoError(t, svc.CreateCustomer(ctx, custID, "Clone Customer", 500_000_000, "USD"))

	srcID, err := svc.CreateCampaign(ctx, testCampaignSpec(custID, "Source Camp", 50_000_000, "clone-src-idem"))
	require.NoError(t, err)

	flowID := uuid.New()
	paths := json.RawMessage(`[{"weight":100,"landers":[],"offers":[]}]`)
	_, err = pool.Exec(ctx, `INSERT INTO flows (id, name, paths) VALUES ($1, 'src-flow', $2)`, flowID, paths)
	require.NoError(t, err)
	require.NoError(t, svc.AssignCampaignFlow(ctx, srcID, flowID))

	_, err = pool.Exec(ctx, `
		UPDATE campaigns
		SET traffic_template_id = 'meta-facebook',
		    click_query_params = '{"sub2":"{{campaign.id}}"}'::jsonb
		WHERE id = $1`, srcID)
	require.NoError(t, err)

	_, err = pool.Exec(ctx, `
		INSERT INTO postback_configs (campaign_id, provider, url_template, api_token_encrypted, target_event)
		VALUES ($1, 'custom', 'https://aff.example/pb?cid={click_id}', '\x00', 'conversion')`, srcID)
	require.NoError(t, err)

	srcBefore, err := svc.GetCampaignRow(ctx, srcID)
	require.NoError(t, err)
	require.Equal(t, int64(0), srcBefore.CurrentSpend)

	result, err := svc.CloneCampaign(ctx, campaign.CloneCampaignSpec{
		SourceID:       srcID,
		IdempotencyKey: "clone-camp-idem-1",
	})
	require.NoError(t, err)
	require.NotEqual(t, srcID.String(), result.ID)
	assert.Equal(t, "Source Camp (copy)", result.Name)

	cloneID, err := uuid.Parse(result.ID)
	require.NoError(t, err)

	cloneRow, err := svc.GetCampaignRow(ctx, cloneID)
	require.NoError(t, err)
	assert.Equal(t, int64(0), cloneRow.CurrentSpend)
	assert.Equal(t, srcBefore.BudgetLimit, cloneRow.BudgetLimit)
	assert.NotEqual(t, srcBefore.FlowID, cloneRow.FlowID)
	require.True(t, cloneRow.FlowID.Valid)

	var clonePaths, srcPaths json.RawMessage
	err = pool.QueryRow(ctx, `SELECT paths FROM flows WHERE id = $1`, uuid.UUID(cloneRow.FlowID.Bytes)).Scan(&clonePaths)
	require.NoError(t, err)
	err = pool.QueryRow(ctx, `SELECT paths FROM flows WHERE id = $1`, flowID).Scan(&srcPaths)
	require.NoError(t, err)
	assert.JSONEq(t, string(srcPaths), string(clonePaths))

	var postbackURL string
	err = pool.QueryRow(ctx, `SELECT url_template FROM postback_configs WHERE campaign_id = $1`, cloneID).Scan(&postbackURL)
	require.NoError(t, err)
	assert.Contains(t, postbackURL, "{click_id}")

	cloneDTO, err := svc.GetCampaign(ctx, cloneID)
	require.NoError(t, err)
	assert.Equal(t, "meta-facebook", cloneDTO.TrafficTemplateID)
	assert.Equal(t, "{{campaign.id}}", cloneDTO.ClickQueryParams["sub2"])

	srcAfter, err := svc.GetCampaignRow(ctx, srcID)
	require.NoError(t, err)
	assert.Equal(t, srcBefore.Name, srcAfter.Name)
	assert.Equal(t, srcBefore.CurrentSpend, srcAfter.CurrentSpend)
	assert.Equal(t, flowID, uuid.UUID(srcAfter.FlowID.Bytes))

	dup, err := svc.CloneCampaign(ctx, campaign.CloneCampaignSpec{
		SourceID:       srcID,
		IdempotencyKey: "clone-camp-idem-1",
	})
	require.NoError(t, err)
	assert.Equal(t, result.ID, dup.ID)
}

func TestCloneCampaign_excludeFraud_holdout(t *testing.T) {
	if testing.Short() {
		t.Skip("integration: campaign clone resets fraud settings when include_fraud is false")
	}
	pool, cleanupDB := database.SetupTestDB(t)
	defer cleanupDB()
	redisClient, cleanupRedis := database.SetupTestRedis(t)
	defer cleanupRedis()

	svc := NewService(context.Background(), pool, []redis.UniversalClient{redisClient}, nil, nil)
	defer svc.Close()

	ctx := context.Background()
	custID := uuid.New()
	require.NoError(t, svc.CreateCustomer(ctx, custID, "Clone Fraud Customer", 500_000_000, "USD"))

	srcID, err := svc.CreateCampaign(ctx, testCampaignSpec(custID, "Fraud Source", 50_000_000, "clone-fraud-src"))
	require.NoError(t, err)
	_, err = pool.Exec(ctx, `
		UPDATE campaigns
		SET fraud_threshold_pass = 5,
		    fraud_threshold_suspect = 10,
		    fraud_threshold_ivt = 15,
		    fraud_threshold_block = 20,
		    silent_reject_enabled = true
		WHERE id = $1`, srcID)
	require.NoError(t, err)

	result, err := svc.CloneCampaign(ctx, campaign.CloneCampaignSpec{
		SourceID:       srcID,
		IdempotencyKey: "clone-fraud-exclude",
		Options: campaign.CloneCampaignOptions{
			IncludeFlow:      true,
			IncludePostbacks: true,
			IncludeFraud:     false,
		},
	})
	require.NoError(t, err)

	cloneID, err := uuid.Parse(result.ID)
	require.NoError(t, err)
	cloneRow, err := svc.GetCampaignRow(ctx, cloneID)
	require.NoError(t, err)
	assert.Equal(t, int64(0), cloneRow.CurrentSpend)
	assert.Equal(t, int16(domain.DefaultFraudThresholdPass), cloneRow.FraudThresholdPass)
	assert.Equal(t, int16(domain.DefaultFraudThresholdSuspect), cloneRow.FraudThresholdSuspect)
	assert.Equal(t, int16(domain.DefaultFraudThresholdIVT), cloneRow.FraudThresholdIvt)
	assert.Equal(t, int16(domain.DefaultFraudThresholdBlock), cloneRow.FraudThresholdBlock)
	assert.False(t, cloneRow.SilentRejectEnabled)
}

func TestCloneCampaign_placementBlocks_holdout(t *testing.T) {
	if testing.Short() {
		t.Skip("integration: campaign clone copies placement blocks when requested")
	}
	pool, cleanupDB := database.SetupTestDB(t)
	defer cleanupDB()
	redisClient, cleanupRedis := database.SetupTestRedis(t)
	defer cleanupRedis()

	svc := NewService(context.Background(), pool, []redis.UniversalClient{redisClient}, nil, nil)
	defer svc.Close()

	ctx := context.Background()
	custID := uuid.New()
	require.NoError(t, svc.CreateCustomer(ctx, custID, "Clone Placement Customer", 500_000_000, "USD"))

	srcID, err := svc.CreateCampaign(ctx, testCampaignSpec(custID, "Placement Source", 50_000_000, "clone-placement-src"))
	require.NoError(t, err)
	srcKey := domain.PlacementBlacklistKey(srcID)
	require.NoError(t, redisClient.HSet(ctx, srcKey, "zone-high-ivt", "1").Err())

	result, err := svc.CloneCampaign(ctx, campaign.CloneCampaignSpec{
		SourceID:       srcID,
		IdempotencyKey: "clone-placement-blocks",
		Options: campaign.CloneCampaignOptions{
			IncludeFlow:            true,
			IncludePostbacks:       true,
			IncludePlacementBlocks: true,
		},
	})
	require.NoError(t, err)

	cloneID, err := uuid.Parse(result.ID)
	require.NoError(t, err)
	cloneKey := domain.PlacementBlacklistKey(cloneID)
	val, err := redisClient.HGet(ctx, cloneKey, "zone-high-ivt").Result()
	require.NoError(t, err)
	assert.Equal(t, "1", val)
}
