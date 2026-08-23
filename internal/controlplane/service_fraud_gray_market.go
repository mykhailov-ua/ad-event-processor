package controlplane

import (
	"context"
	"fmt"

	"github.com/bidshard/ad-event-processor/internal/domain"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

func applyGrayMarketGMAPreset(ctx context.Context, tx pgx.Tx, campaignID uuid.UUID) error {
	tag, err := tx.Exec(ctx, `
		UPDATE campaigns
		SET safe_page_enabled = true,
		    attestation_enabled = true,
		    attestation_mode = 'strict',
		    attestation_ttl_sec = CASE WHEN attestation_ttl_sec < 60 THEN 300 ELSE attestation_ttl_sec END,
		    l15_proxy_vpn_block_enabled = true,
		    tls_fingerprint_block_enabled = true,
		    l1_cidr_block_enabled = true,
		    link_signing_enabled = true,
		    updated_at = CURRENT_TIMESTAMP
		WHERE id = $1`, domain.ToUUID(campaignID))
	if err != nil {
		return fmt.Errorf("apply gray_market GMA preset: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return ErrCampaignNotFound
	}
	return nil
}
