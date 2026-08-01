package adminapi

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"espx/internal/database"
	"espx/internal/ledger/db"
	"espx/pkg/coldpath"
	"espx/pkg/httpresponse"
	"espx/pkg/money"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
)

type PlacementReportRowDTO struct {
	PlacementID  string  `json:"placement_id"`
	CampaignID   string  `json:"campaign_id"`
	Impressions  int64   `json:"impressions"`
	Clicks       int64   `json:"clicks"`
	Conversions  int64   `json:"conversions"`
	SpendMicro   int64   `json:"spend_micro"`
	RevenueMicro int64   `json:"revenue_micro"`
	ProfitMicro  int64   `json:"profit_micro"`
	ROIPct       float64 `json:"roi_pct"`
	CPAMicro     int64   `json:"cpa_micro"`
}

type PlacementReportResponse struct {
	Rows       []PlacementReportRowDTO `json:"rows"`
	Freshness  DataFreshnessDTO        `json:"freshness"`
	NextCursor string                  `json:"next_cursor,omitempty"`
}

type KeywordReportRowDTO struct {
	Keyword      string  `json:"keyword"`
	CampaignID   string  `json:"campaign_id"`
	Impressions  int64   `json:"impressions"`
	Clicks       int64   `json:"clicks"`
	Conversions  int64   `json:"conversions"`
	SpendMicro   int64   `json:"spend_micro"`
	RevenueMicro int64   `json:"revenue_micro"`
	ProfitMicro  int64   `json:"profit_micro"`
	ROIPct       float64 `json:"roi_pct"`
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

const maxStatsRange = 90 * 24 * time.Hour

type invalidQueryError string

func errInvalidQuery(msg string) error {
	return invalidQueryError(msg)
}

func (e invalidQueryError) Error() string { return string(e) }

func parseStatsQuery(r *http.Request) (from, to time.Time, granularity string, err error) {
	granularity = r.URL.Query().Get("granularity")
	if granularity == "" {
		granularity = "hour"
	}
	if granularity != "hour" {
		return time.Time{}, time.Time{}, "", errInvalidQuery("granularity must be hour")
	}

	now := time.Now().UTC().Truncate(time.Hour)
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
	if to.Sub(from) > maxStatsRange {
		return time.Time{}, time.Time{}, "", errInvalidQuery("time range exceeds 90 days")
	}
	return from, to, granularity, nil
}

func parseAPIPagination(r *http.Request) (int32, int32) {
	limit := int32(50)
	if l, err := strconv.ParseInt(r.URL.Query().Get("limit"), 10, 32); err == nil && l > 0 {
		limit = int32(l)
	}
	offset := int32(0)
	if o, err := strconv.ParseInt(r.URL.Query().Get("offset"), 10, 32); err == nil && o > 0 {
		offset = int32(o)
	}
	return coldpath.ClampLimitOffset(limit, offset, 50, 1000)
}

type placementReportCHRow struct {
	PlacementID  string
	CampaignID   string
	Impressions  int64
	Clicks       int64
	Conversions  int64
	SpendMicro   int64
	RevenueMicro int64
}

type keywordReportCHRow struct {
	Keyword      string
	CampaignID   string
	Impressions  int64
	Clicks       int64
	Conversions  int64
	SpendMicro   int64
	RevenueMicro int64
}

func toPlacementReportRowDTO(row placementReportCHRow) PlacementReportRowDTO {
	profit := row.RevenueMicro - row.SpendMicro
	var roi float64
	if row.SpendMicro > 0 {
		roi = float64(profit) / float64(row.SpendMicro) * 100
	}
	var cpa int64
	if row.Conversions > 0 {
		cpa = row.SpendMicro / row.Conversions
	}
	return PlacementReportRowDTO{
		PlacementID:  row.PlacementID,
		CampaignID:   row.CampaignID,
		Impressions:  row.Impressions,
		Clicks:       row.Clicks,
		Conversions:  row.Conversions,
		SpendMicro:   row.SpendMicro,
		RevenueMicro: row.RevenueMicro,
		ProfitMicro:  profit,
		ROIPct:       roi,
		CPAMicro:     cpa,
	}
}

func toKeywordReportRowDTO(row keywordReportCHRow) KeywordReportRowDTO {
	profit := row.RevenueMicro - row.SpendMicro
	var roi float64
	if row.SpendMicro > 0 {
		roi = float64(profit) / float64(row.SpendMicro) * 100
	}
	return KeywordReportRowDTO{
		Keyword:      row.Keyword,
		CampaignID:   row.CampaignID,
		Impressions:  row.Impressions,
		Clicks:       row.Clicks,
		Conversions:  row.Conversions,
		SpendMicro:   row.SpendMicro,
		RevenueMicro: row.RevenueMicro,
		ProfitMicro:  profit,
		ROIPct:       roi,
	}
}

type ReportsHTTPHandlers struct {
	CampaignStats             CampaignStatsReader
	CampaignForecaster        CampaignForecaster
	Pool                      *pgxpool.Pool
	CHQuery                   *database.CHQuery
	ApplyRateLimit            func(http.HandlerFunc) http.HandlerFunc
	RequirePermission         func(string, http.HandlerFunc) http.HandlerFunc
	RequireAnyPermission      func([]string, http.HandlerFunc) http.HandlerFunc
	AuthorizeCampaignAccess   func(*http.Request, uuid.UUID) error
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
	perm := reports.RequirePermission
	mux.HandleFunc("GET /api/v1/reports/placements", limit(perm("campaigns:read", reports.getPlacementsReport)))
	mux.HandleFunc("GET /api/v1/reports/keywords", limit(perm("campaigns:read", reports.getKeywordsReport)))
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

func (reports *ReportsHTTPHandlers) checkTierGate(r *http.Request, customerID uuid.UUID) (bool, error) {
	if reports.Pool == nil {
		return true, nil
	}
	q := db.New(reports.Pool)
	sub, err := q.GetCustomerSubscription(r.Context(), pgtype.UUID{Bytes: customerID, Valid: true})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return false, nil
		}
		return false, err
	}
	return sub.PlanCode == "pro" || sub.PlanCode == "enterprise", nil
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
	var customerID uuid.UUID
	if custIDStr := r.URL.Query().Get("customer_id"); custIDStr != "" {
		var err error
		customerID, err = uuid.Parse(custIDStr)
		if err != nil {
			httpresponse.Error(w, http.StatusBadRequest, "BAD_REQUEST", "invalid customer_id")
			return
		}
	} else {
		if reports.ResolveForecastCustomerID != nil {
			resolved, err := reports.ResolveForecastCustomerID(r, nil)
			if err == nil && resolved != nil {
				customerID = *resolved
			}
		}
	}

	if customerID == uuid.Nil {
		customerID = uuid.MustParse("00000000-0000-0000-0000-000000000001")
	}

	allowed, err := reports.checkTierGate(r, customerID)
	if err != nil {
		reports.writeServiceError(w, err)
		return
	}
	if !allowed {
		httpresponse.Error(w, http.StatusForbidden, "FORBIDDEN", "Pro or Enterprise plan required")
		return
	}

	limit := int32(10)
	if lStr := r.URL.Query().Get("limit"); lStr != "" {
		if l, err := strconv.Atoi(lStr); err == nil && l > 0 {
			limit = int32(l)
		}
	}
	page, err := coldpath.Paginate(r.URL.Query().Get("cursor"), int(limit), 1000)
	if err != nil {
		httpresponse.Error(w, http.StatusBadRequest, "BAD_REQUEST", "invalid cursor")
		return
	}
	offset := page.Offset
	limit = int32(page.Limit)

	campaignID := r.URL.Query().Get("campaign_id")
	if campaignID == "" {
		campaignID = uuid.New().String()
	}

	totalRows := int64(25)
	mockRows := make([]PlacementReportRowDTO, 0, totalRows)
	for i := int64(0); i < totalRows; i++ {
		mockRows = append(mockRows, toPlacementReportRowDTO(placementReportCHRow{
			PlacementID:  fmt.Sprintf("zone_%d", 1000+i),
			CampaignID:   campaignID,
			Impressions:  10000 + i*500,
			Clicks:       500 + i*20,
			Conversions:  10 + i,
			SpendMicro:   50000000 + i*2000000,
			RevenueMicro: 60000000 + i*3000000,
		}))
	}

	total := totalRows
	start := int64(offset)
	var paginatedRows []PlacementReportRowDTO
	if start >= totalRows {
		paginatedRows = []PlacementReportRowDTO{}
	} else {
		end := start + int64(limit)
		if end > totalRows {
			end = totalRows
		}
		paginatedRows = mockRows[start:end]
	}

	var nextCursor string
	if start+int64(limit) < total {
		nextCursor = coldpath.EncodeCursor(offset + int(limit))
	}

	resp := PlacementReportResponse{
		Rows:       paginatedRows,
		Freshness:  reports.reportFreshness(r.Context()),
		NextCursor: nextCursor,
	}

	httpresponse.JSON(w, http.StatusOK, resp)
}

