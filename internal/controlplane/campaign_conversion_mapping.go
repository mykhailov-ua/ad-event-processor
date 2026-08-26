package controlplane

import (
	"context"
	"fmt"
	"strings"

	db "ad-event-processor/internal/domain/db"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
)

type ConversionMappingDTO struct {
	InboundStatus string `json:"inbound_status"`
	GoalName      string `json:"goal_name"`
	PayoutMicro   int64  `json:"payout_micro"`
}

type ConversionMappingListResponse struct {
	Mappings []ConversionMappingDTO `json:"mappings"`
}

type ReplaceConversionMappingsRequest struct {
	Mappings []ConversionMappingDTO `json:"mappings"`
}

func (s *Service) ListCampaignConversionMappings(ctx context.Context, campaignID uuid.UUID) ([]ConversionMappingDTO, error) {
	if s == nil || s.pool == nil {
		return nil, fmt.Errorf("service unavailable")
	}
	rows, err := db.New(s.pool).ListConversionMappingsByCampaign(ctx, domainToPgUUID(campaignID))
	if err != nil {
		return nil, err
	}
	out := make([]ConversionMappingDTO, 0, len(rows))
	for i := range rows {
		out = append(out, conversionMappingToDTO(&rows[i]))
	}
	return out, nil
}

func (s *Service) ReplaceCampaignConversionMappings(ctx context.Context, campaignID uuid.UUID, mappings []ConversionMappingDTO) ([]ConversionMappingDTO, error) {
	if s == nil || s.pool == nil {
		return nil, fmt.Errorf("service unavailable")
	}
	normalized, err := normalizeConversionMappings(mappings)
	if err != nil {
		return nil, err
	}
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return nil, err
	}
	defer func() { _ = tx.Rollback(ctx) }()
	q := db.New(tx)
	if err := q.DeleteConversionMappingsByCampaign(ctx, domainToPgUUID(campaignID)); err != nil {
		return nil, err
	}
	for i := range normalized {
		row := &normalized[i]
		if err := q.InsertConversionMapping(ctx, db.InsertConversionMappingParams{
			CampaignID:    domainToPgUUID(campaignID),
			InboundStatus: row.InboundStatus,
			GoalName:      row.GoalName,
			PayoutMicro:   row.PayoutMicro,
		}); err != nil {
			return nil, err
		}
	}
	if err := tx.Commit(ctx); err != nil {
		return nil, err
	}
	return normalized, nil
}

func normalizeConversionMappings(mappings []ConversionMappingDTO) ([]ConversionMappingDTO, error) {
	if len(mappings) == 0 {
		return []ConversionMappingDTO{}, nil
	}
	seen := make(map[string]struct{}, len(mappings))
	out := make([]ConversionMappingDTO, 0, len(mappings))
	for i := range mappings {
		row := mappings[i]
		status := strings.ToLower(strings.TrimSpace(row.InboundStatus))
		if status == "" {
			return nil, fmt.Errorf("inbound_status is required")
		}
		goal := strings.TrimSpace(row.GoalName)
		if goal == "" {
			goal = status
		}
		if row.PayoutMicro < 0 {
			return nil, fmt.Errorf("payout_micro must be non-negative")
		}
		if _, dup := seen[status]; dup {
			return nil, fmt.Errorf("duplicate inbound_status %q", status)
		}
		seen[status] = struct{}{}
		out = append(out, ConversionMappingDTO{
			InboundStatus: status,
			GoalName:      goal,
			PayoutMicro:   row.PayoutMicro,
		})
	}
	return out, nil
}

func conversionMappingToDTO(row *db.CampaignConversionMapping) ConversionMappingDTO {
	if row == nil {
		return ConversionMappingDTO{}
	}
	return ConversionMappingDTO{
		InboundStatus: row.InboundStatus,
		GoalName:      row.GoalName,
		PayoutMicro:   row.PayoutMicro,
	}
}

func domainToPgUUID(id uuid.UUID) pgtype.UUID {
	return pgtype.UUID{Bytes: id, Valid: true}
}

type AffiliateStatusPresetDTO struct {
	Name     string                          `json:"name"`
	Statuses []AffiliateStatusPresetEntryDTO `json:"statuses"`
}

type AffiliateStatusPresetEntryDTO struct {
	InboundStatus string `json:"inbound_status"`
	GoalName      string `json:"goal_name"`
}
