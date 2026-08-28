package platformsync_test

import (
	"testing"

	db "ad-event-processor/internal/domain/db"
	"ad-event-processor/internal/platformsync"

	"github.com/jackc/pgx/v5/pgtype"
	"github.com/stretchr/testify/require"
)

func TestPreviewMutation_tiktokPauseNoopWhenDisabled(t *testing.T) {
	link := db.PlatformCampaignLink{
		Network:        platformsync.NetworkTikTok,
		ExternalStatus: "DISABLE",
	}
	preview, err := platformsync.PreviewMutation(link, platformsync.ActionPause, platformsync.MutationRequest{})
	require.NoError(t, err)
	require.True(t, preview.Noop)
	require.False(t, preview.VendorWrite)
}

func TestPreviewMutation_microsoftResumeSetsActiveStatus(t *testing.T) {
	link := db.PlatformCampaignLink{
		Network:        platformsync.NetworkMicrosoftAds,
		ExternalStatus: "Paused",
	}
	preview, err := platformsync.PreviewMutation(link, platformsync.ActionResume, platformsync.MutationRequest{})
	require.NoError(t, err)
	require.Equal(t, "Active", preview.StatusTo)
	require.True(t, preview.VendorWrite)
}

func TestPreviewMutation_tiktokBudgetUnsupported(t *testing.T) {
	link := db.PlatformCampaignLink{Network: platformsync.NetworkTikTok}
	_, err := platformsync.PreviewMutation(link, platformsync.ActionSetDailyBudget, platformsync.MutationRequest{DailyBudgetMicro: 1_000_000})
	require.Error(t, err)
}

func TestNetworkSupported_tiktokAndMicrosoft(t *testing.T) {
	require.True(t, platformsync.NetworkSupported(platformsync.NetworkTikTok))
	require.True(t, platformsync.NetworkSupported(platformsync.NetworkMicrosoftAds))
}

func TestMutationFault_remoteFailureDoesNotImplyLocalPause_holdout(t *testing.T) {
	link := db.PlatformCampaignLink{
		CampaignID:     pgtype.UUID{Valid: true},
		Network:        platformsync.NetworkTikTok,
		ExternalStatus: "ACTIVE",
	}
	preview, err := platformsync.PreviewMutation(link, platformsync.ActionPause, platformsync.MutationRequest{})
	require.NoError(t, err)
	require.Equal(t, "ACTIVE", preview.StatusFrom)
	require.Equal(t, "DISABLE", preview.StatusTo)
	require.False(t, preview.Noop)
}
