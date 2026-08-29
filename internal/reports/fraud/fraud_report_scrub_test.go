package fraud

import (
	"context"
	"testing"

	"ad-event-processor/internal/reports"

	"ad-event-processor/internal/controlplane/authz"

	"github.com/stretchr/testify/assert"
)

func TestScrubFraudBreakdownRow_holdoutMasksReason(t *testing.T) {
	t.Parallel()
	ctx := authz.WithSnapshot(context.Background(), authz.Snapshot{
		Mask: authz.MaskMasked,
	})
	row := scrubFraudBreakdownRow(ctx, reports.FraudBreakdownRowDTO{
		FraudReason: "tls_ja4_mismatch",
		PlacementID: "pl-1",
	})
	assert.Empty(t, row.FraudReason)
	assert.Empty(t, row.PlacementID)
	assert.Equal(t, fraudCategoryInvalidDevice, row.FraudCategory)
}

func TestScrubFraudBreakdownRow_operatorKeepsReason(t *testing.T) {
	t.Parallel()
	ctx := authz.WithSnapshot(context.Background(), authz.Snapshot{
		Mask: authz.MaskFull,
	})
	row := scrubFraudBreakdownRow(ctx, reports.FraudBreakdownRowDTO{
		FraudReason: "tls_ja4_mismatch",
		PlacementID: "pl-1",
	})
	assert.Equal(t, "tls_ja4_mismatch", row.FraudReason)
	assert.Equal(t, "pl-1", row.PlacementID)
}
