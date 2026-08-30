package reportjob

import (
	"context"

	"ad-event-processor/internal/controlplane/authz"
)

const (
	exportProfileOperatorFull  = "operator_full"
	exportProfileBuyerSummary  = "buyer_summary"
	exportProfileSupportMasked = "support_masked"
)

func resolveExportRedactionProfile(ctx context.Context) string {
	// HTTP-edge profile selection; column scrub runs inside reports.WriteReport, not here.
	snap, ok := authz.SnapshotFromContext(ctx)
	if !ok {
		return exportProfileOperatorFull
	}
	switch snap.Mask {
	case authz.MaskMasked:
		return exportProfileBuyerSummary
	case authz.MaskFull:
		if snap.Has("audit:read") {
			return exportProfileOperatorFull
		}
		return exportProfileBuyerSummary
	default:
		return exportProfileSupportMasked
	}
}
