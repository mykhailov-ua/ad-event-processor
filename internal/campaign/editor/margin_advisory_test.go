package editor

import (
	"context"
	"testing"

	"ad-event-processor/internal/campaign"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type marginCampaignStub struct {
	diffCampaignStub
	margin campaign.CampaignMarginDTO
}

func (s *marginCampaignStub) GetCampaignMargin(context.Context, uuid.UUID) (campaign.CampaignMarginDTO, error) {
	return s.margin, nil
}

func TestMarginAdvisoryForCampaign_thresholdLabelUsesBasisPoints_holdout(t *testing.T) {
	t.Parallel()
	campID := uuid.New()
	advisories := marginAdvisoryForCampaign(context.Background(), campID, &marginCampaignStub{
		margin: campaign.CampaignMarginDTO{
			CampaignID:   campID.String(),
			ThresholdBps: 500,
			MarginBreach: true,
		},
	})
	require.Len(t, advisories, 1)
	assert.Equal(t, "5.0%", advisories[0].ImpactLabel)
	assert.InDelta(t, 5.0, advisories[0].FloorPct, 0.001)
}

func TestMarginAdvisoryForCampaign_noBreachEmpty(t *testing.T) {
	t.Parallel()
	advisories := marginAdvisoryForCampaign(context.Background(), uuid.New(), &marginCampaignStub{
		margin: campaign.CampaignMarginDTO{MarginBreach: false},
	})
	assert.Empty(t, advisories)
}

func TestMarginAdvisoryForCampaign_readOnlyNoBudgetMutation_holdout(t *testing.T) {
	t.Parallel()
	campID := uuid.New()
	before := campaign.CampaignMarginDTO{
		CampaignID:           campID.String(),
		ThresholdBps:         500,
		MarginBreach:         true,
		AdvertiserSpendMicro: 1_000_000,
	}
	stub := &marginCampaignStub{margin: before}
	advisories := marginAdvisoryForCampaign(context.Background(), campID, stub)
	require.Len(t, advisories, 1)
	after, err := stub.GetCampaignMargin(context.Background(), campID)
	require.NoError(t, err)
	assert.Equal(t, before.AdvertiserSpendMicro, after.AdvertiserSpendMicro)
}