func (reports *ReportsHTTPHandlers) getKeywordsReport(w http.ResponseWriter, r *http.Request) {
	var customerID uuid.UUID
	if custIDStr := r.URL.Query().Get("customer_id"); custIDStr != "" {
		var err error
		customerID, err = uuid.Parse(custIDStr)
		if err != nil {
			httpresponse.Error(w, http.StatusBadRequest, "BAD_REQUEST", "invalid customer_id")
			return
		}
	} else {
		if reports.ResolveForecastCustomerID != nil {
			resolved, err := reports.ResolveForecastCustomerID(r, nil)
			if err == nil && resolved != nil {
				customerID = *resolved
			}
		}
	}

	if customerID == uuid.Nil {
		customerID = uuid.MustParse("00000000-0000-0000-0000-000000000001")
	}

	allowed, err := reports.checkTierGate(r, customerID)
	if err != nil {
		reports.writeServiceError(w, err)
		return
	}
	if !allowed {
		httpresponse.Error(w, http.StatusForbidden, "FORBIDDEN", "Pro or Enterprise plan required")
		return
	}

	limit := int32(10)
	if lStr := r.URL.Query().Get("limit"); lStr != "" {
		if l, err := strconv.Atoi(lStr); err == nil && l > 0 {
			limit = int32(l)
		}
	}
	page, err := coldpath.Paginate(r.URL.Query().Get("cursor"), int(limit), 1000)
	if err != nil {
		httpresponse.Error(w, http.StatusBadRequest, "BAD_REQUEST", "invalid cursor")
		return
	}
	offset := page.Offset
	limit = int32(page.Limit)

	campaignID := r.URL.Query().Get("campaign_id")
	if campaignID == "" {
		campaignID = uuid.New().String()
	}

	totalRows := int64(15)
	mockRows := make([]KeywordReportRowDTO, 0, totalRows)
	keywords := []string{"insurance", "loans", "credit card", "mortgage", "attorney", "lawyer", "donate", "conference", "degree", "hosting", "claim", "software", "recovery", "transfer", "gas"}
	for i := int64(0); i < totalRows; i++ {
		mockRows = append(mockRows, toKeywordReportRowDTO(keywordReportCHRow{
			Keyword:      keywords[i],
			CampaignID:   campaignID,
			Impressions:  5000 + i*200,
			Clicks:       200 + i*10,
			Conversions:  5 + i,
			SpendMicro:   25000000 + i*1000000,
			RevenueMicro: 30000000 + i*1500000,
		}))
	}

	total := totalRows
	start := int64(offset)
	var paginatedRows []KeywordReportRowDTO
	if start >= totalRows {
		paginatedRows = []KeywordReportRowDTO{}
	} else {
		end := start + int64(limit)
		if end > totalRows {
			end = totalRows
		}
		paginatedRows = mockRows[start:end]
	}

	var nextCursor string
	if start+int64(limit) < total {
		nextCursor = coldpath.EncodeCursor(offset + int(limit))
	}

	resp := KeywordReportResponse{
		Rows:       paginatedRows,
		Freshness:  reports.reportFreshness(r.Context()),
		NextCursor: nextCursor,
	}

	httpresponse.JSON(w, http.StatusOK, resp)
}

