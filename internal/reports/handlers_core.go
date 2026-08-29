package reports

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"ad-event-processor/internal/controlplane/authz"
	"ad-event-processor/internal/database"
	"ad-event-processor/pkg/httpresponse"
	"ad-event-processor/pkg/money"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

type PlacementReportRowDTO struct {
	PlacementID  string               `json:"placement_id"`
	CampaignID   string               `json:"campaign_id"`
	Impressions  int64                `json:"impressions"`
	Clicks       int64                `json:"clicks"`
	Conversions  int64                `json:"conversions"`
	SpendMicro   int64                `json:"spend_micro"`
	RevenueMicro int64                `json:"revenue_micro"`
	ProfitMicro  int64                `json:"profit_micro"`
	ROIPct       float64              `json:"roi_pct"`
	CPAMicro     int64                `json:"cpa_micro"`
	CTR          float64              `json:"ctr,omitempty"`
	IVTRate      float64              `json:"ivt_rate,omitempty"`
	Compare      *ReportCompareDeltas `json:"compare,omitempty"`
}

type PlacementReportResponse struct {
	Rows       []PlacementReportRowDTO `json:"rows"`
	Freshness  DataFreshnessDTO        `json:"freshness"`
	NextCursor string                  `json:"next_cursor,omitempty"`
}

type KeywordReportRowDTO struct {
	Keyword      string               `json:"keyword"`
	CampaignID   string               `json:"campaign_id"`
	Impressions  int64                `json:"impressions"`
	Clicks       int64                `json:"clicks"`
	Conversions  int64                `json:"conversions"`
	SpendMicro   int64                `json:"spend_micro"`
	RevenueMicro int64                `json:"revenue_micro"`
	ProfitMicro  int64                `json:"profit_micro"`
	ROIPct       float64              `json:"roi_pct"`
	CPAMicro     int64                `json:"cpa_micro,omitempty"`
	CTR          float64              `json:"ctr,omitempty"`
	IVTRate      float64              `json:"ivt_rate,omitempty"`
	Compare      *ReportCompareDeltas `json:"compare,omitempty"`
}

type KeywordReportResponse struct {
	Rows       []KeywordReportRowDTO `json:"rows"`
	Freshness  DataFreshnessDTO      `json:"freshness"`
	NextCursor string                `json:"next_cursor,omitempty"`
}

const (
	MetricSpendMicro     = "spend_micro"
	MetricRevenueMicro   = "revenue_micro"
	MetricProfitMicro    = "profit_micro"
	MetricROIPct         = "roi_pct"
	MetricCPAMicro       = "cpa_micro"
	MetricCPCMicro       = "cpc_micro"
	MetricCPMMicro       = "cpm_micro"
	MetricCTR            = "ctr"
	MetricEPCMicro       = "epc_micro"
	MetricIVTRate        = "ivt_rate"
	MetricUtilizationPct = "utilization_pct"
	MetricAvailableMicro = "available_micro"
	MetricPacingDriftPct = "pacing_drift_pct"
)

var MetricFormulas = map[string]string{
	MetricSpendMicro:     "SUM(ledger debits) or CH cost",
	MetricRevenueMicro:   "SUM(postback payout)",
	MetricProfitMicro:    "revenue_micro - spend_micro",
	MetricROIPct:         "profit_micro / spend_micro * 100",
	MetricCPAMicro:       "spend_micro / conversions",
	MetricIVTRate:        "ivt / clicks",
	MetricUtilizationPct: "current_spend / budget_limit",
	MetricAvailableMicro: "balance + overdraft - reserved",
	MetricPacingDriftPct: "(actual - planned) / planned",
}

type CampaignStatsReader interface {
	GetCampaignStats(ctx context.Context, campaignID uuid.UUID, from, to time.Time, granularity string) (CampaignStatsDTO, error)
}

type CampaignForecaster interface {
	ForecastCampaign(ctx context.Context, in CampaignForecastInput) (CampaignForecastDTO, error)
}

const (
	maxStatsRange    = 90 * 24 * time.Hour
	maxStatsRangeDay = 365 * 24 * time.Hour
)

func (h *ReportsHTTPHandlers) requestHasShardsRead(r *http.Request) bool {
	if snap, ok := authz.SnapshotFromContext(r.Context()); ok {
		return snap.Has(permShardsRead)
	}
	if h != nil && h.RequestHasShardsRead != nil {
		return h.RequestHasShardsRead(r)
	}
	return false
}

