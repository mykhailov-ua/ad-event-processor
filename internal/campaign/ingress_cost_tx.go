package campaign

import (
	"context"
	"encoding/json"
	"fmt"

	"ad-event-processor/internal/domain"
	db "ad-event-processor/internal/domain/db"

	"github.com/google/uuid"
)

func applyCampaignIngressCostPatchTx(
	ctx context.Context,
	q *db.Queries,
	campaignID uuid.UUID,
	cfg IngressCostConfigDTO,
) error {
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
	_, err = q.UpdateCampaignIngressCostConfig(ctx, db.UpdateCampaignIngressCostConfigParams{
		ID:                domain.ToUUID(campaignID),
		IngressCostConfig: raw,
	})
	return err
}
