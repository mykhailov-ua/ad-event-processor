package campaign

import (
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

func TestPutManualCampaignCost_rejectsInvalidDate_holdout(t *testing.T) {
	t.Parallel()
	_, err := PutManualCampaignCost(nil, nil, uuid.New(), uuid.New(), ManualCampaignCostRequest{
		CostDate:    "not-a-date",
		AmountMicro: 1_000_000,
	})
	require.Error(t, err)
}
