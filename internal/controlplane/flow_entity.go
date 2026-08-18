package controlplane

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
)

func (s *Service) CreateLander(ctx context.Context, req CreateLanderRequest) (LanderDTO, error) {
	if s == nil || s.pool == nil {
		return LanderDTO{}, fmt.Errorf("service unavailable")
	}
	name := strings.TrimSpace(req.Name)
	url := strings.TrimSpace(req.URL)
	if name == "" || url == "" {
		return LanderDTO{}, fmt.Errorf("name and url are required")
	}
	var dto LanderDTO
	err := s.pool.QueryRow(ctx, `
		INSERT INTO landers (name, url) VALUES ($1, $2)
		RETURNING id, name, url, created_at`, name, url).Scan(&dto.ID, &dto.Name, &dto.URL, &dto.CreatedAt)
	if err != nil {
		return LanderDTO{}, err
	}
	return dto, nil
}

func (s *Service) ListLanders(ctx context.Context) ([]LanderDTO, error) {
	if s == nil || s.pool == nil {
		return nil, fmt.Errorf("service unavailable")
	}
	rows, err := s.pool.Query(ctx, `SELECT id, name, url, created_at FROM landers ORDER BY created_at DESC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []LanderDTO
	for rows.Next() {
		var dto LanderDTO
		if err := rows.Scan(&dto.ID, &dto.Name, &dto.URL, &dto.CreatedAt); err != nil {
			return nil, err
		}
		out = append(out, dto)
	}
	return out, rows.Err()
}

func (s *Service) CreateOffer(ctx context.Context, req CreateOfferRequest) (OfferDTO, error) {
	if s == nil || s.pool == nil {
		return OfferDTO{}, fmt.Errorf("service unavailable")
	}
	name := strings.TrimSpace(req.Name)
	url := strings.TrimSpace(req.URL)
	if name == "" || url == "" {
		return OfferDTO{}, fmt.Errorf("name and url are required")
	}
	var dto OfferDTO
	err := s.pool.QueryRow(ctx, `
		INSERT INTO offers (name, url) VALUES ($1, $2)
		RETURNING id, name, url, created_at`, name, url).Scan(&dto.ID, &dto.Name, &dto.URL, &dto.CreatedAt)
	if err != nil {
		return OfferDTO{}, err
	}
	return dto, nil
}

func (s *Service) ListOffers(ctx context.Context) ([]OfferDTO, error) {
	if s == nil || s.pool == nil {
		return nil, fmt.Errorf("service unavailable")
	}
	rows, err := s.pool.Query(ctx, `SELECT id, name, url, created_at FROM offers ORDER BY created_at DESC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []OfferDTO
	for rows.Next() {
		var dto OfferDTO
		if err := rows.Scan(&dto.ID, &dto.Name, &dto.URL, &dto.CreatedAt); err != nil {
			return nil, err
		}
		out = append(out, dto)
	}
	return out, rows.Err()
}

func (s *Service) CreateFlow(ctx context.Context, req CreateFlowRequest) (FlowDTO, error) {
	if s == nil || s.pool == nil {
		return FlowDTO{}, fmt.Errorf("service unavailable")
	}
	name := strings.TrimSpace(req.Name)
	if name == "" {
		return FlowDTO{}, fmt.Errorf("name is required")
	}
	if len(req.Paths) == 0 {
		return FlowDTO{}, fmt.Errorf("paths are required")
	}
	raw, err := json.Marshal(req.Paths)
	if err != nil {
		return FlowDTO{}, err
	}
	var dto FlowDTO
	err = s.pool.QueryRow(ctx, `
		INSERT INTO flows (name, paths) VALUES ($1, $2::jsonb)
		RETURNING id, name, paths, created_at`, name, raw).Scan(&dto.ID, &dto.Name, &dto.Paths, &dto.CreatedAt)
	if err != nil {
		return FlowDTO{}, err
	}
	return dto, nil
}

func (s *Service) ListFlows(ctx context.Context) ([]FlowDTO, error) {
	if s == nil || s.pool == nil {
		return nil, fmt.Errorf("service unavailable")
	}
	rows, err := s.pool.Query(ctx, `SELECT id, name, paths, created_at FROM flows ORDER BY created_at DESC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []FlowDTO
	for rows.Next() {
		var dto FlowDTO
		if err := rows.Scan(&dto.ID, &dto.Name, &dto.Paths, &dto.CreatedAt); err != nil {
			return nil, err
		}
		out = append(out, dto)
	}
	return out, rows.Err()
}

func (s *Service) AssignCampaignFlow(ctx context.Context, campaignID, flowID uuid.UUID) error {
	if s == nil || s.pool == nil {
		return fmt.Errorf("service unavailable")
	}
	if campaignID == uuid.Nil {
		return fmt.Errorf("campaign id required")
	}
	if flowID != uuid.Nil {
		var one int
		err := s.pool.QueryRow(ctx, `SELECT 1 FROM flows WHERE id = $1`, flowID).Scan(&one)
		if err != nil {
			if errors.Is(err, pgx.ErrNoRows) {
				return fmt.Errorf("flow not found")
			}
			return err
		}
	}
	tag, err := s.pool.Exec(ctx, `UPDATE campaigns SET flow_id = $2, updated_at = now() WHERE id = $1 AND deleted_at IS NULL`, campaignID, flowIDOrNil(flowID))
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return fmt.Errorf("campaign not found")
	}
	_ = s.publishCampaignUpdate(ctx, campaignID.String())
	return nil
}

func flowIDOrNil(id uuid.UUID) any {
	if id == uuid.Nil {
		return nil
	}
	return id
}

func (s *Service) campaignFlowID(ctx context.Context, campaignID uuid.UUID) (string, error) {
	if s == nil || s.pool == nil {
		return "", fmt.Errorf("service unavailable")
	}
	var flowID pgtype.UUID
	err := s.pool.QueryRow(ctx, `SELECT flow_id FROM campaigns WHERE id = $1`, campaignID).Scan(&flowID)
	if err != nil {
		return "", err
	}
	if !flowID.Valid {
		return "", nil
	}
	return uuid.UUID(flowID.Bytes).String(), nil
}