func parseStatsQuery(r *http.Request, allowDayGranularity bool) (from, to time.Time, granularity string, err error) {
	granularity = r.URL.Query().Get("granularity")
	if granularity == "" {
		granularity = "hour"
	}
	switch granularity {
	case "hour":
	case "day":
		if !allowDayGranularity {
			return time.Time{}, time.Time{}, "", errInvalidQuery("granularity must be hour")
		}
	default:
		return time.Time{}, time.Time{}, "", errInvalidQuery("granularity must be hour or day")
	}

	now := time.Now().UTC()
	if granularity == "hour" {
		now = now.Truncate(time.Hour)
	} else {
		now = now.Truncate(24 * time.Hour)
	}
	to = now
	from = now.Add(-7 * 24 * time.Hour)

	if toStr := r.URL.Query().Get("to"); toStr != "" {
		to, err = time.Parse(time.RFC3339, toStr)
		if err != nil {
			return time.Time{}, time.Time{}, "", errInvalidQuery("invalid to timestamp")
		}
		to = to.UTC()
	}
	if fromStr := r.URL.Query().Get("from"); fromStr != "" {
		from, err = time.Parse(time.RFC3339, fromStr)
		if err != nil {
			return time.Time{}, time.Time{}, "", errInvalidQuery("invalid from timestamp")
		}
		from = from.UTC()
	}

	if !to.After(from) {
		return time.Time{}, time.Time{}, "", errInvalidQuery("to must be after from")
	}
	maxRange := maxStatsRange
	if granularity == "day" {
		maxRange = maxStatsRangeDay
	}
	if to.Sub(from) > maxRange {
		return time.Time{}, time.Time{}, "", errInvalidQuery(fmt.Sprintf("time range exceeds %d days", int(maxRange/(24*time.Hour))))
	}
	return from, to, granularity, nil
}

type reportMetricsCHRow struct {
	Dimension    string
	CampaignID   string
	Impressions  int64
	Clicks       int64
	Conversions  int64
	SpendMicro   int64
	RevenueMicro int64
}

type reportMetricsComputed struct {
	Impressions  int64
	Clicks       int64
	Conversions  int64
	SpendMicro   int64
	RevenueMicro int64
	ProfitMicro  int64
	ROIPct       float64
	CPAMicro     int64
	CTR          float64
	IVTRate      float64
}

func computeReportMetrics(row reportMetricsCHRow, ivtRate float64) reportMetricsComputed {
	profit := row.RevenueMicro - row.SpendMicro
	return reportMetricsComputed{
		Impressions:  row.Impressions,
		Clicks:       row.Clicks,
		Conversions:  row.Conversions,
		SpendMicro:   row.SpendMicro,
		RevenueMicro: row.RevenueMicro,
		ProfitMicro:  profit,
		ROIPct:       calcROIPct(profit, row.SpendMicro),
		CPAMicro:     calcCPAMicro(row.SpendMicro, row.Conversions),
		CTR:          calcCTR(row.Clicks, row.Impressions),
		IVTRate:      ivtRate,
	}
}

func ToPlacementReportRowDTO(row reportMetricsCHRow, ivtRate float64) PlacementReportRowDTO {
	m := computeReportMetrics(row, ivtRate)
	return PlacementReportRowDTO{
		PlacementID:  row.Dimension,
		CampaignID:   row.CampaignID,
		Impressions:  m.Impressions,
		Clicks:       m.Clicks,
		Conversions:  m.Conversions,
		SpendMicro:   m.SpendMicro,
		RevenueMicro: m.RevenueMicro,
		ProfitMicro:  m.ProfitMicro,
		ROIPct:       m.ROIPct,
		CPAMicro:     m.CPAMicro,
		CTR:          m.CTR,
		IVTRate:      m.IVTRate,
	}
}

func toKeywordReportRowDTO(row reportMetricsCHRow, ivtRate float64) KeywordReportRowDTO {
	m := computeReportMetrics(row, ivtRate)
	return KeywordReportRowDTO{
		Keyword:      row.Dimension,
		CampaignID:   row.CampaignID,
		Impressions:  m.Impressions,
		Clicks:       m.Clicks,
		Conversions:  m.Conversions,
		SpendMicro:   m.SpendMicro,
		RevenueMicro: m.RevenueMicro,
		ProfitMicro:  m.ProfitMicro,
		ROIPct:       m.ROIPct,
		CPAMicro:     m.CPAMicro,
		CTR:          m.CTR,
		IVTRate:      m.IVTRate,
	}
}

