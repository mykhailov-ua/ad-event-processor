package controlplane

import (
	"context"
	"fmt"

	"github.com/google/uuid"
)

const maxFlowPaths = 32

func validateFlowPathShape(paths []FlowPathDTO) error {
	if len(paths) == 0 {
		return fmt.Errorf("paths are required")
	}
	if len(paths) > maxFlowPaths {
		return fmt.Errorf("too many paths (max %d)", maxFlowPaths)
	}
	for i, path := range paths {
		if path.Weight <= 0 {
			return fmt.Errorf("path %d weight must be positive", i+1)
		}
		if len(path.Landers) == 0 {
			return fmt.Errorf("path %d requires at least one lander", i+1)
		}
		if len(path.Offers) == 0 {
			return fmt.Errorf("path %d requires at least one offer", i+1)
		}
		for j, lander := range path.Landers {
			if lander.LanderID == uuid.Nil {
				return fmt.Errorf("path %d lander %d id is required", i+1, j+1)
			}
			if lander.Weight <= 0 {
				return fmt.Errorf("path %d lander %d weight must be positive", i+1, j+1)
			}
		}
		for j, offer := range path.Offers {
			if offer.OfferID == uuid.Nil {
				return fmt.Errorf("path %d offer %d id is required", i+1, j+1)
			}
			if offer.Weight <= 0 {
				return fmt.Errorf("path %d offer %d weight must be positive", i+1, j+1)
			}
		}
	}
	return nil
}

func (s *Service) validateFlowPaths(ctx context.Context, paths []FlowPathDTO) error {
	if err := validateFlowPathShape(paths); err != nil {
		return err
	}
	landerIDs := make([]uuid.UUID, 0)
	offerIDs := make([]uuid.UUID, 0)
	landerSet := make(map[uuid.UUID]struct{})
	offerSet := make(map[uuid.UUID]struct{})
	for _, path := range paths {
		for _, lander := range path.Landers {
			if _, ok := landerSet[lander.LanderID]; !ok {
				landerSet[lander.LanderID] = struct{}{}
				landerIDs = append(landerIDs, lander.LanderID)
			}
		}
		for _, offer := range path.Offers {
			if _, ok := offerSet[offer.OfferID]; !ok {
				offerSet[offer.OfferID] = struct{}{}
				offerIDs = append(offerIDs, offer.OfferID)
			}
		}
	}
	if err := s.validateFlowLanderIDs(ctx, landerIDs); err != nil {
		return err
	}
	return s.validateFlowOfferIDs(ctx, offerIDs)
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
