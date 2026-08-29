package flow

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
)

type CampaignAssignHost interface {
	PublishCampaignUpdate(ctx context.Context, campaignID string)
}

func flowIDOrNil(id uuid.UUID) any {
	if id == uuid.Nil {
		return nil
	}
	return id
}

func AssignCampaignFlow(ctx context.Context, pool *pgxpool.Pool, host CampaignAssignHost, campaignID, flowID uuid.UUID) error {
	if pool == nil {
		return fmt.Errorf("service unavailable")
	}
	if campaignID == uuid.Nil {
		return fmt.Errorf("campaign id required")
	}
	if flowID != uuid.Nil {
		var one int
		err := pool.QueryRow(ctx, `SELECT 1 FROM flows WHERE id = $1`, flowID).Scan(&one)
		if err != nil {
			if errors.Is(err, pgx.ErrNoRows) {
				return fmt.Errorf("flow not found")
			}
			return err
		}
	}
	tag, err := pool.Exec(ctx, `UPDATE campaigns SET flow_id = $2, updated_at = now() WHERE id = $1 AND deleted_at IS NULL`, campaignID, flowIDOrNil(flowID))
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return fmt.Errorf("campaign not found")
	}
	if host != nil {
		host.PublishCampaignUpdate(ctx, campaignID.String())
	}
	return nil
}

func CampaignFlowID(ctx context.Context, pool *pgxpool.Pool, campaignID uuid.UUID) (string, error) {
	if pool == nil {
		return "", fmt.Errorf("service unavailable")
	}
	var flowID pgtype.UUID
	err := pool.QueryRow(ctx, `SELECT flow_id FROM campaigns WHERE id = $1`, campaignID).Scan(&flowID)
	if err != nil {
		return "", err
	}
	if !flowID.Valid {
		return "", nil
	}
	return uuid.UUID(flowID.Bytes).String(), nil
}
