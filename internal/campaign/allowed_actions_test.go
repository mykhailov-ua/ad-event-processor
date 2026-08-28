package campaign

import (
	"context"
	"testing"

	"ad-event-processor/internal/controlplane/authz"

	"github.com/stretchr/testify/assert"
)

func TestComputeCampaignAllowedActions_buyerPauseOnly(t *testing.T) {
	t.Parallel()
	ctx := authz.WithSnapshot(context.Background(), authz.Snapshot{
		Permissions: map[string]struct{}{
			authz.PermCampaignsReadMasked: {},
			authz.PermCampaignsPause:      {},
		},
		Mask: authz.MaskMasked,
	})
	actions, denied := computeCampaignAllowedActions(ctx, "ACTIVE")
	assert.Contains(t, actions, "pause")
	assert.NotContains(t, actions, "edit_fraud")
	assert.Equal(t, "requires_campaigns_write", denied["edit_fraud"])
}
