package controlplane

import (
	"context"
	"errors"
	"fmt"

	"github.com/bidshard/ad-event-processor/internal/domain"
	db "github.com/bidshard/ad-event-processor/internal/domain/db"
	"github.com/bidshard/ad-event-processor/pkg/coldpath"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

type CTVSettlementResult = domain.CTVSettlementResult

type ctvGtaxSettlementPayload struct {
	SettlementID string `json:"settlement_id"`
	CustomerID   string `json:"customer_id"`
	CampaignID   string `json:"campaign_id"`
	SpendMicro   int64  `json:"spend_micro"`
}

func (s *Service) ApplyCTVSettlement(
	ctx context.Context,
	settlementID string,
	customerID, campaignID uuid.UUID,
	spendMicro int64,
) (CTVSettlementResult, error) {
	var out CTVSettlementResult
	if s == nil || s.pool == nil {
		return out, fmt.Errorf("service unavailable")
	}
	if settlementID == "" || spendMicro <= 0 {
		return out, fmt.Errorf("invalid ctv settlement input")
	}

	result, err := domain.ApplyCTVSettlement(ctx, s.pool, settlementID, customerID, campaignID, spendMicro)
	if err != nil {
		if errors.Is(err, domain.ErrCTVSettlementCampaignNotFound) {
			return out, ErrCampaignNotFound
		}
		return out, err
	}

	if result.Applied {
		payload, marshalErr := coldpath.MarshalOutbox(ctvGtaxSettlementPayload{
			SettlementID: settlementID,
			CustomerID:   customerID.String(),
			CampaignID:   campaignID.String(),
			SpendMicro:   spendMicro,
		})
		if marshalErr != nil {
			return result, fmt.Errorf("marshal gtv settlement payload: %w", marshalErr)
		}
		enqueueErr := pgx.BeginFunc(ctx, s.pool, func(tx pgx.Tx) error {
			_, err := db.New(tx).CreateOutboxEvent(ctx, db.CreateOutboxEventParams{
				EventType: "APPLY_GTV_SETTLEMENT",
				Payload:   payload,
			})
			return err
		})
		if enqueueErr != nil {
			return result, fmt.Errorf("enqueue gtv settlement: %w", enqueueErr)
		}
	}

	return result, nil
}

func (worker *OutboxWorker) handleApplyGTVSettlement(ctx context.Context, payload []byte) error {
	p, err := coldpath.UnmarshalStrict[ctvGtaxSettlementPayload](payload)
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
	_, err = domain.ApplyCTVSettlement(ctx, worker.svc.GetPool(), p.SettlementID, customerID, campaignID, p.SpendMicro)
	return err
}