type ReportsHTTPHandlers struct {
	CampaignStats               CampaignStatsReader
	CampaignForecaster          CampaignForecaster
	Pool                        *pgxpool.Pool
	ClickHouseQuery             *database.ClickHouseQuery
	BuyerPortfolio              BuyerPortfolioReader
	EdgeMetricsReader           func(context.Context) (EdgeMetricsPanelDTO, error)
	ApplyRateLimit              func(http.HandlerFunc) http.HandlerFunc
	RequirePermission           func(string, http.HandlerFunc) http.HandlerFunc
	RequireAnyPermission        func([]string, http.HandlerFunc) http.HandlerFunc
	AuthorizeCampaignAccess     func(*http.Request, uuid.UUID) error
	AuthorizeCustomerAccess     func(*http.Request, string) error
	ResolveForecastCustomerID   func(*http.Request, *uuid.UUID) (*uuid.UUID, error)
	WriteServiceError           func(http.ResponseWriter, error)
	RequestHasShardsRead        func(*http.Request) bool
	RequireLicenseFeature       func(http.ResponseWriter, string) bool
	DenyScopedAPIKeyReport      func(http.ResponseWriter, *http.Request, string) bool
	FraudEvidencePackHMACSecret []byte
}

const permShardsRead = "shards:read"

const permCampaignsReadMasked = "campaigns:read:masked"

func (h *ReportsHTTPHandlers) writeServiceError(w http.ResponseWriter, err error) {
	var q invalidQueryError
	if errors.As(err, &q) {
		httpresponse.Error(w, http.StatusBadRequest, "BAD_REQUEST", string(q))
		return
	}
	if errors.Is(err, ErrForbidden) {
		httpresponse.Error(w, http.StatusForbidden, "FORBIDDEN", "forbidden")
		return
	}
	if h.WriteServiceError != nil {
		h.WriteServiceError(w, err)
		return
	}
	httpresponse.Error(w, http.StatusInternalServerError, "INTERNAL_ERROR", "internal error")
}

func (h *ReportsHTTPHandlers) reportFreshness(ctx context.Context) DataFreshnessDTO {
	if h == nil {
		return DataFreshnessFromClickHouse(ctx, nil)
	}
	return DataFreshnessFromClickHouse(ctx, h.ClickHouseQuery)
}

func (h *ReportsHTTPHandlers) resolveForecastCustomerID(r *http.Request, bodyCustomerID *uuid.UUID) (*uuid.UUID, error) {
	if h.ResolveForecastCustomerID == nil {
		return bodyCustomerID, nil
	}
	return h.ResolveForecastCustomerID(r, bodyCustomerID)
}

func forecastParseBudgetMicro(micro *int64, legacy float64, hasLegacy bool) (int64, error) {
	if micro != nil {
		if *micro <= 0 {
			return 0, errInvalidQuery("budget must be positive")
		}
		return *micro, nil
	}
	if hasLegacy {
		v, err := money.LegacyFloatToMicro(legacy)
		if err != nil || v <= 0 {
			return 0, errInvalidQuery("budget must be positive")
		}
		return v, nil
	}
	return 0, errInvalidQuery("budget is required")
}

type ForecastUnavailableResponse struct {
	Error      ForecastErrorDetail `json:"error"`
	RetryAfter int                 `json:"retry_after"`
}

type ForecastErrorDetail struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

func WriteForecastError(w http.ResponseWriter, err error) {
	if errors.Is(err, ErrForecastClickHouseTimeout) || errors.Is(err, ErrForecastUnavailable) {
		w.Header().Set("Retry-After", strconv.Itoa(ForecastRetryAfterSec()))
		httpresponse.JSON(w, http.StatusServiceUnavailable, ForecastUnavailableResponse{
			Error: ForecastErrorDetail{
				Code:    "FORECAST_UNAVAILABLE",
				Message: err.Error(),
			},
			RetryAfter: ForecastRetryAfterSec(),
		})
		return
	}
	if errors.Is(err, ErrClickHouseNotConfigured) {
		httpresponse.Error(w, http.StatusServiceUnavailable, "CLICKHOUSE_UNAVAILABLE", "clickhouse not configured")
		return
	}
	httpresponse.Error(w, http.StatusInternalServerError, "INTERNAL_ERROR", "internal error")
}

func (h *ReportsHTTPHandlers) WrapReport(reportKey string, next http.HandlerFunc) http.HandlerFunc {
	return h.wrapReport(reportKey, next)
}

func (h *ReportsHTTPHandlers) ResolveReportCustomerID(w http.ResponseWriter, r *http.Request) (uuid.UUID, bool) {
	return h.resolveReportCustomerID(w, r)
}

func (h *ReportsHTTPHandlers) ReportFreshness(ctx context.Context) DataFreshnessDTO {
	return h.reportFreshness(ctx)
}

func (h *ReportsHTTPHandlers) AuthorizeReportCampaign(w http.ResponseWriter, r *http.Request, campaignID uuid.UUID) bool {
	return h.authorizeReportCampaign(w, r, campaignID)
}

func WriteReportsHandlerError(h *ReportsHTTPHandlers, w http.ResponseWriter, err error) {
	h.writeServiceError(w, err)
}
