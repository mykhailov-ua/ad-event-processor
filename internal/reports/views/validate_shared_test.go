package views

import (
	"context"
	"encoding/json"
	"testing"

	"ad-event-processor/internal/controlplane/authz"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

func TestValidateSharedSavedViewForActor_opsSharedReportBuyerDenied_holdout(t *testing.T) {
	t.Parallel()
	ctx := authz.WithSnapshot(context.Background(), authz.Snapshot{
		Permissions: map[string]struct{}{"campaigns:read:masked": {}},
		Mask:        authz.MaskMasked,
	})
	view := SavedViewDTO{
		ReportKey:      "fraud-evidence-pack",
		Spec:           json.RawMessage(`{"from":"2026-03-01T00:00:00Z","to":"2026-03-05T00:00:00Z"}`),
		IsShared:       true,
		OwnerMaskLevel: string(authz.MaskFull),
	}
	require.Error(t, validateSharedSavedViewForActor(ctx, view))
}

func TestValidateSharedSavedViewForActor_buyerSharedPlacementsAllowed(t *testing.T) {
	t.Parallel()
	ctx := authz.WithSnapshot(context.Background(), authz.Snapshot{
		Permissions: map[string]struct{}{"campaigns:read:masked": {}},
		Mask:        authz.MaskMasked,
	})
	view := SavedViewDTO{
		ReportKey:      "placements",
		Spec:           json.RawMessage(`{"limit":25}`),
		IsShared:       true,
		OwnerMaskLevel: string(authz.MaskFull),
	}
	require.NoError(t, validateSharedSavedViewForActor(ctx, view))
}

func TestValidateReportScheduleForActor_boundCustomerMismatch_holdout(t *testing.T) {
	t.Parallel()
	customerID := "11111111-1111-1111-1111-111111111111"
	ctx := authz.WithAuthenticatedUser(context.Background(), authz.AuthenticatedUser{
		Role:       authz.RoleBuyer,
		CustomerID: mustParseUUID(t, "22222222-2222-2222-2222-222222222222"),
	})
	err := ValidateReportScheduleForActor(ctx, customerID, "placements", json.RawMessage(`{"limit":10}`))
	require.ErrorIs(t, err, ErrForbidden)
}

func mustParseUUID(t *testing.T, raw string) uuid.UUID {
	t.Helper()
	id, err := uuid.Parse(raw)
	require.NoError(t, err)
	return id
}
