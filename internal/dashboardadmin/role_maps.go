package dashboardadmin

import (
	"context"

	"ad-event-processor/internal/domain"
	"ad-event-processor/internal/fraudadmin"
	"ad-event-processor/internal/ledger"
	"ad-event-processor/internal/opsadmin"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

func SumDisputeExposure(ctx context.Context, pool *pgxpool.Pool, customerID uuid.UUID) int64 {
	if pool == nil || customerID == uuid.Nil {
		return 0
	}
	var exposure int64
	_ = pool.QueryRow(ctx, `
		SELECT COALESCE(SUM(d.amount_micro), 0)
		FROM payment.payment_disputes d
		JOIN payment.payment_intents i ON i.id = d.payment_intent_id
		WHERE i.customer_id = $1
		  AND d.status IN ('OPEN', 'FUNDS_WITHDRAWN')`,
		domain.ToUUID(customerID),
	).Scan(&exposure)
	return exposure
}

func BillingStatementFromLines(lines []ledger.InvoiceLineDTO, taxMicro, closingBalanceMicro, invoiceTotalMicro int64) BillingStatement {
	return BillingStatement{
		Lines:               lines,
		TaxMicro:            taxMicro,
		ClosingBalanceMicro: closingBalanceMicro,
		InvoiceTotalMicro:   invoiceTotalMicro,
	}
}

func BillingInvariantFrom(ok bool, diffMicro int64) BillingInvariant {
	return BillingInvariant{OK: ok, DiffMicro: diffMicro}
}

func FraudMLSnapshotFrom(snap fraudadmin.FraudMLSnapshot) FraudMLSnapshot {
	return FraudMLSnapshot{
		VersionID: snap.VersionID, ArtifactHash: snap.ArtifactHash, Precision: snap.Precision, Recall: snap.Recall,
		DriftDetected: snap.DriftDetected, DriftSummary: snap.DriftSummary, EvalGeneratedAt: snap.EvalGeneratedAt,
		EvalStatus: snap.EvalStatus, EvalStale: snap.EvalStale, LabelMethod: snap.LabelMethod, ShardsConsistent: snap.ShardsConsistent,
	}
}

func MLManualLabelsFrom(rows []fraudadmin.MLManualLabelDTO) []MLManualLabelDTO {
	out := make([]MLManualLabelDTO, len(rows))
	for i := range rows {
		out[i] = MLManualLabelDTO{
			IPHash: rows[i].IPHash, Label: rows[i].Label, Reason: rows[i].Reason,
			Source: rows[i].Source, CreatedAt: rows[i].CreatedAt,
		}
	}
	return out
}

func EdgeMetricsFrom(panel opsadmin.EdgeMetricsPanelDTO) EdgeMetricsPanelDTO {
	return EdgeMetricsPanelDTO{
		UpdatedAt: panel.UpdatedAt, IngressH1: panel.IngressH1, IngressH2: panel.IngressH2, IngressH3: panel.IngressH3,
		BodyStream: panel.BodyStream, BodyPeek: panel.BodyPeek, BodyRead: panel.BodyRead,
		Blocked: panel.Blocked, TarpitTotal: panel.TarpitTotal, BlacklistStale: panel.BlacklistStale,
	}
}
