package controlplane

import (
	"context"

	"github.com/google/uuid"
)

type fraudLabelsAPIAdapter struct {
	svc *Service
}

func (a fraudLabelsAPIAdapter) ListMLManualLabelsForCustomer(ctx context.Context, customerID uuid.UUID, limit int) ([]MLManualLabelDTO, error) {
	return a.svc.ListMLManualLabelsForCustomer(ctx, customerID, limit)
}

func (a fraudLabelsAPIAdapter) UpsertMLManualLabelForCustomer(ctx context.Context, customerID uuid.UUID, ipHash string, label int, reason string) error {
	return a.svc.UpsertMLManualLabelForCustomer(ctx, customerID, ipHash, label, reason)
}

func (a fraudLabelsAPIAdapter) BulkUpsertMLManualLabelsForCustomer(ctx context.Context, customerID uuid.UUID, rows []FraudManualLabelRow) (int, error) {
	inputs := make([]MLManualLabelInput, len(rows))
	for i, row := range rows {
		inputs[i] = MLManualLabelInput{
			IPHash: row.IPHash,
			Label:  row.Label,
			Reason: row.Reason,
		}
	}
	return a.svc.BulkUpsertMLManualLabelsForCustomer(ctx, customerID, inputs)
}
