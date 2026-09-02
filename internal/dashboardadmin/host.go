package dashboardadmin

import (
	"context"
	"time"

	"ad-event-processor/internal/database"
	"ad-event-processor/internal/ledger"
	"ad-event-processor/internal/reports"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

type BillingStatement struct {
	Lines               []ledger.InvoiceLineDTO
	TaxMicro            int64
	ClosingBalanceMicro int64
	InvoiceTotalMicro   int64
}

type BillingInvariant struct {
	OK        bool
	DiffMicro int64
}

type FraudMLSnapshot struct {
	VersionID        string
	ArtifactHash     string
	Precision        float64
	Recall           float64
	DriftDetected    bool
	DriftSummary     string
	EvalGeneratedAt  string
	EvalStatus       string
	EvalStale        bool
	LabelMethod      string
	ShardsConsistent *bool
}

type RoleHost interface {
	ErrValidation(msg string) error
	Pool() *pgxpool.Pool
	ClickHouseQuery() *database.ClickHouseQuery
	ReportCHTimeout() time.Duration
	GetBuyerPortfolio(ctx context.Context, customerID uuid.UUID) (BuyerPortfolioDTO, error)
	BuildStatement(ctx context.Context, customerID uuid.UUID, from, to time.Time) (BillingStatement, error)
	GetInvariant(ctx context.Context, customerID *uuid.UUID) (BillingInvariant, error)
	SumDisputeExposure(ctx context.Context, customerID uuid.UUID) int64
	FraudMLSnapshot(ctx context.Context) (FraudMLSnapshot, error)
	ListMLManualLabels(ctx context.Context, customerID uuid.UUID, limit int) ([]MLManualLabelDTO, error)
	FetchEdgeMetrics(ctx context.Context) (EdgeMetricsPanelDTO, error)
}

type RoleReportHost interface {
	ListCustomerCampaignIDs(ctx context.Context, customerID uuid.UUID) ([]uuid.UUID, error)
	QueryWorstIVTSources(ctx context.Context, campaignIDs []uuid.UUID, from, to time.Time, limit int) ([]reports.SourceRowDTO, error)
	QueryWorstIVTCountries(ctx context.Context, campaignIDs []uuid.UUID, from, to time.Time, limit int) ([]reports.FraudGeoHintDTO, error)
	QueryCustomerDashboardSeries(ctx context.Context, customerID uuid.UUID, campaignIDs []uuid.UUID, from, to time.Time, granularity reports.ChartGranularity) ([]reports.DashboardSeriesPointDTO, error)
}

type CampaignHost interface {
	ErrValidation(msg string) error
	Pool() *pgxpool.Pool
	ClickHouseQuery() *database.ClickHouseQuery
	ReportCHTimeout() time.Duration
	ClickHouseLag(ctx context.Context) time.Duration
	MapCampaignNotFound(err error) error
}

type PublisherHost interface {
	Pool() *pgxpool.Pool
	ClickHouseQuery() *database.ClickHouseQuery
	ReportCHTimeout() time.Duration
	BuildSellersJSON(ctx context.Context) ([]byte, error)
	BuildAdsTxt(ctx context.Context) (string, error)
}
