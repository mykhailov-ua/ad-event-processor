package controlplane

import (
	"bytes"
	"context"
	"encoding/hex"
	"fmt"
	"strings"

	"github.com/bidshard/ad-event-processor/internal/domain"
	"github.com/bidshard/ad-event-processor/pkg/piihash"

	"github.com/google/uuid"
)

type FraudOverrideRequest struct {
	CampaignID *string `json:"campaign_id,omitempty"`
	IP         *string `json:"ip,omitempty"`
	IPHash     *string `json:"ip_hash,omitempty"`
}

type FraudOverridesService interface {
	ApplyFraudScoringOverrideForCustomer(ctx context.Context, customerID uuid.UUID, req FraudOverrideRequest) error
}

func (s *Service) ApplyFraudScoringOverrideForCustomer(ctx context.Context, customerID uuid.UUID, req FraudOverrideRequest) error {
	if customerID == uuid.Nil {
		return errValidation("customer_id is required")
	}

	var campaignIDRaw string
	if req.CampaignID != nil {
		campaignIDRaw = strings.TrimSpace(*req.CampaignID)
		if campaignIDRaw != "" {
			if _, err := uuid.Parse(campaignIDRaw); err != nil {
				return errValidation("invalid campaign_id format")
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
			if err := validateMLIPHash(ipHash); err != nil {
				return err
			}
		}
	}
	if campaignIDRaw == "" && ip == "" && ipHash == "" {
		return errValidation("at least one of campaign_id, ip, or ip_hash is required")
	}

	if s == nil || s.GetPool() == nil {
		return fmt.Errorf("postgres pool not configured")
	}

	var campaignIDPtr *string
	if campaignIDRaw != "" {
		campID, err := uuid.Parse(campaignIDRaw)
		if err != nil {
			return errValidation("invalid campaign_id")
		}
		if err := s.assertCampaignOwnedByCustomer(ctx, customerID, campID); err != nil {
			return err
		}
		campaignIDPtr = &campaignIDRaw
	}

	if ip == "" && ipHash != "" {
		resolved, err := s.resolveBlacklistIPByHash(ctx, ipHash)
		if err != nil {
			return err
		}
		ip = resolved
	}

	hasCampaign := campaignIDPtr != nil
	hasIP := ip != ""
	if !hasCampaign && !hasIP {
		if ipHash != "" {
			return errValidation("ip_hash not found on fraud blacklist")
		}
		return errValidation("at least one of campaign_id, ip, or ip_hash is required")
	}

	override := FraudScoringOverrideRequest{}
	if hasCampaign {
		override.CampaignID = campaignIDPtr
	}
	if hasIP {
		override.IP = &ip
	}
	return s.ApplyFraudScoringOverride(ctx, override)
}

func (s *Service) assertCampaignOwnedByCustomer(ctx context.Context, customerID, campaignID uuid.UUID) error {
	var exists bool
	err := s.GetPool().QueryRow(ctx, `
		SELECT EXISTS(SELECT 1 FROM campaigns WHERE id = $1 AND customer_id = $2)`,
		domain.ToUUID(campaignID), domain.ToUUID(customerID),
	).Scan(&exists)
	if err != nil {
		return err
	}
	if !exists {
		return ErrCampaignNotFound
	}
	return nil
}

func (s *Service) resolveBlacklistIPByHash(ctx context.Context, ipHash string) (string, error) {
	if s == nil || s.GetPool() == nil {
		return "", fmt.Errorf("postgres pool not configured")
	}
	want, err := hex.DecodeString(ipHash)
	if err != nil || len(want) != 16 {
		return "", errValidation("ip_hash must be 32 hex characters")
	}
	hasher, err := piihash.NewFromConfig(s.cfg)
	if err != nil {
		return "", fmt.Errorf("piihash: %w", err)
	}

	rows, err := s.GetPool().Query(ctx, `SELECT ip FROM ip_blacklist WHERE reason = 'fraud'`)
	if err != nil {
		return "", fmt.Errorf("query ip_blacklist: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		var ip string
		if err := rows.Scan(&ip); err != nil {
			return "", err
		}
		h := hasher.HashIP(ip)
		if bytes.Equal(h[:], want) {
			return ip, nil
		}
	}
	if err := rows.Err(); err != nil {
		return "", err
	}
	return "", nil
}
