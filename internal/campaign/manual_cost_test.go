package campaign

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

func TestPutManualCampaignCost_rejectsInvalidDate_holdout(t *testing.T) {
	t.Parallel()
	_, err := PutManualCampaignCost(context.Background(), nil, uuid.New(), uuid.New(), ManualCampaignCostRequest{
		CostDate:    "not-a-date",
		AmountMicro: 1_000_000,
	})
	require.Error(t, err)
}