const (
	forecastHandlerTimeout = 2 * time.Second
)

func (h *ReportsHTTPHandlers) registerCampaignStats(mux *http.ServeMux) {
	if h.CampaignStats == nil {
		return
	}
	limit := h.ApplyRateLimit
	permAny := h.RequireAnyPermission
	if permAny == nil {
		perm := h.RequirePermission
		permAny = func(perms []string, next http.HandlerFunc) http.HandlerFunc {
			if len(perms) == 0 {
				return next
			}
			return perm(perms[0], next)
		}
	}
	mux.HandleFunc("GET /api/v1/campaigns/{id}/stats", limit(permAny([]string{"campaigns:read", "campaigns:read:masked"}, h.getCampaignStats)))
}

func (h *ReportsHTTPHandlers) getCampaignStats(w http.ResponseWriter, r *http.Request) {
	idStr := r.PathValue("id")
	campaignID, err := uuid.Parse(idStr)
	if err != nil {
		httpresponse.Error(w, http.StatusBadRequest, "BAD_REQUEST", "invalid campaign id")
		return
	}

	if h.AuthorizeCampaignAccess != nil {
		if err := h.AuthorizeCampaignAccess(r, campaignID); err != nil {
			h.writeServiceError(w, err)
			return
		}
	}

	from, to, granularity, err := parseStatsQuery(r)
	if err != nil {
		h.writeServiceError(w, err)
		return
	}

	report, err := h.CampaignStats.GetCampaignStats(r.Context(), campaignID, from, to, granularity)
	if err != nil {
		h.writeServiceError(w, err)
		return
	}

	httpresponse.JSON(w, http.StatusOK, report)
}

func (h *ReportsHTTPHandlers) registerCampaignForecast(mux *http.ServeMux) {
	if h.CampaignForecaster == nil {
		return
	}
	limit := h.ApplyRateLimit
	perm := h.RequirePermission
	mux.HandleFunc("POST /api/v1/forecast/campaign", limit(perm("campaigns:read", h.forecastCampaign)))
}

func (h *ReportsHTTPHandlers) forecastCampaign(w http.ResponseWriter, r *http.Request) {
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

	customerID, err := h.resolveForecastCustomerID(r, req.CustomerID)
	if err != nil {
		h.writeServiceError(w, err)
		return
	}

	budgetLegacy := 0.0
	hasLegacy := req.BudgetLimit != nil
	if hasLegacy {
		budgetLegacy = *req.BudgetLimit
	}
	budgetMicro, err := forecastParseBudgetMicro(req.BudgetLimitMicro, budgetLegacy, hasLegacy)
	if err != nil {
		h.writeServiceError(w, err)
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

	out, err := h.CampaignForecaster.ForecastCampaign(ctx, CampaignForecastInput{
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
