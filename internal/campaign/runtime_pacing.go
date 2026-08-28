package campaign

import (
	"context"
	"fmt"

	"ad-event-processor/internal/controlplane/authz"
	"ad-event-processor/internal/domain"
	db "ad-event-processor/internal/domain/db"
	"ad-event-processor/pkg/coldpath"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type pacingOutboxPayload struct {
	CampaignID string `json:"campaign_id"`
	PacingMode string `json:"pacing_mode"`
}

type auditPacingChange struct {
	OldPacingMode string `json:"old_pacing_mode"`
	NewPacingMode string `json:"new_pacing_mode"`
}

func updateCampaignPacing(ctx context.Context, pool *pgxpool.Pool, fx Effects, campaignID uuid.UUID, newMode string) (CampaignDTO, error) {
	pacing, err := parsePacingMode(newMode)
	if err != nil {
		return CampaignDTO{}, err
	}
	if pool == nil || fx == nil {
		return CampaignDTO{}, errServiceUnavailable()
	}

	var updatedCamp db.Campaign
	err = pgx.BeginFunc(ctx, pool, func(tx pgx.Tx) error {
		q := db.New(tx)

		camp, err := q.GetCampaignForUpdate(ctx, domain.ToUUID(campaignID))
		if err != nil {
			return mapCampaignStoreError(err)
		}

		updatedCamp, err = q.UpdateCampaignPacing(ctx, db.UpdateCampaignPacingParams{
			ID:         domain.ToUUID(campaignID),
			PacingMode: pacing,
		})
		if err != nil {
			return fmt.Errorf("failed to update campaign pacing: %w", err)
		}

		adminID := uuid.Nil
		if user, ok := authz.GetUser(ctx); ok {
			adminID = user.UserID
		}

		fx.AuditLog(ctx, q, adminID, "UPDATE_CAMPAIGN_PACING", "campaign", &campaignID, auditPacingChange{
			OldPacingMode: string(camp.PacingMode),
			NewPacingMode: string(pacing),
		}, nil)

		payloadBytes, err := coldpath.MarshalOutbox(pacingOutboxPayload{
			CampaignID: campaignID.String(),
			PacingMode: string(pacing),
		})
		if err != nil {
			return fmt.Errorf("marshal update campaign pacing outbox payload: %w", err)
		}

		_, err = q.CreateOutboxEvent(ctx, db.CreateOutboxEventParams{
			EventType: "UPDATE_CAMPAIGN_PACING",
			Payload:   payloadBytes,
		})
		if err != nil {
			return fmt.Errorf("failed to create outbox event: %w", err)
		}

		return nil
	})
	if err != nil {
		return CampaignDTO{}, err
	}

	return scrubCampaignDTO(ctx, updatedCamp), nil
}

func parsePacingMode(newMode string) (db.PacingModeType, error) {
	switch newMode {
	case "ASAP":
		return db.PacingModeTypeASAP, nil
	case "EVEN", "off", "OFF":
		return db.PacingModeTypeEVEN, nil
	case "VPP", "vpp":
		return db.PacingModeTypeVPP, nil
	default:
		return "", fmt.Errorf("%w: %s", ErrInvalidPacingMode, newMode)
	}
}
