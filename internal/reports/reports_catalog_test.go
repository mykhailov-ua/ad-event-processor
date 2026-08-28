package reports

import (
	"context"
	"testing"

	"ad-event-processor/internal/controlplane/authz"

	"github.com/stretchr/testify/assert"
)

func TestFilterReportCatalog_buyerOmitsEvidencePack(t *testing.T) {
	t.Parallel()
	ctx := authz.WithSnapshot(context.Background(), authz.Snapshot{
		Permissions: map[string]struct{}{
			permCampaignsReadMasked: {},
		},
		Mask: authz.MaskMasked,
	})
	rows := FilterReportCatalog(ctx, ReportCatalogEntries)
	for _, row := range rows {
		assert.NotEqual(t, "fraud-evidence-pack", row.Key)
		assert.NotEqual(t, "filter-rejects", row.Key)
	}
	found := false
	for _, row := range rows {
		if row.Key == "customer-fraud-by-type" {
			found = true
		}
	}
	assert.True(t, found)
}

func TestFilterReportCatalog_operatorSeesEvidencePack(t *testing.T) {
	t.Parallel()
	ctx := authz.WithSnapshot(context.Background(), authz.Snapshot{
		Permissions: map[string]struct{}{
			"audit:read":     {},
			"campaigns:read": {},
		},
		Mask: authz.MaskFull,
	})
	rows := FilterReportCatalog(ctx, ReportCatalogEntries)
	found := false
	for _, row := range rows {
		if row.Key == "fraud-evidence-pack" {
			found = true
		}
	}
	assert.True(t, found)
}
