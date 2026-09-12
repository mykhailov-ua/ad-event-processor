package dashboardadmin

import (
	"context"
	"testing"
	"time"

	"ad-event-processor/internal/database"
	"ad-event-processor/internal/reports"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type stubAdOpsRoleHost struct {
	from, to time.Time
}

func (s *stubAdOpsRoleHost) ErrValidation(msg string) error { return InvalidQuery(msg) }
func (s *stubAdOpsRoleHost) Pool() *pgxpool.Pool            { return nil }
func (s *stubAdOpsRoleHost) ClickHouseQuery() *database.ClickHouseQuery {
	return nil
}
func (s *stubAdOpsRoleHost) ReportCHTimeout() time.Duration { return time.Second }
func (s *stubAdOpsRoleHost) GetBuyerPortfolio(context.Context, uuid.UUID) (BuyerPortfolioDTO, error) {
	return BuyerPortfolioDTO{}, nil
}

func (s *stubAdOpsRoleHost) GetBuyerPortfolioRange(_ context.Context, customerID uuid.UUID, _ *uuid.UUID, from, to time.Time, _ reports.ChartGranularity) (BuyerPortfolioDTO, error) {
	s.from = from
	s.to = to
	return BuyerPortfolioDTO{
		CustomerID: customerID.String(),
		Period: PeriodDTO{
			From: from.Format(time.RFC3339),
			To:   to.Format(time.RFC3339),
		},
		KPIs: &MetricsBlockDTO{},
	}, nil
}

func (s *stubAdOpsRoleHost) BuildStatement(context.Context, uuid.UUID, time.Time, time.Time) (BillingStatement, error) {
	return BillingStatement{}, nil
}

func (s *stubAdOpsRoleHost) GetInvariant(context.Context, *uuid.UUID) (BillingInvariant, error) {
	return BillingInvariant{}, nil
}
func (s *stubAdOpsRoleHost) SumDisputeExposure(context.Context, uuid.UUID) int64 { return 0 }
func (s *stubAdOpsRoleHost) FraudMLSnapshot(context.Context) (FraudMLSnapshot, error) {
	return FraudMLSnapshot{}, nil
}

func (s *stubAdOpsRoleHost) ListMLManualLabels(context.Context, uuid.UUID, int) ([]MLManualLabelDTO, error) {
	return nil, nil
}

func (s *stubAdOpsRoleHost) FetchEdgeMetrics(context.Context) (EdgeMetricsPanelDTO, error) {
	return EdgeMetricsPanelDTO{}, nil
}

func TestGetAdOpsDashboard_usesRequestedRange_holdout(t *testing.T) {
	t.Parallel()
	host := &stubAdOpsRoleHost{}
	st := NewRoleService(host, nil)
	customerID := uuid.New()
	from := time.Date(2026, 3, 1, 0, 0, 0, 0, time.UTC)
	to := time.Date(2026, 3, 7, 23, 59, 59, 0, time.UTC)
	resp, err := st.GetAdOpsDashboard(context.Background(), customerID, from, to)
	require.NoError(t, err)
	assert.Equal(t, from, host.from)
	assert.Equal(t, to, host.to)
	assert.Equal(t, from.Format(time.RFC3339), resp.Period.From)
	assert.Equal(t, to.Format(time.RFC3339), resp.Period.To)
}

func TestWorstSourcesFromCampaigns_qualityMonotone(t *testing.T) {
	t.Parallel()
	campaigns := []BuyerCampaignRowDTO{
		{ID: "a", Name: "Low drift", PacingDriftPct: 10, SpendMicro: 100},
		{ID: "b", Name: "High drift", PacingDriftPct: 80, OverspendRisk: true, SpendMicro: 200},
	}
	out := worstSourcesFromCampaigns(campaigns)
	if assert.Len(t, out, 2) {
		assert.Equal(t, "b", out[0].CampaignID)
		assert.InDelta(t, reports.CalcQualityFromDrift(80), out[0].QualityScore, 1e-9)
		assert.Equal(t, float64(0), out[0].IVTRate)
	}
}
