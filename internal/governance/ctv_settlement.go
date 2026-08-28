package governance

import (
	"context"
	"fmt"

	"ad-event-processor/internal/domain"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

type CTVSettlementHost interface {
	Pool() *pgxpool.Pool
}

func ApplyCTVSettlement(
	ctx context.Context,
	host CTVSettlementHost,
	settlementID string,
	customerID, campaignID uuid.UUID,
	spendMicro int64,
) (domain.CTVSettlementResult, error) {
	var out domain.CTVSettlementResult
	if host == nil || host.Pool() == nil {
		return out, fmt.Errorf("service unavailable")
	}
	if settlementID == "" || spendMicro <= 0 {
		return out, fmt.Errorf("invalid ctv settlement input")
	}
	return domain.ApplyCTVSettlement(ctx, host.Pool(), settlementID, customerID, campaignID, spendMicro)
}
