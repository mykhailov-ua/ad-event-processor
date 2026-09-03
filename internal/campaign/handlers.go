package campaign

import (
	"context"
	"errors"
	"net/http"
	"strings"

	"ad-event-processor/internal/controlplane/authz"
	"ad-event-processor/internal/database"
	"ad-event-processor/internal/reportjob"
	"ad-event-processor/pkg/coldpath"
	"ad-event-processor/pkg/httpresponse"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

type CampaignListResponse = ListResponse[CampaignDTO]

type CampaignReader interface {
	GetCampaign(ctx context.Context, campaignID uuid.UUID) (CampaignDTO, error)
	GetCampaignMargin(ctx context.Context, campaignID uuid.UUID) (CampaignMarginDTO, error)
	ListCampaigns(ctx context.Context, customerID uuid.UUID, status string, limit, offset int32) ([]CampaignDTO, int64, error)
	ListCampaignsFiltered(ctx context.Context, filter ListCampaignsFilter) ([]CampaignDTO, int64, error)
	CountCampaignStatusTotals(ctx context.Context, filter ListCampaignsFilter, searchQuery, pacingMode string) (CampaignStatusTotalsDTO, error)
	AttachCampaignListMarginBreach(ctx context.Context, items []CampaignDTO)
	PatchCampaign(ctx context.Context, campaignID uuid.UUID, req PatchCampaignRequest) (CampaignDTO, error)
	PublishCampaign(ctx context.Context, campaignID uuid.UUID, force bool) (CampaignDTO, error)
	EvaluateCampaignPublish(ctx context.Context, campaignID uuid.UUID) (CampaignPublishCheckDTO, error)
	AssignCampaignOwner(ctx context.Context, campaignID, ownerUserID uuid.UUID) error
	ListCampaignEvents(ctx context.Context, campaignID uuid.UUID, limit, offset int32) ([]CampaignEventDTO, int64, error)
	BlockCampaignPlacement(ctx context.Context, campaignID uuid.UUID, placementID string) error
	CloneCampaign(ctx context.Context, spec CloneCampaignSpec) (CloneCampaignResult, error)
	ExportCampaign(ctx context.Context, campaignID uuid.UUID) (CampaignExportBundle, error)
	ImportCampaign(ctx context.Context, spec ImportCampaignSpec) (ImportCampaignResult, error)
	ImportMigrationCampaigns(ctx context.Context, spec ImportMigrationSpec) (ImportMigrationResult, error)
	GetCampaignIntegrationHealth(ctx context.Context, campaignID uuid.UUID) (IntegrationHealthDTO, error)
	PauseCampaign(ctx context.Context, campaignID uuid.UUID, reason string) error
	ResumeCampaign(ctx context.Context, campaignID uuid.UUID, reason string) error
}

type CampaignsHTTPHandlers struct {
	Campaigns                 CampaignReader
	CampaignFraud             CampaignFraudService
	ConversionMappings        ConversionMappingService
	GetCampaignFlow           func(ctx context.Context, flowID uuid.UUID) (FlowDTO, error)
	ValidateCampaignFlowPaths CampaignFlowPathValidator
	RecordRevisionConflict    func(ctx context.Context, campaignID uuid.UUID, expectedRevision string)
	ClickHouseQuery           *database.ClickHouseQuery
	PostgresPool              *pgxpool.Pool
	MarginDefaultThresholdBps int
	ApplyRateLimit            func(http.HandlerFunc) http.HandlerFunc
	RequireAnyPermission      func([]string, http.HandlerFunc) http.HandlerFunc
	AuthorizeCampaignAccess   func(*http.Request, uuid.UUID) error
	ResolveCustomerID         func(*http.Request, *uuid.UUID) (uuid.UUID, error)
	AllowFraudPreview         func(campaignID string) bool
	LicenseFeatureAllowed     func(featureKey string) (allowed bool, planCode string)
	ReportJobs                *reportjob.ReportJobRunner
	WriteServiceError         func(http.ResponseWriter, error)
}

func (h *CampaignsHTTPHandlers) Register(mux *http.ServeMux) {
	if h == nil || h.Campaigns == nil {
		return
	}
	limit := h.ApplyRateLimit
	perm := h.RequireAnyPermission
	if limit == nil {
		limit = func(next http.HandlerFunc) http.HandlerFunc { return next }
	}
	if perm == nil {
		perm = func(_ []string, next http.HandlerFunc) http.HandlerFunc { return next }
	}
	mux.HandleFunc("GET /api/v1/campaigns", limit(perm([]string{"campaigns:read", "campaigns:read:masked"}, h.listCampaigns)))
	mux.HandleFunc("GET /api/v1/campaigns/list-facets", limit(perm([]string{"campaigns:read", "campaigns:read:masked"}, h.listCampaignListFacets)))
	mux.HandleFunc("GET /api/v1/campaigns/target-countries", limit(perm([]string{"campaigns:read", "campaigns:read:masked"}, h.listCampaignTargetCountries)))
	mux.HandleFunc("GET /api/v1/campaigns/metrics", limit(perm([]string{"campaigns:read", "campaigns:read:masked"}, h.listCampaignMetrics)))
	mux.HandleFunc("GET /api/v1/campaigns/metrics-totals", limit(perm([]string{"campaigns:read", "campaigns:read:masked"}, h.listCampaignMetricsTotals)))
	mux.HandleFunc("GET /api/v1/campaigns/{id}", limit(perm([]string{"campaigns:read", "campaigns:read:masked"}, h.getCampaign)))
	mux.HandleFunc("PATCH /api/v1/campaigns/{id}", limit(perm([]string{"campaigns:write"}, h.patchCampaign)))
	mux.HandleFunc("PUT /api/v1/campaigns/{id}/owner", limit(perm([]string{"campaigns:write"}, h.assignCampaignOwner)))
	mux.HandleFunc("GET /api/v1/campaigns/{id}/events", limit(perm([]string{"campaigns:read"}, h.listCampaignEvents)))
	mux.HandleFunc("GET /api/v1/campaigns/{id}/margin", limit(perm([]string{"campaigns:read"}, h.getCampaignMargin)))
	mux.HandleFunc("POST /api/v1/campaigns/{id}/placement-blocks", limit(perm([]string{"campaigns:write"}, h.blockCampaignPlacement)))
	mux.HandleFunc("POST /api/v1/campaigns/{id}/clone", limit(perm([]string{"campaigns:write"}, h.cloneCampaign)))
	mux.HandleFunc("GET /api/v1/campaigns/{id}/export", limit(perm([]string{"campaigns:read"}, h.exportCampaign)))
	mux.HandleFunc("POST /api/v1/campaigns/import", limit(perm([]string{"campaigns:write"}, h.importCampaign)))
	h.registerMigrationRoutes(mux, limit, perm)
	h.registerIntegrationHealthRoutes(mux, limit, perm)
	h.registerConversionMappingRoutes(mux, limit, perm)
	h.registerCampaignFraudRoutes(mux, limit, perm)
	h.registerCampaignEditorRoutes(mux, limit, perm)
	h.registerCampaignPublishRoutes(mux, limit, perm)
	h.registerCampaignSmokeRoutes(mux, limit, perm)
	h.registerCampaignWizardRoutes(mux, limit, perm)
}

func (h *CampaignsHTTPHandlers) listCampaigns(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()

	var customerID uuid.UUID
	if custStr := q.Get("customer_id"); custStr != "" {
		id, err := uuid.Parse(custStr)
		if err != nil {
			httpresponse.Error(w, http.StatusBadRequest, "BAD_REQUEST", "invalid customer_id")
			return
		}
		customerID = id
	}

	if h.ResolveCustomerID != nil {
		resolved, err := h.ResolveCustomerID(r, nonNilUUID(customerID))
		if err != nil {
			h.WriteHandlerError(w, err)
			return
		}
		customerID = resolved
	}

	status := q.Get("status")
	sortField, order, sortErr := parseListSort(r, CampaignListAllowedSortFields(), "updated_at")
	if sortErr != nil {
		httpresponse.Error(w, http.StatusBadRequest, "BAD_REQUEST", sortErr.Error())
		return
	}
	search := strings.TrimSpace(q.Get("q"))
	pacingMode := strings.TrimSpace(q.Get("pacing_mode"))
	limit, offset := coldpath.ParseAPIPagination(r)
	listFilter := ListCampaignsFilter{
		CustomerID:     customerID,
		Status:         status,
		OwnerUserID:    ResolveListOwnerUserFilter(r.Context(), r),
		TargetCountry:  parseTargetCountryQuery(r),
		BudgetMinMicro: parseOptionalBudgetMicroQuery(r, "budget_min_micro"),
		BudgetMaxMicro: parseOptionalBudgetMicroQuery(r, "budget_max_micro"),
		SearchQuery:    search,
		PacingMode:     pacingMode,
		SortField:      sortField,
		SortOrder:      order,
		Limit:          limit,
		Offset:         offset,
	}
	if IsCampaignListMetricWindowSortField(sortField) {
		from, to, rangeErr := parseCampaignListMetricsRange(r)
		if rangeErr != nil {
			httpresponse.Error(w, http.StatusBadRequest, "BAD_REQUEST", rangeErr.Error())
			return
		}
		statsFrom, statsTo := CampaignListPGStatsDates(from, to)
		listFilter.StatsFrom = statsFrom
		listFilter.StatsTo = statsTo
		listFilter.StatsRangeFrom = from
		listFilter.StatsRangeTo = to
		listFilter.StatsRangeSet = true
	}

	items, total, err := h.Campaigns.ListCampaignsFiltered(r.Context(), listFilter)
	if err != nil {
		h.WriteHandlerError(w, err)
		return
	}
	if listFilter.StatsRangeSet {
		AttachCampaignListMarginBreachInRange(
			r.Context(),
			h.PostgresPool,
			h.MarginDefaultThresholdBps,
			items,
			listFilter.StatsRangeFrom,
			listFilter.StatsRangeTo,
		)
	} else {
		h.Campaigns.AttachCampaignListMarginBreach(r.Context(), items)
	}
	for i := range items {
		items[i].StatusLabel = campaignStatusLabel(items[i].Status)
		items[i].StatusTone = campaignStatusTone(items[i].Status)
	}
	totalsFilter := listFilter
	totalsFilter.Status = ""
	statusTotals, totalsErr := h.Campaigns.CountCampaignStatusTotals(r.Context(), totalsFilter, search, pacingMode)
	if totalsErr != nil {
		h.WriteHandlerError(w, totalsErr)
		return
	}
	httpresponse.JSON(w, http.StatusOK, ListEnvelope[CampaignDTO]{
		Items:          items,
		Total:          total,
		Limit:          limit,
		Offset:         offset,
		FiltersApplied: filtersAppliedFromQuery(r, "customer_id", "status", "q", "pacing_mode", "owner_user_id", "country", "budget_min_micro", "budget_max_micro"),
		Sort:           &ListSortDTO{Field: sortField, Order: order},
		StatusTotals:   &statusTotals,
	})
}

func (h *CampaignsHTTPHandlers) listCampaignMetrics(w http.ResponseWriter, r *http.Request) {
	if h.PostgresPool == nil {
		httpresponse.Error(w, http.StatusServiceUnavailable, "UNAVAILABLE", "service unavailable")
		return
	}
	campaignIDs, err := ParseCampaignListMetricsIDs(r.URL.Query().Get("ids"))
	if err != nil {
		httpresponse.Error(w, http.StatusBadRequest, "BAD_REQUEST", err.Error())
		return
	}
	if h.AuthorizeCampaignAccess != nil {
		for _, campaignID := range campaignIDs {
			if authErr := h.AuthorizeCampaignAccess(r, campaignID); authErr != nil {
				h.WriteHandlerError(w, authErr)
				return
			}
		}
	}
	from, to, err := parseCampaignListMetricsRange(r)
	if err != nil {
		httpresponse.Error(w, http.StatusBadRequest, "BAD_REQUEST", err.Error())
		return
	}
	report, err := BatchCampaignListMetrics(
		r.Context(),
		h.PostgresPool,
		h.ClickHouseQuery,
		h.MarginDefaultThresholdBps,
		campaignIDs,
		from,
		to,
	)
	if err != nil {
		h.WriteHandlerError(w, err)
		return
	}
	httpresponse.JSON(w, http.StatusOK, report)
}

func (h *CampaignsHTTPHandlers) campaignListFilterFromRequest(r *http.Request) (ListCampaignsFilter, error) {
	q := r.URL.Query()

	var customerID uuid.UUID
	if custStr := q.Get("customer_id"); custStr != "" {
		id, err := uuid.Parse(custStr)
		if err != nil {
			return ListCampaignsFilter{}, invalidQueryError("invalid customer_id")
		}
		customerID = id
	}
	if h.ResolveCustomerID != nil {
		resolved, err := h.ResolveCustomerID(r, nonNilUUID(customerID))
		if err != nil {
			return ListCampaignsFilter{}, err
		}
		customerID = resolved
	}

	return ListCampaignsFilter{
		CustomerID:     customerID,
		Status:         q.Get("status"),
		OwnerUserID:    ResolveListOwnerUserFilter(r.Context(), r),
		TargetCountry:  parseTargetCountryQuery(r),
		BudgetMinMicro: parseOptionalBudgetMicroQuery(r, "budget_min_micro"),
		BudgetMaxMicro: parseOptionalBudgetMicroQuery(r, "budget_max_micro"),
		SearchQuery:    strings.TrimSpace(q.Get("q")),
		PacingMode:     strings.TrimSpace(q.Get("pacing_mode")),
	}, nil
}

func (h *CampaignsHTTPHandlers) getCampaign(w http.ResponseWriter, r *http.Request) {
	campaignID, ok := h.ParseCampaignID(w, r)
	if !ok {
		return
	}
	campaign, err := h.Campaigns.GetCampaign(r.Context(), campaignID)
	if err != nil {
		h.WriteHandlerError(w, err)
		return
	}
	httpresponse.JSON(w, http.StatusOK, campaign)
}

func (h *CampaignsHTTPHandlers) getCampaignMargin(w http.ResponseWriter, r *http.Request) {
	campaignID, ok := h.ParseCampaignID(w, r)
	if !ok {
		return
	}
	margin, err := h.Campaigns.GetCampaignMargin(r.Context(), campaignID)
	if err != nil {
		h.WriteHandlerError(w, err)
		return
	}
	httpresponse.JSON(w, http.StatusOK, margin)
}

func (h *CampaignsHTTPHandlers) patchCampaign(w http.ResponseWriter, r *http.Request) {
	campaignID, ok := h.ParseCampaignID(w, r)
	if !ok {
		return
	}
	body, err := coldpath.ReadLimitedBody(w, r, coldpath.DefaultMaxBody)
	if err != nil {
		return
	}
	req, err := coldpath.DecodeBody[PatchCampaignRequest](body)
	if err != nil {
		httpresponse.Error(w, http.StatusBadRequest, "BAD_REQUEST", "invalid request body")
		return
	}
	resolveExpectedRevision(r, &req)
	req.PublishForce = ParsePublishForceQuery(r.URL.Query().Get("force"))
	if req.PublishForce && !CanForceCampaignPublish(r.Context()) {
		httpresponse.Error(w, http.StatusForbidden, "FORBIDDEN", "force publish requires admin role")
		return
	}
	if req.ExpectedRevision != nil {
		current, getErr := h.Campaigns.GetCampaign(r.Context(), campaignID)
		if getErr != nil {
			h.WriteHandlerError(w, getErr)
			return
		}
		if campaignRevision(current.UpdatedAt) != strings.TrimSpace(*req.ExpectedRevision) {
			h.writeCampaignRevisionConflict(w, r, campaignID, current, req)
			return
		}
	}
	updated, err := h.Campaigns.PatchCampaign(r.Context(), campaignID, req)
	if err != nil {
		if errors.Is(err, ErrCampaignRevisionConflict) {
			current, getErr := h.Campaigns.GetCampaign(r.Context(), campaignID)
			if getErr != nil {
				h.WriteHandlerError(w, getErr)
				return
			}
			h.writeCampaignRevisionConflict(w, r, campaignID, current, req)
			return
		}
		h.WriteHandlerError(w, err)
		return
	}
	httpresponse.JSON(w, http.StatusOK, updated)
}

func (h *CampaignsHTTPHandlers) assignCampaignOwner(w http.ResponseWriter, r *http.Request) {
	campaignID, ok := h.ParseCampaignID(w, r)
	if !ok {
		return
	}
	req, ok := coldpath.DecodeRequestOrBadRequest[AssignCampaignOwnerRequest](w, r, coldpath.DefaultMaxBody)
	if !ok {
		return
	}
	ownerID, err := uuid.Parse(strings.TrimSpace(req.UserID))
	if err != nil {
		httpresponse.Error(w, http.StatusBadRequest, "BAD_REQUEST", "invalid user_id")
		return
	}
	if err := h.Campaigns.AssignCampaignOwner(r.Context(), campaignID, ownerID); err != nil {
		h.WriteHandlerError(w, err)
		return
	}
	httpresponse.JSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

func (h *CampaignsHTTPHandlers) blockCampaignPlacement(w http.ResponseWriter, r *http.Request) {
	campaignID, ok := h.ParseCampaignID(w, r)
	if !ok {
		return
	}
	body, err := coldpath.ReadLimitedBody(w, r, coldpath.DefaultMaxBody)
	if err != nil {
		return
	}
	req, err := coldpath.DecodeBody[struct {
		PlacementID string `json:"placement_id"`
	}](body)
	if err != nil || strings.TrimSpace(req.PlacementID) == "" {
		httpresponse.Error(w, http.StatusBadRequest, "BAD_REQUEST", "invalid request body")
		return
	}
	if err := h.Campaigns.BlockCampaignPlacement(r.Context(), campaignID, req.PlacementID); err != nil {
		h.WriteHandlerError(w, err)
		return
	}
	w.WriteHeader(http.StatusAccepted)
}

type CloneCampaignHTTPRequest struct {
	NamePrefix string               `json:"name_prefix,omitempty"`
	NameSuffix string               `json:"name_suffix,omitempty"`
	Options    CloneCampaignOptions `json:"options,omitempty"`
}

func (h *CampaignsHTTPHandlers) exportCampaign(w http.ResponseWriter, r *http.Request) {
	campaignID, ok := h.ParseCampaignID(w, r)
	if !ok {
		return
	}
	bundle, err := h.Campaigns.ExportCampaign(r.Context(), campaignID)
	if err != nil {
		h.WriteHandlerError(w, err)
		return
	}
	httpresponse.JSON(w, http.StatusOK, bundle)
}

type ImportCampaignHTTPRequest struct {
	CustomerID       string `json:"customer_id"`
	NameOverride     string `json:"name_override,omitempty"`
	BudgetLimitMicro *int64 `json:"budget_limit_micro,omitempty"`
	CampaignExportBundle
}

func (h *CampaignsHTTPHandlers) importCampaign(w http.ResponseWriter, r *http.Request) {
	idempotencyKey := strings.TrimSpace(r.Header.Get("Idempotency-Key"))
	if idempotencyKey == "" {
		httpresponse.Error(w, http.StatusBadRequest, "BAD_REQUEST", "Idempotency-Key header is required")
		return
	}
	req, ok := coldpath.DecodeRequestOrBadRequest[ImportCampaignHTTPRequest](w, r, coldpath.DefaultMaxBody)
	if !ok {
		return
	}
	customerID, err := uuid.Parse(strings.TrimSpace(req.CustomerID))
	if err != nil {
		httpresponse.Error(w, http.StatusBadRequest, "BAD_REQUEST", "invalid customer_id")
		return
	}
	if h.ResolveCustomerID != nil {
		customerID, err = h.ResolveCustomerID(r, nonNilUUID(customerID))
		if err != nil {
			h.WriteHandlerError(w, err)
			return
		}
	}
	result, err := h.Campaigns.ImportCampaign(r.Context(), ImportCampaignSpec{
		CustomerID:     customerID,
		NameOverride:   req.NameOverride,
		BudgetOverride: req.BudgetLimitMicro,
		IdempotencyKey: idempotencyKey,
		Bundle:         req.CampaignExportBundle,
	})
	if err != nil {
		h.WriteHandlerError(w, err)
		return
	}
	httpresponse.JSON(w, http.StatusCreated, result)
}

func (h *CampaignsHTTPHandlers) cloneCampaign(w http.ResponseWriter, r *http.Request) {
	campaignID, ok := h.ParseCampaignID(w, r)
	if !ok {
		return
	}
	idempotencyKey := strings.TrimSpace(r.Header.Get("Idempotency-Key"))
	if idempotencyKey == "" {
		httpresponse.Error(w, http.StatusBadRequest, "BAD_REQUEST", "Idempotency-Key header is required")
		return
	}
	req, ok := coldpath.DecodeRequestOrBadRequest[CloneCampaignHTTPRequest](w, r, coldpath.DefaultMaxBody)
	if !ok {
		return
	}
	result, err := h.Campaigns.CloneCampaign(r.Context(), CloneCampaignSpec{
		SourceID:       campaignID,
		NamePrefix:     req.NamePrefix,
		NameSuffix:     req.NameSuffix,
		IdempotencyKey: idempotencyKey,
		Options:        req.Options,
	})
	if err != nil {
		h.WriteHandlerError(w, err)
		return
	}
	httpresponse.JSON(w, http.StatusCreated, result)
}

func (h *CampaignsHTTPHandlers) listCampaignEvents(w http.ResponseWriter, r *http.Request) {
	campaignID, ok := h.ParseCampaignID(w, r)
	if !ok {
		return
	}
	limit, offset := coldpath.ParseAPIPagination(r)
	items, total, err := h.Campaigns.ListCampaignEvents(r.Context(), campaignID, limit, offset)
	if err != nil {
		h.WriteHandlerError(w, err)
		return
	}
	if strings.TrimSpace(r.URL.Query().Get("format")) == "timeline" {
		masked := maskLevelFromContext(r.Context()) != authz.MaskFull
		httpresponse.JSON(w, http.StatusOK, buildCampaignEventTimeline(items, masked))
		return
	}
	httpresponse.JSON(w, http.StatusOK, CampaignEventListResponse{Items: items, Total: total})
}

func (h *CampaignsHTTPHandlers) ParseCampaignID(w http.ResponseWriter, r *http.Request) (uuid.UUID, bool) {
	campaignID, err := coldpath.ParsePathUUID(r, "id")
	if err != nil {
		httpresponse.Error(w, http.StatusBadRequest, "BAD_REQUEST", "invalid campaign id")
		return uuid.Nil, false
	}
	if h.AuthorizeCampaignAccess != nil {
		if err := h.AuthorizeCampaignAccess(r, campaignID); err != nil {
			h.WriteHandlerError(w, err)
			return uuid.Nil, false
		}
	}
	return campaignID, true
}

func (h *CampaignsHTTPHandlers) WriteHandlerError(w http.ResponseWriter, err error) {
	if h.WriteServiceError != nil {
		h.WriteServiceError(w, err)
		return
	}
	httpresponse.Error(w, http.StatusInternalServerError, "INTERNAL", "internal error")
}

func NonNilUUID(id uuid.UUID) *uuid.UUID {
	return nonNilUUID(id)
}

func nonNilUUID(id uuid.UUID) *uuid.UUID {
	if id == uuid.Nil {
		return nil
	}
	return &id
}
