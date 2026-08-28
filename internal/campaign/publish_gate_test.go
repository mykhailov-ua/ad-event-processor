package campaign

import (
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestEvaluatePublishBlocked_missingFlow_holdout(t *testing.T) {
	t.Parallel()
	blocked := EvaluatePublishBlocked(PublishGateEvalInput{
		CampaignID:  uuid.New(),
		Name:        "camp",
		BudgetLimit: 100,
		FlowMissing: true,
	})
	require.NotNil(t, blocked)
	assert.Contains(t, blocked.FieldErrors, "flow_id")
}

func TestEvaluatePublishBlocked_validMinimal(t *testing.T) {
	t.Parallel()
	blocked := EvaluatePublishBlocked(PublishGateEvalInput{
		CampaignID:          uuid.New(),
		Name:                "camp",
		BudgetLimit:         100,
		TargetURL:           "https://offer.example/?cid={{campaign.id}}",
		ClickDelivery:       "redirect",
		AllowHTTPInsecure:   true,
		IntegrationHealthOK: true,
	})
	assert.Nil(t, blocked)
}
