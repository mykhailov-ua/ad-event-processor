package smartalerts

import (
	"context"
	"testing"
	"time"

	"ad-event-processor/internal/database"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/stretchr/testify/require"
)

type alertActionHostStub struct {
	pausedCampaigns []uuid.UUID
	blacklisted     []string
}

func (h *alertActionHostStub) Pool() *pgxpool.Pool { return nil }

func (h *alertActionHostStub) ClickHouseQuery() *database.ClickHouseQuery { return nil }

func (h *alertActionHostStub) DrainStuckThresholdSec() int { return 0 }

func (h *alertActionHostStub) AlertDrainStuck(context.Context, int32, int16, string, string, time.Time) {
}

func (h *alertActionHostStub) PauseCampaign(_ context.Context, campaignID uuid.UUID, _ string) error {
	h.pausedCampaigns = append(h.pausedCampaigns, campaignID)
	return nil
}

func (h *alertActionHostStub) BlacklistPlacement(_ context.Context, campaignID uuid.UUID, placementID string) error {
	h.blacklisted = append(h.blacklisted, campaignID.String()+":"+placementID)
	return nil
}

func TestPauseCampaignsFromMarginActivity_dedupes_holdout(t *testing.T) {
	t.Parallel()
	host := &alertActionHostStub{}
	w := &Worker{host: host}
	campaignID := uuid.New()
	targets := []marginActivityTarget{
		{campaignID: campaignID, placementID: "p1"},
		{campaignID: campaignID, placementID: "p2"},
	}
	require.NoError(t, w.pauseCampaignsFromMarginActivity(context.Background(), targets, "test"))
	require.Len(t, host.pausedCampaigns, 1)
	require.Equal(t, campaignID, host.pausedCampaigns[0])
}

func TestBlacklistPlacementsFromMarginActivity_skipsEmpty_holdout(t *testing.T) {
	t.Parallel()
	host := &alertActionHostStub{}
	w := &Worker{host: host}
	campaignID := uuid.New()
	targets := []marginActivityTarget{
		{campaignID: campaignID, placementID: ""},
		{campaignID: campaignID, placementID: "site-1"},
	}
	require.NoError(t, w.blacklistPlacementsFromMarginActivity(context.Background(), targets, "test"))
	require.Equal(t, []string{campaignID.String() + ":site-1"}, host.blacklisted)
}
