package outbox

import (
	"context"
	"fmt"

	"ad-event-processor/internal/domain"
	"ad-event-processor/pkg/coldpath"

	"github.com/google/uuid"
)

func (w *Worker) handleApplyGTVSettlement(ctx context.Context, payload []byte) error {
	p, err := coldpath.UnmarshalStrict[CTVGtaxSettlementPayload](payload)
	if err != nil {
		return err
	}
	customerID, err := uuid.Parse(p.CustomerID)
	if err != nil {
		return fmt.Errorf("invalid customer id: %w", err)
	}
	campaignID, err := uuid.Parse(p.CampaignID)
	if err != nil {
		return fmt.Errorf("invalid campaign id: %w", err)
	}
	_, err = domain.ApplyCTVSettlement(ctx, w.host.Pool(), p.SettlementID, customerID, campaignID, p.SpendMicro)
	return err
}
