package controlplane

import (
	"context"
	"encoding/json"
	"testing"

	"ad-event-processor/internal/controlplane/authz"

	"github.com/stretchr/testify/require"
)

func TestValidateSavedViewActorPolicy_buyerBlocksOpsReport_holdout(t *testing.T) {
	t.Parallel()
	ctx := authz.WithSnapshot(context.Background(), authz.Snapshot{
		Mask: authz.MaskMasked,
	})
	spec := json.RawMessage(`{"from":"2026-03-01T00:00:00Z","to":"2026-03-05T00:00:00Z"}`)
	err := validateSavedViewActorPolicy(ctx, "fraud-evidence-pack", spec)
	require.Error(t, err)
}

func TestValidateSavedViewRangeCap_buyerExceedsSevenDays_holdout(t *testing.T) {
	t.Parallel()
	spec := json.RawMessage(`{"from":"2026-03-01T00:00:00Z","to":"2026-03-15T00:00:00Z"}`)
	err := validateSavedViewRangeCap(spec, savedViewMaxRangeDaysBuyer)
	require.Error(t, err)
}

func TestValidateSavedViewInput_unknownReportKey(t *testing.T) {
	t.Parallel()
	err := validateSavedViewInput("test", "not-a-report", nil)
	require.Error(t, err)
}
