package fraudadmin

import (
	"bytes"
	"context"
	"encoding/hex"
	"fmt"
	"strings"

	"ad-event-processor/internal/campaign"
	"ad-event-processor/internal/domain"
	db "ad-event-processor/internal/domain/db"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

func ApplyFraudScoringOverrideForCustomer(ctx context.Context, host OverridesHost, customerID uuid.UUID, req FraudOverrideRequest) error {
	if customerID == uuid.Nil {
		return ValidationError("customer_id is required")
	}

	var campaignIDRaw string
	if req.CampaignID != nil {
		campaignIDRaw = strings.TrimSpace(*req.CampaignID)
		if campaignIDRaw != "" {
			if _, err := uuid.Parse(campaignIDRaw); err != nil {
				return ValidationError("invalid campaign_id format")
			}
		}
	}

	ip := ""
	if req.IP != nil {
		ip = strings.TrimSpace(*req.IP)
	}
	ipHash := ""
	if req.IPHash != nil {
		ipHash = strings.TrimSpace(strings.ToLower(*req.IPHash))
		if ipHash != "" {
			if err := ValidateMLIPHash(ipHash); err != nil {
				return err
			}
		}
	}
	if campaignIDRaw == "" && ip == "" && ipHash == "" {
		return ValidationError("at least one of campaign_id, ip, or ip_hash is required")
	}

	if host == nil || host.OverridesPool() == nil {
		return fmt.Errorf("postgres pool not configured")
	}

	var campaignIDPtr *string
	if campaignIDRaw != "" {
		campID, err := uuid.Parse(campaignIDRaw)
		if err != nil {
			return ValidationError("invalid campaign_id")
		}
		if err := assertCampaignOwnedByCustomer(ctx, host.OverridesPool(), customerID, campID); err != nil {
			return err
		}
		campaignIDPtr = &campaignIDRaw
	}

	if ip == "" && ipHash != "" {
		resolved, err := resolveBlacklistIPByHash(ctx, host, ipHash)
		if err != nil {
			return err
		}
		ip = resolved
	}

	hasCampaign := campaignIDPtr != nil
	hasIP := ip != ""
	if !hasCampaign && !hasIP {
		if ipHash != "" {
			return ValidationError("ip_hash not found on fraud blacklist")
		}
		return ValidationError("at least one of campaign_id, ip, or ip_hash is required")
	}

	override := FraudScoringOverrideRequest{}
	if hasCampaign {
		override.CampaignID = campaignIDPtr
	}
	if hasIP {
		override.IP = &ip
	}
	return ApplyFraudScoringOverride(ctx, host, override)
}

func ApplyFraudScoringOverride(ctx context.Context, host OverridesHost, req FraudScoringOverrideRequest) error {
	hasCampaign := req.CampaignID != nil && strings.TrimSpace(*req.CampaignID) != ""
	hasIP := req.IP != nil && strings.TrimSpace(*req.IP) != ""
	if !hasCampaign && !hasIP {
		return ValidationError("at least one of campaign_id or ip is required")
	}
	if host == nil || host.OverridesPool() == nil {
		return fmt.Errorf("postgres pool not configured")
	}
	return pgx.BeginFunc(ctx, host.OverridesPool(), func(tx pgx.Tx) error {
		q := db.New(tx)
		uid := host.OverrideActorID(ctx)

		if req.CampaignID != nil && *req.CampaignID != "" {
			campUUID, err := uuid.Parse(*req.CampaignID)
			if err != nil {
				return ValidationError("invalid campaign_id format")
			}

			host.OverrideAuditLog(ctx, q, uid, "FRAUD_CLEAR_BOOST", "campaign", &campUUID, map[string]string{"campaign_id": *req.CampaignID}, nil)

			if err := host.OverrideEnqueueClearBoost(ctx, q, *req.CampaignID); err != nil {
				return err
			}
		}

		if req.IP != nil && *req.IP != "" {
			err := q.DeleteBlacklistIP(ctx, *req.IP)
			if err != nil {
				return err
			}

			host.OverrideAuditLog(ctx, q, uid, "FRAUD_REMOVE_FALSE_POSITIVE", "system", nil, map[string]string{"ip": *req.IP}, nil)

			if err := host.OverrideEnqueueBlacklistRemove(ctx, q, *req.IP); err != nil {
				return err
			}
		}

		return nil
	})
}

func assertCampaignOwnedByCustomer(ctx context.Context, pool interface {
	QueryRow(context.Context, string, ...any) pgx.Row
}, customerID, campaignID uuid.UUID,
) error {
	var exists bool
	err := pool.QueryRow(ctx, `
		SELECT EXISTS(SELECT 1 FROM campaigns WHERE id = $1 AND customer_id = $2)`,
		domain.ToUUID(campaignID), domain.ToUUID(customerID),
	).Scan(&exists)
	if err != nil {
		return err
	}
	if !exists {
		return campaign.ErrCampaignNotFound
	}
	return nil
}

func resolveBlacklistIPByHash(ctx context.Context, host OverridesHost, ipHash string) (string, error) {
	if host == nil || host.OverridesPool() == nil {
		return "", fmt.Errorf("postgres pool not configured")
	}
	want, err := hex.DecodeString(ipHash)
	if err != nil || len(want) != 16 {
		return "", ValidationError("ip_hash must be 32 hex characters")
	}

	rows, err := host.OverridesPool().Query(ctx, `SELECT ip FROM ip_blacklist WHERE reason = 'fraud'`)
	if err != nil {
		return "", fmt.Errorf("query ip_blacklist: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		var ip string
		if err := rows.Scan(&ip); err != nil {
			return "", err
		}
		h, err := host.OverrideHashIP(ip)
		if err != nil {
			return "", err
		}
		if bytes.Equal(h[:], want) {
			return ip, nil
		}
	}
	if err := rows.Err(); err != nil {
		return "", err
	}
	return "", nil
}
