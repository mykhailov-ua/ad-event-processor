package controlplane

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"ad-event-processor/internal/controlplane/authz"
	"ad-event-processor/internal/database"
	"ad-event-processor/pkg/coldpath"
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

func requestHasShardsRead(r *http.Request) bool {
	if snap, ok := authz.SnapshotFromContext(r.Context()); ok {
		return snap.Has(PermShardsRead)
	}
	user, ok := GetUser(r.Context())
	if !ok {
		return false
	}
	return HasPermission(user.Role, PermShardsRead)
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

func toPlacementReportRowDTO(row reportMetricsCHRow, ivtRate float64) PlacementReportRowDTO {
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
	CampaignStats             CampaignStatsReader
	CampaignForecaster        CampaignForecaster
	ReportJobs                *ReportJobRunner
	Pool                      *pgxpool.Pool
	CHQuery                   *database.CHQuery
	BuyerPortfolio            BuyerPortfolioReader
	EdgeMetricsReader         func(context.Context) (EdgeMetricsPanelDTO, error)
	ApplyRateLimit            func(http.HandlerFunc) http.HandlerFunc
	RequirePermission         func(string, http.HandlerFunc) http.HandlerFunc
	RequireAnyPermission      func([]string, http.HandlerFunc) http.HandlerFunc
	AuthorizeCampaignAccess   func(*http.Request, uuid.UUID) error
	AuthorizeCustomerAccess   func(*http.Request, string) error
	ResolveForecastCustomerID func(*http.Request, *uuid.UUID) (*uuid.UUID, error)
	WriteServiceError         func(http.ResponseWriter, error)
}

func (reports *ReportsHTTPHandlers) Register(mux *http.ServeMux) {
	if reports == nil {
		return
	}
	if reports.ApplyRateLimit == nil {
		reports.ApplyRateLimit = func(next http.HandlerFunc) http.HandlerFunc { return next }
	}
	if reports.RequirePermission == nil {
		reports.RequirePermission = func(_ string, next http.HandlerFunc) http.HandlerFunc { return next }
	}

	reports.registerCampaignStats(mux)
	reports.registerCampaignForecast(mux)

	limit := reports.ApplyRateLimit
	permAny := reports.RequireAnyPermission
	if permAny == nil {
		permAny = func(_ []string, next http.HandlerFunc) http.HandlerFunc { return next }
	}
	readCampaigns := []string{"campaigns:read", "campaigns:read:masked"}
	mux.HandleFunc("GET /api/v1/reports/placements", limit(permAny(readCampaigns, reports.wrapReport("placements", reports.getPlacementsReport))))
	mux.HandleFunc("GET /api/v1/reports/keywords", limit(permAny(readCampaigns, reports.wrapReport("keywords", reports.getKeywordsReport))))
	reports.registerIVTBySource(mux)
	reports.registerTrafficSources(mux)
	reports.registerGeoROI(mux)
	reports.registerExtendedReports(mux)
	reports.registerDataQualityReport(mux)
	reports.registerFilterRejectsReport(mux)
	reports.registerFraudBreakdownReport(mux)
	reports.registerGhostImpressionFunnelReport(mux)
	reports.registerRtbReports(mux)
	reports.registerPostbackReconReport(mux)
	reports.registerPacingDriftReport(mux)
	reports.registerCostCoverageReport(mux)
	reports.registerMLReports(mux)
	reports.registerEdgeParityReport(mux)
	reports.registerReportSchedules(mux)
	reports.registerReportJobs(mux)
}

func (reports *ReportsHTTPHandlers) writeServiceError(w http.ResponseWriter, err error) {
	var q invalidQueryError
	if errors.As(err, &q) {
		httpresponse.Error(w, http.StatusBadRequest, "BAD_REQUEST", string(q))
		return
	}
	if errors.Is(err, ErrForbidden) {
		httpresponse.Error(w, http.StatusForbidden, "FORBIDDEN", "forbidden")
		return
	}
	if reports.WriteServiceError != nil {
		reports.WriteServiceError(w, err)
		return
	}
	httpresponse.Error(w, http.StatusInternalServerError, "INTERNAL_ERROR", "internal error")
}

func (reports *ReportsHTTPHandlers) reportFreshness(ctx context.Context) DataFreshnessDTO {
	dto := DataFreshnessDTO{
		AsOf:        time.Now().UTC().Format(time.RFC3339),
		Consistency: "eventual",
	}
	if reports == nil || reports.CHQuery == nil {
		dto.Stale = true
		return dto
	}
	lag, err := reports.CHQuery.IngestionLag(ctx)
	if err != nil {
		dto.Stale = true
		return dto
	}
	dto.Stale, dto.CHLagSeconds = database.Freshness(lag, 5*time.Minute)
	return dto
}

func (reports *ReportsHTTPHandlers) getPlacementsReport(w http.ResponseWriter, r *http.Request) {
	customerID, ok := reports.resolveReportCustomerID(w, r)
	if !ok {
		return
	}

	if reports.CHQuery == nil {
		httpresponse.Error(w, http.StatusServiceUnavailable, "CLICKHOUSE_UNAVAILABLE", "clickhouse not configured")
		return
	}

	from, to, err := parseReportRange(r)
	if err != nil {
		reports.writeServiceError(w, err)
		return
	}

	limit := int32(10)
	page, err := coldpath.ParseCursorPagination(r, int(limit), 1000)
	if err != nil {
		httpresponse.Error(w, http.StatusBadRequest, "BAD_REQUEST", "invalid cursor")
		return
	}
	offset := page.Offset
	limit = int32(page.Limit)

	var campaignIDs []uuid.UUID
	if campaignIDStr := r.URL.Query().Get("campaign_id"); campaignIDStr != "" {
		campaignID, err := uuid.Parse(campaignIDStr)
		if err != nil {
			httpresponse.Error(w, http.StatusBadRequest, "BAD_REQUEST", "invalid campaign_id")
			return
		}
		if !reports.authorizeReportCampaign(w, r, campaignID) {
			return
		}
		campaignIDs = []uuid.UUID{campaignID}
	} else if reports.Pool != nil {
		campaignIDs, err = listCustomerCampaignIDs(r.Context(), reports.Pool, customerID)
		if err != nil {
			reports.writeServiceError(w, err)
			return
		}
	}
	if len(campaignIDs) == 0 {
		httpresponse.JSON(w, http.StatusOK, PlacementReportResponse{
			Rows:       []PlacementReportRowDTO{},
			Freshness:  reports.reportFreshness(r.Context()),
			NextCursor: "",
		})
		return
	}

	chCtx, cancel := context.WithTimeout(r.Context(), reportCHQueryTimeout)
	defer cancel()

	chRows, total, err := queryPlacementReportRows(chCtx, reports.CHQuery, campaignIDs, from, to, int(limit), offset)
	if err != nil {
		reports.writeServiceError(w, err)
		return
	}

	ivtRates, err := queryPlacementIVTRates(chCtx, reports.CHQuery, campaignIDs, from, to)
	if err != nil {
		reports.writeServiceError(w, err)
		return
	}

	rows := coldpath.MapSlice(chRows, func(row reportMetricsCHRow) PlacementReportRowDTO {
		ivt := ivtRates[reportMetricsKey(row.Dimension, row.CampaignID)]
		return toPlacementReportRowDTO(row, ivt)
	})

	if parseComparePrevious(r) {
		prevFrom, prevTo := previousReportRange(from, to)
		prevRows, _, perr := queryPlacementReportRows(chCtx, reports.CHQuery, campaignIDs, prevFrom, prevTo, int(limit), offset)
		if perr != nil {
			reports.writeServiceError(w, perr)
			return
		}
		attachPlacementCompareDeltas(rows, prevRows)
	}

	var nextCursor string
	if int64(offset)+int64(len(rows)) < total {
		nextCursor = coldpath.EncodeCursor(offset + int(limit))
	}

	resp := PlacementReportResponse{
		Rows:       rows,
		Freshness:  reports.reportFreshness(r.Context()),
		NextCursor: nextCursor,
	}
	httpresponse.JSON(w, http.StatusOK, resp)
}

func (reports *ReportsHTTPHandlers) getKeywordsReport(w http.ResponseWriter, r *http.Request) {
	customerID, ok := reports.resolveReportCustomerID(w, r)
	if !ok {
		return
	}

	if reports.CHQuery == nil {
		httpresponse.Error(w, http.StatusServiceUnavailable, "CLICKHOUSE_UNAVAILABLE", "clickhouse not configured")
		return
	}

	from, to, err := parseReportRange(r)
	if err != nil {
		reports.writeServiceError(w, err)
		return
	}

	limit := int32(10)
	page, err := coldpath.ParseCursorPagination(r, int(limit), 1000)
	if err != nil {
		httpresponse.Error(w, http.StatusBadRequest, "BAD_REQUEST", "invalid cursor")
		return
	}
	offset := page.Offset
	limit = int32(page.Limit)

	var campaignIDs []uuid.UUID
	if campaignIDStr := r.URL.Query().Get("campaign_id"); campaignIDStr != "" {
		campaignID, parseErr := uuid.Parse(campaignIDStr)
		if parseErr != nil {
			httpresponse.Error(w, http.StatusBadRequest, "BAD_REQUEST", "invalid campaign_id")
			return
		}
		if !reports.authorizeReportCampaign(w, r, campaignID) {
			return
		}
		campaignIDs = []uuid.UUID{campaignID}
	} else if reports.Pool != nil {
		campaignIDs, err = listCustomerCampaignIDs(r.Context(), reports.Pool, customerID)
		if err != nil {
			reports.writeServiceError(w, err)
			return
		}
	}
	if len(campaignIDs) == 0 {
		httpresponse.JSON(w, http.StatusOK, KeywordReportResponse{
			Rows:       []KeywordReportRowDTO{},
			Freshness:  reports.reportFreshness(r.Context()),
			NextCursor: "",
		})
		return
	}

	chCtx, cancel := context.WithTimeout(r.Context(), reportCHQueryTimeout)
	defer cancel()

	chRows, total, err := queryKeywordReportRows(chCtx, reports.CHQuery, campaignIDs, from, to, int(limit), offset)
	if err != nil {
		reports.writeServiceError(w, err)
		return
	}

	ivtRates, err := queryKeywordIVTRates(chCtx, reports.CHQuery, campaignIDs, from, to)
	if err != nil {
		reports.writeServiceError(w, err)
		return
	}

	rows := coldpath.MapSlice(chRows, func(row reportMetricsCHRow) KeywordReportRowDTO {
		ivt := ivtRates[reportMetricsKey(row.Dimension, row.CampaignID)]
		return toKeywordReportRowDTO(row, ivt)
	})

	if parseComparePrevious(r) {
		prevFrom, prevTo := previousReportRange(from, to)
		prevRows, _, perr := queryKeywordReportRows(chCtx, reports.CHQuery, campaignIDs, prevFrom, prevTo, int(limit), offset)
		if perr != nil {
			reports.writeServiceError(w, perr)
			return
		}
		attachKeywordCompareDeltas(rows, prevRows)
	}

	var nextCursor string
	if int64(offset)+int64(len(rows)) < total {
		nextCursor = coldpath.EncodeCursor(offset + int(limit))
	}

	resp := KeywordReportResponse{
		Rows:       rows,
		Freshness:  reports.reportFreshness(r.Context()),
		NextCursor: nextCursor,
	}
	httpresponse.JSON(w, http.StatusOK, resp)
}

const (
	forecastHandlerTimeout = 2 * time.Second
)

func (reports *ReportsHTTPHandlers) registerCampaignStats(mux *http.ServeMux) {
	if reports.CampaignStats == nil {
		return
	}
	limit := reports.ApplyRateLimit
	permAny := reports.RequireAnyPermission
	if permAny == nil {
		perm := reports.RequirePermission
		permAny = func(perms []string, next http.HandlerFunc) http.HandlerFunc {
			if len(perms) == 0 {
				return next
			}
			return perm(perms[0], next)
		}
	}
	mux.HandleFunc("GET /api/v1/campaigns/{id}/stats", limit(permAny([]string{"campaigns:read", "campaigns:read:masked"}, reports.wrapReport("campaign-stats", reports.getCampaignStats))))
}

func (reports *ReportsHTTPHandlers) getCampaignStats(w http.ResponseWriter, r *http.Request) {
	idStr := r.PathValue("id")
	campaignID, err := uuid.Parse(idStr)
	if err != nil {
		httpresponse.Error(w, http.StatusBadRequest, "BAD_REQUEST", "invalid campaign id")
		return
	}

	if reports.AuthorizeCampaignAccess != nil {
		if err := reports.AuthorizeCampaignAccess(r, campaignID); err != nil {
			reports.writeServiceError(w, err)
			return
		}
	}

	from, to, granularity, err := parseStatsQuery(r, requestHasShardsRead(r))
	if err != nil {
		reports.writeServiceError(w, err)
		return
	}

	report, err := reports.CampaignStats.GetCampaignStats(r.Context(), campaignID, from, to, granularity)
	if err != nil {
		reports.writeServiceError(w, err)
		return
	}

	httpresponse.JSON(w, http.StatusOK, report)
}

func (reports *ReportsHTTPHandlers) registerCampaignForecast(mux *http.ServeMux) {
	if reports.CampaignForecaster == nil {
		return
	}
	limit := reports.ApplyRateLimit
	perm := reports.RequirePermission
	mux.HandleFunc("POST /api/v1/forecast/campaign", limit(perm("campaigns:read", reports.forecastCampaign)))
}

func (reports *ReportsHTTPHandlers) forecastCampaign(w http.ResponseWriter, r *http.Request) {
	body, err := coldpath.ReadLimitedBody(w, r, coldpath.DefaultMaxBody)
	if err != nil {
		httpresponse.Error(w, http.StatusBadRequest, "BAD_REQUEST", "failed to read request body")
		return
	}

	req, err := coldpath.DecodeBody[struct {
		CustomerID       *uuid.UUID `json:"customer_id,omitempty"`
		BudgetLimitMicro *int64     `json:"budget_limit_micro"`
		BudgetLimit      *float64   `json:"budget_limit"`
		TargetCountries  []string   `json:"target_countries"`
		DaypartHours     []int16    `json:"daypart_hours"`
		StartAt          *time.Time `json:"start_at"`
		EndAt            *time.Time `json:"end_at"`
		PacingMode       string     `json:"pacing_mode"`
		Timezone         string     `json:"timezone"`
	}](body)
	if err != nil {
		httpresponse.Error(w, http.StatusBadRequest, "BAD_REQUEST", "invalid request body")
		return
	}

	customerID, err := reports.resolveForecastCustomerID(r, req.CustomerID)
	if err != nil {
		reports.writeServiceError(w, err)
		return
	}

	budgetLegacy := 0.0
	hasLegacy := req.BudgetLimit != nil
	if hasLegacy {
		budgetLegacy = *req.BudgetLimit
	}
	budgetMicro, err := forecastParseBudgetMicro(req.BudgetLimitMicro, budgetLegacy, hasLegacy)
	if err != nil {
		reports.writeServiceError(w, err)
		return
	}

	if req.StartAt == nil || req.EndAt == nil {
		httpresponse.Error(w, http.StatusBadRequest, "BAD_REQUEST", "start_at and end_at are required")
		return
	}

	tz := req.Timezone
	if tz == "" {
		tz = "UTC"
	}

	ctx, cancel := context.WithTimeout(r.Context(), forecastHandlerTimeout)
	defer cancel()

	out, err := reports.CampaignForecaster.ForecastCampaign(ctx, CampaignForecastInput{
		CustomerID:       customerID,
		BudgetLimitMicro: budgetMicro,
		TargetCountries:  req.TargetCountries,
		DaypartHours:     req.DaypartHours,
		StartAt:          req.StartAt.UTC(),
		EndAt:            req.EndAt.UTC(),
		PacingMode:       req.PacingMode,
		Timezone:         tz,
	})
	if err != nil {
		WriteForecastError(w, err)
		return
	}

	httpresponse.JSON(w, http.StatusOK, out)
}

func (reports *ReportsHTTPHandlers) resolveForecastCustomerID(r *http.Request, bodyCustomerID *uuid.UUID) (*uuid.UUID, error) {
	if reports.ResolveForecastCustomerID == nil {
		return bodyCustomerID, nil
	}
	return reports.ResolveForecastCustomerID(r, bodyCustomerID)
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
