package reports

import (
	"context"

	"ad-event-processor/internal/controlplane/authz"
)

const permCampaignsReadMasked = "campaigns:read:masked"

var reportPermsCampaignRead = []string{"campaigns:read", permCampaignsReadMasked}

var reportPermsFraudOperator = []string{"audit:read", "campaigns:read"}

func ReportPermsFraudCustomer() []string {
	return []string{"audit:read", "campaigns:read", permCampaignsReadMasked}
}

func maskLevelFromContext(ctx context.Context) authz.MaskLevel {
	snap, ok := authz.SnapshotFromContext(ctx)
	if !ok {
		return authz.MaskMasked
	}
	return snap.Mask
}

func scrubFraudBreakdownRow(ctx context.Context, row FraudBreakdownRowDTO) FraudBreakdownRowDTO {
	if maskLevelFromContext(ctx) == authz.MaskFull {
		return row
	}
	category, label := FraudReasonToCategory(row.FraudReason)
	out := row
	out.FraudReason = ""
	out.PlacementID = ""
	out.FraudCategory = category
	out.FraudCategoryLabel = label
	return out
}

func scrubFraudBreakdownRows(ctx context.Context, rows []FraudBreakdownRowDTO) []FraudBreakdownRowDTO {
	if maskLevelFromContext(ctx) == authz.MaskFull {
		return rows
	}
	out := make([]FraudBreakdownRowDTO, len(rows))
	for i := range rows {
		out[i] = scrubFraudBreakdownRow(ctx, rows[i])
	}
	return out
}
