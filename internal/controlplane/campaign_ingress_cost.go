package controlplane

import (
	"context"
	"encoding/json"
	"fmt"

	"ad-event-processor/internal/domain"
	db "ad-event-processor/internal/domain/db"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

func (s *Service) applyCampaignIngressCostPatch(ctx context.Context, campaignID uuid.UUID, cfg IngressCostConfigDTO) error {
	if s == nil || s.pool == nil {
		return fmt.Errorf("service unavailable")
	}
	raw, err := json.Marshal(cfg)
	if err != nil {
		return fmt.Errorf("invalid ingress_cost_config")
	}
	parsed := domain.ParseIngressCostConfigJSON(raw)
	if cfg.Param != "" && !parsed.Enabled() {
		return fmt.Errorf("invalid ingress_cost_config.param")
	}
	if parsed.MaxMicro < 0 {
		return fmt.Errorf("invalid ingress_cost_config.max_micro")
	}

	err = pgx.BeginFunc(ctx, s.GetPool(), func(tx pgx.Tx) error {
		q := db.New(tx)
		if _, err := q.UpdateCampaignIngressCostConfig(ctx, db.UpdateCampaignIngressCostConfigParams{
			ID:                domain.ToUUID(campaignID),
			IngressCostConfig: raw,
		}); err != nil {
			return err
		}
		return nil
	})
	if err != nil {
		return err
	}
	_ = s.publishCampaignUpdate(ctx, campaignID.String())
	return nil
}
