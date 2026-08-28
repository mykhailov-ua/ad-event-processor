package payment

import (
	"context"
	"errors"
	"fmt"

	"ad-event-processor/internal/domain"
	db "ad-event-processor/internal/domain/db"
	"ad-event-processor/pkg/coldpath"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type CTVSettlementHost interface {
	Pool() *pgxpool.Pool
	ErrCampaignNotFound() error
}

type ctvGtaxSettlementPayload struct {
	SettlementID string `json:"settlement_id"`
	CustomerID   string `json:"customer_id"`
	CampaignID   string `json:"campaign_id"`
	SpendMicro   int64  `json:"spend_micro"`
}

func ApplyCTVSettlement(
	ctx context.Context,
	host CTVSettlementHost,
	settlementID string,
	customerID, campaignID uuid.UUID,
	spendMicro int64,
) (domain.CTVSettlementResult, error) {
	var out domain.CTVSettlementResult
	if host == nil || host.Pool() == nil {
		return out, fmt.Errorf("service unavailable")
	}
	if settlementID == "" || spendMicro <= 0 {
		return out, fmt.Errorf("invalid ctv settlement input")
	}

	result, err := domain.ApplyCTVSettlement(ctx, host.Pool(), settlementID, customerID, campaignID, spendMicro)
	if err != nil {
		if errors.Is(err, domain.ErrCTVSettlementCampaignNotFound) {
			return out, host.ErrCampaignNotFound()
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
		enqueueErr := pgx.BeginFunc(ctx, host.Pool(), func(tx pgx.Tx) error {
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
