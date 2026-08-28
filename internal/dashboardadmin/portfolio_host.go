package dashboardadmin

import (
	"context"
	"time"

	"ad-event-processor/internal/campaign"
	"ad-event-processor/internal/database"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

type PortfolioHost interface {
	Pool() *pgxpool.Pool
	ClickHouseQuery() *database.ClickHouseQuery
	ClickHouseIngestionLag(ctx context.Context) (time.Duration, error)
	ReportCHTimeout() time.Duration
	ErrValidation(msg string) error
	ListCampaigns(ctx context.Context, customerID uuid.UUID, status string, limit, offset int32) ([]campaign.CampaignDTO, int64, error)
	BatchCampaignMarginBreach(ctx context.Context, campaignIDs []uuid.UUID) (map[uuid.UUID]bool, error)
}

type Portfolio struct {
	host PortfolioHost
}

func NewPortfolio(host PortfolioHost) *Portfolio {
	return &Portfolio{host: host}
}
