package controlplane

import (
	"context"
	"fmt"

	"github.com/bidshard/ad-event-processor/internal/domain"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

func applySocialInAppPreset(ctx context.Context, tx pgx.Tx, campaignID uuid.UUID) error {
	tag, err := tx.Exec(ctx, `
		UPDATE campaigns
		SET social_in_app_enabled = true,
		    tls_fingerprint_block_enabled = true,
		    l15_proxy_vpn_block_enabled = true,
		    conn_type_policy = 'mobile_only',
		    updated_at = CURRENT_TIMESTAMP
		WHERE id = $1`, domain.ToUUID(campaignID))
	if err != nil {
		return fmt.Errorf("apply social_in_app preset: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return ErrCampaignNotFound
	}
	return nil
}
