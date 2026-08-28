package controlplane

import (
	"context"
	"encoding/json"
	"strings"
	"testing"

	"ad-event-processor/internal/campaign"
	"ad-event-processor/internal/database"

	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCampaignWizard_commitIncompleteSession_holdout(t *testing.T) {
	if testing.Short() {
		t.Skip("integration: wizard incomplete commit holdout")
	}
	pool, cleanupDB := database.SetupTestDB(t)
	defer cleanupDB()
	redisClient, cleanupRedis := database.SetupTestRedis(t)
	defer cleanupRedis()

	svc := NewService(context.Background(), pool, []redis.UniversalClient{redisClient}, nil, nil)
	defer svc.Close()

	ctx := context.Background()
	custID := uuid.New()
	require.NoError(t, svc.CreateCustomer(ctx, custID, "Wizard Customer", 500_000_000, "USD"))

	session, err := svc.CreateCampaignWizardSession(ctx, custID, "")
	require.NoError(t, err)
	sessionID, err := uuid.Parse(session.SessionID)
	require.NoError(t, err)

	_, err = svc.CommitCampaignWizardSession(ctx, sessionID, "wizard-commit-idem-1", false)
	require.Error(t, err)
	assert.ErrorIs(t, err, errCampaignWizardIncomplete)
}

func TestCampaignWizard_commitCreatesCampaignFlow_holdout(t *testing.T) {
	if testing.Short() {
		t.Skip("integration: wizard commit creates campaign and flow")
	}
	pool, cleanupDB := database.SetupTestDB(t)
	defer cleanupDB()
	redisClient, cleanupRedis := database.SetupTestRedis(t)
	defer cleanupRedis()

	svc := NewService(context.Background(), pool, []redis.UniversalClient{redisClient}, nil, nil)
	defer svc.Close()

	ctx := context.Background()
	custID := uuid.New()
	require.NoError(t, svc.CreateCustomer(ctx, custID, "Wizard Commit Customer", 500_000_000, "USD"))

	session, err := svc.CreateCampaignWizardSession(ctx, custID, "")
	require.NoError(t, err)
	sessionID, err := uuid.Parse(session.SessionID)
	require.NoError(t, err)

	_, err = svc.UpdateCampaignWizardSessionStep(ctx, sessionID, wizardStepTrafficSource, mustWizardJSON(t, campaign.CampaignWizardTrafficSourceStep{
		Name:              "Wizard Camp",
		TrafficTemplateID: "meta-facebook",
	}))
	require.NoError(t, err)
	_, err = svc.UpdateCampaignWizardSessionStep(ctx, sessionID, wizardStepIntegrationTemplate, mustWizardJSON(t, campaign.CampaignWizardIntegrationTemplateStep{
		IntegrationSchema: "traffic_facebook",
	}))
	require.NoError(t, err)
	_, err = svc.UpdateCampaignWizardSessionStep(ctx, sessionID, wizardStepFlowSkeleton, mustWizardJSON(t, campaign.CampaignWizardFlowSkeletonStep{
		FlowName: "wizard-flow",
		Lander: campaign.CampaignWizardAssetRef{
			Name: "Wizard Lander",
			URL:  "https://lander.example/wizard",
		},
		Offer: campaign.CampaignWizardAssetRef{
			Name: "Wizard Offer",
			URL:  "https://offer.example/wizard",
		},
	}))
	require.NoError(t, err)
	_, err = svc.UpdateCampaignWizardSessionStep(ctx, sessionID, wizardStepBudget, mustWizardJSON(t, campaign.CampaignWizardBudgetStep{
		BudgetLimitMicro: 25_000_000,
		Timezone:         "UTC",
		TargetCountries:  []string{"US"},
	}))
	require.NoError(t, err)

	result, err := svc.CommitCampaignWizardSession(ctx, sessionID, "wizard-commit-idem-2", false)
	require.NoError(t, err)
	require.NotEmpty(t, result.Campaign.ID)

	campaignID, err := uuid.Parse(result.Campaign.ID)
	require.NoError(t, err)
	row, err := svc.GetCampaignRow(ctx, campaignID)
	require.NoError(t, err)
	assert.Equal(t, "Wizard Camp", row.Name)
	assert.True(t, row.FlowID.Valid)
	assert.Equal(t, "meta-facebook", formatOptionalText(row.TrafficTemplateID))
	assert.NotEmpty(t, strings.TrimSpace(row.TargetUrl))

	var flowCount int
	require.NoError(t, pool.QueryRow(ctx, `SELECT COUNT(*) FROM flows WHERE id = $1`, uuid.UUID(row.FlowID.Bytes)).Scan(&flowCount))
	assert.Equal(t, 1, flowCount)
}

func TestCampaignWizardSessionGET_omitsSecrets(t *testing.T) {
	dto := campaign.CampaignWizardSessionDTO{
		SessionID:  uuid.NewString(),
		CustomerID: uuid.NewString(),
		Steps: campaign.CampaignWizardStepsDTO{
			IntegrationTemplate: &campaign.CampaignWizardIntegrationTemplateStep{
				IntegrationSchema: "traffic_facebook",
				TrackingDomain:    "https://trk.example",
			},
		},
	}
	raw, err := json.Marshal(dto)
	require.NoError(t, err)
	body := string(raw)
	assert.NotContains(t, body, "api_token")
	assert.NotContains(t, body, "secret")
}

func mustWizardJSON(t *testing.T, v any) json.RawMessage {
	t.Helper()
	raw, err := json.Marshal(v)
	require.NoError(t, err)
	return raw
}
