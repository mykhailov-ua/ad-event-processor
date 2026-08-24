package platformsync_test

import (
	"testing"

	db "ad-event-processor/internal/domain/db"
	"ad-event-processor/internal/platformsync"

	"github.com/jackc/pgx/v5/pgtype"
	"github.com/stretchr/testify/require"
)

func TestPreviewMutation_pauseNoopWhenAlreadyPaused(t *testing.T) {
	link := db.PlatformCampaignLink{
		CampaignID:     pgtype.UUID{Valid: true},
		Network:        platformsync.NetworkFacebook,
		ExternalStatus: "PAUSED",
	}
	preview, err := platformsync.PreviewMutation(link, platformsync.ActionPause, platformsync.MutationRequest{})
	require.NoError(t, err)
	require.True(t, preview.Noop)
	require.False(t, preview.VendorWrite)
}

func TestPreviewMutation_budgetRequiresAmount(t *testing.T) {
	link := db.PlatformCampaignLink{Network: platformsync.NetworkGoogle}
	_, err := platformsync.PreviewMutation(link, platformsync.ActionSetDailyBudget, platformsync.MutationRequest{})
	require.Error(t, err)
}

func TestPreviewMutation_resumeSetsActiveStatus(t *testing.T) {
	link := db.PlatformCampaignLink{
		Network:        platformsync.NetworkFacebook,
		ExternalStatus: "PAUSED",
	}
	preview, err := platformsync.PreviewMutation(link, platformsync.ActionResume, platformsync.MutationRequest{})
	require.NoError(t, err)
	require.Equal(t, "ACTIVE", preview.StatusTo)
	require.True(t, preview.VendorWrite)
}
