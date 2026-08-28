package controlplane

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"ad-event-processor/internal/flow"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
)

var _ flow.Host = (*Service)(nil)

func (s *Service) FlowStore() *flow.Store {
	if s == nil {
		return nil
	}
	if s.flowStore == nil {
		s.flowStore = flow.NewStore(s.pool, s)
	}
	return s.flowStore
}

func (s *Service) LanderPublicBase(ctx context.Context) string {
	return s.landerPublicBase(ctx)
}

func (s *Service) PublishFlowReload(ctx context.Context) error {
	if s == nil {
		return nil
	}
	channel := flowReloadChannel
	if s.cfg != nil && strings.TrimSpace(s.cfg.FlowReloadChannel) != "" {
		channel = strings.TrimSpace(s.cfg.FlowReloadChannel)
	}
	return publishFlowReload(ctx, s.redisShards, channel)
}

func (s *Service) CreateLander(ctx context.Context, req CreateLanderRequest) (LanderDTO, error) {
	return s.FlowStore().CreateLander(ctx, req)
}

func (s *Service) ListLanders(ctx context.Context) ([]LanderDTO, error) {
	return s.FlowStore().ListLanders(ctx)
}

func (s *Service) CreateOffer(ctx context.Context, req CreateOfferRequest) (OfferDTO, error) {
	return s.FlowStore().CreateOffer(ctx, req)
}

func (s *Service) ListOffers(ctx context.Context) ([]OfferDTO, error) {
	return s.FlowStore().ListOffers(ctx)
}

func (s *Service) CreateFlow(ctx context.Context, req CreateFlowRequest) (FlowDTO, error) {
	return s.FlowStore().CreateFlow(ctx, req)
}

func (s *Service) ListFlows(ctx context.Context) ([]FlowDTO, error) {
	return s.FlowStore().ListFlows(ctx)
}

func (s *Service) GetFlow(ctx context.Context, flowID uuid.UUID) (FlowDTO, error) {
	return s.FlowStore().GetFlow(ctx, flowID)
}

func (s *Service) UpdateFlow(ctx context.Context, flowID uuid.UUID, req UpdateFlowRequest) (FlowDTO, error) {
	return s.FlowStore().UpdateFlow(ctx, flowID, req)
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
