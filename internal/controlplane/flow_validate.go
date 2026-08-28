package controlplane

import (
	"context"
	"fmt"

	"ad-event-processor/internal/flow"

	"github.com/google/uuid"
)

func (s *Service) ValidateCampaignFlowPaths(ctx context.Context, paths []FlowPathDTO) error {
	return s.validateFlowPaths(ctx, paths)
}

func (s *Service) validateFlowPaths(ctx context.Context, paths []FlowPathDTO) error {
	return flow.ValidatePathRefs(ctx, s, paths)
}

func (s *Service) ValidateLanderIDs(ctx context.Context, ids []uuid.UUID) error {
	return s.validateFlowLanderIDs(ctx, ids)
}

func (s *Service) ValidateOfferIDs(ctx context.Context, ids []uuid.UUID) error {
	return s.validateFlowOfferIDs(ctx, ids)
}

func (s *Service) validateFlowLanderIDs(ctx context.Context, ids []uuid.UUID) error {
	if len(ids) == 0 {
		return nil
	}
	rows, err := s.pool.Query(ctx, `
		SELECT id, COALESCE(url, '') != '' OR hosted_asset_id IS NOT NULL
		FROM landers WHERE id = ANY($1)`, ids)
	if err != nil {
		return err
	}
	defer rows.Close()
	found := make(map[uuid.UUID]bool, len(ids))
	for rows.Next() {
		var id uuid.UUID
		var routable bool
		if err := rows.Scan(&id, &routable); err != nil {
			return err
		}
		if !routable {
			return fmt.Errorf("lander %s has no URL or hosted asset", id)
		}
		found[id] = true
	}
	if err := rows.Err(); err != nil {
		return err
	}
	for _, id := range ids {
		if !found[id] {
			return fmt.Errorf("lander %s not found", id)
		}
	}
	return nil
}

func (s *Service) validateFlowOfferIDs(ctx context.Context, ids []uuid.UUID) error {
	if len(ids) == 0 {
		return nil
	}
	rows, err := s.pool.Query(ctx, `
		SELECT id FROM offers WHERE id = ANY($1) AND COALESCE(url, '') != ''`, ids)
	if err != nil {
		return err
	}
	defer rows.Close()
	found := make(map[uuid.UUID]struct{}, len(ids))
	for rows.Next() {
		var id uuid.UUID
		if err := rows.Scan(&id); err != nil {
			return err
		}
		found[id] = struct{}{}
	}
	if err := rows.Err(); err != nil {
		return err
	}
	for _, id := range ids {
		if _, ok := found[id]; !ok {
			return fmt.Errorf("offer %s not found", id)
		}
	}
	return nil
}

var _ flow.PathRefChecker = (*Service)(nil)
