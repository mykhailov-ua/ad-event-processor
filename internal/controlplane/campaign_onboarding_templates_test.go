package controlplane

import (
	"context"
	"testing"

	"ad-event-processor/internal/database"

	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCampaignWizard_templateCommitRoundTrip_holdout(t *testing.T) {
	if testing.Short() {
		t.Skip("integration: onboarding template wizard commit round-trip")
	}
	pool, cleanupDB := database.SetupTestDB(t)
	defer cleanupDB()
	redisClient, cleanupRedis := database.SetupTestRedis(t)
	defer cleanupRedis()

	svc := NewService(context.Background(), pool, []redis.UniversalClient{redisClient}, nil, nil)
	defer svc.Close()

	ctx := context.Background()
	custID := uuid.New()
	require.NoError(t, svc.CreateCustomer(ctx, custID, "Template Wizard Customer", 500_000_000, "USD"))

	templateKeys := []string{"meta_social_funnel", "popunder_propeller", "push_house_funnel"}
	for i, templateKey := range templateKeys {
		session, err := svc.CreateCampaignWizardSession(ctx, custID, templateKey)
		require.NoError(t, err, templateKey)
		require.True(t, session.ReadyToCommit, templateKey)
		assert.Equal(t, wizardStepReview, session.CurrentStep, templateKey)

		sessionID, err := uuid.Parse(session.SessionID)
		require.NoError(t, err)
		result, err := svc.CommitCampaignWizardSession(ctx, sessionID, "wizard-template-commit-"+templateKey, false)
		require.NoError(t, err, templateKey)
		require.NotEmpty(t, result.Campaign.ID)

		campaignID, err := uuid.Parse(result.Campaign.ID)
		require.NoError(t, err)
		row, err := svc.GetCampaignRow(ctx, campaignID)
		require.NoError(t, err)
		assert.True(t, row.FlowID.Valid, templateKey)
		_ = i
	}
}
