package fraud

import (
	"context"
	"testing"

	"ad-event-processor/internal/reports"

	"ad-event-processor/internal/controlplane/authz"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestFilterWireSignalBreakdownRows_tlsJa4(t *testing.T) {
	t.Parallel()
	raw := []reports.FraudBreakdownRowDTO{
		{CampaignID: "c1", FraudReason: "tls_ja4_mismatch", EventCount: 40, SilentRejectCount: 8},
		{CampaignID: "c1", FraudReason: "low_ttc", EventCount: 10, SilentRejectCount: 1},
	}
	ctx := authz.WithSnapshot(context.Background(), authz.Snapshot{Mask: authz.MaskMasked})
	rows, total, err := filterWireSignalBreakdownRows(raw, 50, 0, ctx)
	require.NoError(t, err)
	require.Equal(t, int64(1), total)
	require.Len(t, rows, 1)
	assert.Empty(t, rows[0].FraudReason)
	assert.Equal(t, fraudCategoryInvalidDevice, rows[0].FraudCategory)
	assert.InDelta(t, 0.2, rows[0].SilentRejectRatio, 0.0001)
}
