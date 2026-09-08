package controlplane

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"ad-event-processor/internal/campaign"
	"ad-event-processor/internal/campaign/editor"
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

	_, err = pool.Exec(ctx, `
		INSERT INTO campaign_conversion_mappings (campaign_id, inbound_status, goal_name, payout_micro)
		VALUES ($1, 'approved', 'sale', 1000000)`, srcID)
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

	var mappingCount int
	err = pool.QueryRow(ctx, `SELECT COUNT(*) FROM campaign_conversion_mappings WHERE campaign_id = $1`, cloneID).Scan(&mappingCount)
	require.NoError(t, err)
	require.Equal(t, 1, mappingCount)

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

func TestBulkCloneCampaignsHTTP_holdout(t *testing.T) {
	if testing.Short() {
		t.Skip("integration: bulk clone HTTP copies mappings and enforces customer scope")
	}
	pool, cleanupDB := database.SetupTestDB(t)
	defer cleanupDB()
	redisClient, cleanupRedis := database.SetupTestRedis(t)
	defer cleanupRedis()

	svc := NewService(context.Background(), pool, []redis.UniversalClient{redisClient}, nil, nil)
	defer svc.Close()

	ctx := context.Background()
	custID := uuid.New()
	require.NoError(t, svc.CreateCustomer(ctx, custID, "Bulk Clone Customer", 500_000_000, "USD"))

	srcIDs := make([]uuid.UUID, 2)
	for i := range srcIDs {
		srcID, err := svc.CreateCampaign(ctx, testCampaignSpec(custID, "Bulk Source", 20_000_000, "bulk-src-"+uuid.NewString()))
		require.NoError(t, err)
		srcIDs[i] = srcID
		_, err = pool.Exec(ctx, `UPDATE campaigns SET current_spend = 5000 WHERE id = $1`, srcID)
		require.NoError(t, err)
		_, err = pool.Exec(ctx, `
			INSERT INTO campaign_conversion_mappings (campaign_id, inbound_status, goal_name, payout_micro)
			VALUES ($1, 'hold', 'lead', 0)`, srcID)
		require.NoError(t, err)
	}

	otherCust := uuid.New()
	require.NoError(t, svc.CreateCustomer(ctx, otherCust, "Other", 100_000_000, "USD"))
	foreignID, err := svc.CreateCampaign(ctx, testCampaignSpec(otherCust, "Foreign", 10_000_000, "bulk-foreign"))
	require.NoError(t, err)

	h := &campaign.CampaignsHTTPHandlers{Campaigns: svc}
	mux := http.NewServeMux()
	limit := func(next http.HandlerFunc) http.HandlerFunc { return next }
	perm := func(_ []string, next http.HandlerFunc) http.HandlerFunc { return next }
	editor.RegisterRoutes(h, mux, limit, perm)

	payload, err := json.Marshal(map[string]any{
		"source_campaign_ids": []string{srcIDs[0].String(), srcIDs[1].String(), foreignID.String()},
		"customer_id":         custID.String(),
		"name_suffix":         " (bulk)",
	})
	require.NoError(t, err)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/campaigns/bulk-clone", bytes.NewReader(payload))
	req.Header.Set("Idempotency-Key", "bulk-http-holdout")
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)
	require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())

	var resp campaign.BulkCloneCampaignsResult
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &resp))
	require.Len(t, resp.Results, 3)

	okCount := 0
	for _, row := range resp.Results {
		if row.SourceID == foreignID.String() {
			assert.Equal(t, "customer_mismatch", row.ErrorCode)
			continue
		}
		require.True(t, row.OK, row.ErrorCode)
		okCount++
		cloneID, err := uuid.Parse(row.ID)
		require.NoError(t, err)
		cloneRow, err := svc.GetCampaignRow(ctx, cloneID)
		require.NoError(t, err)
		assert.Equal(t, int64(0), cloneRow.CurrentSpend)
		assert.Contains(t, row.Name, "(bulk)")

		var mappingCount int
		err = pool.QueryRow(ctx, `SELECT COUNT(*) FROM campaign_conversion_mappings WHERE campaign_id = $1`, cloneID).Scan(&mappingCount)
		require.NoError(t, err)
		assert.Equal(t, 1, mappingCount)
	}
	assert.Equal(t, 2, okCount)
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
