package controlplane

import (
	"context"

	"github.com/google/uuid"
)

type fraudDecisionsAPIAdapter struct {
	svc *Service
}

func (a fraudDecisionsAPIAdapter) ExplainFraudDecision(ctx context.Context, customerID uuid.UUID, ipHash string, campaignID *uuid.UUID, hours int) (FraudDecisionDTO, error) {
	return a.svc.ExplainFraudDecision(ctx, customerID, ipHash, campaignID, hours)
}
