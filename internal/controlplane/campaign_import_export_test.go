package controlplane

import (
	"context"
	"encoding/json"
	"testing"

	"ad-event-processor/internal/campaign"
	"ad-event-processor/internal/database"

	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCampaignImportExport_holdout(t *testing.T) {
	if testing.Short() {
		t.Skip("integration: campaign import export round-trip preserves flow and postback config")
	}
	pool, cleanupDB := database.SetupTestDB(t)
	defer cleanupDB()
	redisClient, cleanupRedis := database.SetupTestRedis(t)
	defer cleanupRedis()

	svc := NewService(context.Background(), pool, []redis.UniversalClient{redisClient}, nil, nil)
	defer svc.Close()

	ctx := context.Background()
	custID := uuid.New()
	require.NoError(t, svc.CreateCustomer(ctx, custID, "Import Export Customer", 500_000_000, "USD"))

	srcID, err := svc.CreateCampaign(ctx, testCampaignSpec(custID, "Export Source", 50_000_000, "export-src-idem"))
	require.NoError(t, err)

	landerID := uuid.New()
	offerID := uuid.New()
	_, err = pool.Exec(ctx, `INSERT INTO landers (id, name, url) VALUES ($1, 'L1', 'https://lander.example/')`, landerID)
	require.NoError(t, err)
	_, err = pool.Exec(ctx, `INSERT INTO offers (id, name, url) VALUES ($1, 'O1', 'https://offer.example/')`, offerID)
	require.NoError(t, err)

	flowID := uuid.New()
	paths, err := json.Marshal([]campaign.FlowPathDTO{{
		Weight:  100,
		Landers: []campaign.FlowPathLanderRef{{LanderID: landerID, Weight: 100}},
		Offers:  []campaign.FlowPathOfferRef{{OfferID: offerID, Weight: 100}},
	}})
	require.NoError(t, err)
	_, err = pool.Exec(ctx, `INSERT INTO flows (id, name, paths) VALUES ($1, 'export-flow', $2)`, flowID, paths)
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
		VALUES ($1, 'custom', 'https://aff.example/pb?cid={click_id}', '\xdeadbeef', 'conversion')`, srcID)
	require.NoError(t, err)

	_, err = svc.ReplaceCampaignConversionMappings(ctx, srcID, []campaign.ConversionMappingDTO{
		{InboundStatus: "approved", GoalName: "lead", PayoutMicro: 1_500_000},
	})
	require.NoError(t, err)

	bundle, err := svc.ExportCampaign(ctx, srcID)
	require.NoError(t, err)
	assert.Equal(t, campaignExportVersion, bundle.ExportVersion)
	require.NotNil(t, bundle.Flow)
	require.NotEmpty(t, bundle.Landers)
	require.NotEmpty(t, bundle.Offers)
	require.NotNil(t, bundle.PostbackConfig)
	rawBundle, err := json.Marshal(bundle)
	require.NoError(t, err)
	assert.NotContains(t, string(rawBundle), "deadbeef")
	assert.Len(t, bundle.ConversionMappings, 1)
	assert.Equal(t, "meta-facebook", bundle.Campaign.TrafficTemplateID)
	assert.Equal(t, "{{campaign.id}}", bundle.Campaign.ClickQueryParams["sub2"])

	bundle.Campaign.Name = "Imported Camp"
	result, err := svc.ImportCampaign(ctx, campaign.ImportCampaignSpec{
		CustomerID:     custID,
		IdempotencyKey: "import-camp-idem-1",
		Bundle:         bundle,
	})
	require.NoError(t, err)
	require.NotEqual(t, srcID.String(), result.ID)
	assert.Equal(t, "Imported Camp", result.Name)

	importID, err := uuid.Parse(result.ID)
	require.NoError(t, err)

	importRow, err := svc.GetCampaignRow(ctx, importID)
	require.NoError(t, err)
	require.True(t, importRow.FlowID.Valid)
	assert.NotEqual(t, flowID, uuid.UUID(importRow.FlowID.Bytes))

	var importPaths json.RawMessage
	err = pool.QueryRow(ctx, `SELECT paths FROM flows WHERE id = $1`, uuid.UUID(importRow.FlowID.Bytes)).Scan(&importPaths)
	require.NoError(t, err)
	var parsedPaths []campaign.FlowPathDTO
	require.NoError(t, json.Unmarshal(importPaths, &parsedPaths))
	require.Len(t, parsedPaths, 1)
	require.Len(t, parsedPaths[0].Landers, 1)
	require.Len(t, parsedPaths[0].Offers, 1)

	var postbackURL string
	err = pool.QueryRow(ctx, `SELECT url_template FROM postback_configs WHERE campaign_id = $1`, importID).Scan(&postbackURL)
	require.NoError(t, err)
	assert.Contains(t, postbackURL, "{click_id}")

	mappings, err := svc.ListCampaignConversionMappings(ctx, importID)
	require.NoError(t, err)
	require.Len(t, mappings, 1)
	assert.Equal(t, "approved", mappings[0].InboundStatus)
	assert.Equal(t, int64(1_500_000), mappings[0].PayoutMicro)

	imported, err := svc.GetCampaign(ctx, importID)
	require.NoError(t, err)
	assert.Equal(t, "meta-facebook", imported.TrafficTemplateID)
	assert.Equal(t, "{{campaign.id}}", imported.ClickQueryParams["sub2"])

	dup, err := svc.ImportCampaign(ctx, campaign.ImportCampaignSpec{
		CustomerID:     custID,
		IdempotencyKey: "import-camp-idem-1",
		Bundle:         bundle,
	})
	require.NoError(t, err)
	assert.Equal(t, result.ID, dup.ID)
}
