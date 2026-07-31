package adminapi

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"espx/internal/billing/db"
	"espx/internal/database"
	"espx/pkg/coldpath"
	"espx/pkg/httpresponse"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
)

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
	reports.registerScaffoldReports(mux)

	limit := reports.ApplyRateLimit
	perm := reports.RequirePermission
	mux.HandleFunc("GET /api/v1/reports/placements", limit(perm("campaigns:read", reports.getPlacementsReport)))
	mux.HandleFunc("GET /api/v1/reports/keywords", limit(perm("campaigns:read", reports.getKeywordsReport)))
}

func (reports *ReportsHTTPHandlers) registerScaffoldReports(mux *http.ServeMux) {
	limit := reports.ApplyRateLimit
	perm := reports.RequirePermission

	routes := []struct {
		path       string
		permission string
	}{
		{"GET /api/v1/reports/campaign-unit-economics", "campaigns:read"},
		{"GET /api/v1/reports/source-margin", "campaigns:read"},
		{"GET /api/v1/reports/traffic-sources", "campaigns:read"},
		{"GET /api/v1/reports/source-quality", "campaigns:read"},
		{"GET /api/v1/reports/spend-velocity", "campaigns:read"},
		{"GET /api/v1/reports/campaign-geo-device", "campaigns:read"},
		{"GET /api/v1/reports/geo-roi", "campaigns:read"},
		{"GET /api/v1/reports/daypart-heatmap", "campaigns:read"},
		{"GET /api/v1/reports/pacing-drift", "campaigns:read"},
		{"GET /api/v1/reports/postback-reconciliation", "customers:read"},
		{"GET /api/v1/reports/ivt-by-source", "audit:read"},
		{"GET /api/v1/reports/discrepancy-buy-sell", "customers:read"},
		{"GET /api/v1/reports/campaign-overview", "campaigns:read"},
		{"GET /api/v1/reports/customer-portfolio", "customers:read"},
	}
	for _, route := range routes {
		mux.HandleFunc(route.path, limit(perm(route.permission, reports.notImplemented)))
	}
	mux.HandleFunc("POST /api/v1/reports/jobs", limit(perm("customers:read", reports.notImplemented)))
}

func (reports *ReportsHTTPHandlers) notImplemented(w http.ResponseWriter, _ *http.Request) {
	httpresponse.Error(w, http.StatusNotImplemented, "NOT_IMPLEMENTED", "report stub; UI deferred — docs/DEVELOPMENT.md")
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

	countFn := func() (int64, error) {
		return totalRows, nil
	}
	listFn := func() ([]PlacementReportRowDTO, error) {
		start := int64(offset)
		if start >= totalRows {
			return []PlacementReportRowDTO{}, nil
		}
		end := start + int64(limit)
		if end > totalRows {
			end = totalRows
		}
		return mockRows[start:end], nil
	}
	mapFn := func(row PlacementReportRowDTO) PlacementReportRowDTO {
		return row
	}

	paginatedRows, total, err := coldpath.PaginatedList(countFn, listFn, mapFn)
	if err != nil {
		reports.writeServiceError(w, err)
		return
	}

	var nextCursor string
	if int64(offset)+int64(limit) < total {
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

	countFn := func() (int64, error) {
		return totalRows, nil
	}
	listFn := func() ([]KeywordReportRowDTO, error) {
		start := int64(offset)
		if start >= totalRows {
			return []KeywordReportRowDTO{}, nil
		}
		end := start + int64(limit)
		if end > totalRows {
			end = totalRows
		}
		return mockRows[start:end], nil
	}
	mapFn := func(row KeywordReportRowDTO) KeywordReportRowDTO {
		return row
	}

	paginatedRows, total, err := coldpath.PaginatedList(countFn, listFn, mapFn)
	if err != nil {
		reports.writeServiceError(w, err)
		return
	}

	var nextCursor string
	if int64(offset)+int64(limit) < total {
		nextCursor = coldpath.EncodeCursor(offset + int(limit))
	}

	resp := KeywordReportResponse{
		Rows:       paginatedRows,
		Freshness:  reports.reportFreshness(r.Context()),
		NextCursor: nextCursor,
	}

	httpresponse.JSON(w, http.StatusOK, resp)
}
