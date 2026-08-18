package controlplane

import (
	"context"

	"github.com/google/uuid"
)

type fraudIntegrationsAPIAdapter struct {
	svc *Service
}

func (a fraudIntegrationsAPIAdapter) ListFraudIntegrationsForCustomer(ctx context.Context, customerID uuid.UUID) ([]FraudIntegrationDTO, error) {
	return a.svc.ListFraudIntegrationsForCustomer(ctx, customerID)
}
