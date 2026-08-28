package campaign

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"

	"ad-event-processor/internal/migrationsource"
	"ad-event-processor/internal/reportjob"
	"ad-event-processor/pkg/coldpath"
	"ad-event-processor/pkg/httpresponse"

	"github.com/google/uuid"
)

const migrationMaxBodyBytes = migrationsource.MaxPayloadBytes

func (h *CampaignsHTTPHandlers) registerMigrationRoutes(
	mux *http.ServeMux,
	limit func(http.HandlerFunc) http.HandlerFunc,
	perm func([]string, http.HandlerFunc) http.HandlerFunc,
) {
	mux.HandleFunc("GET /api/v1/campaigns/migrate/sources", limit(perm([]string{"campaigns:read"}, h.listMigrationSources)))
	mux.HandleFunc("POST /api/v1/campaigns/migrate/preview", limit(perm([]string{"campaigns:write"}, h.previewMigration)))
	mux.HandleFunc("POST /api/v1/campaigns/import/validate", limit(perm([]string{"campaigns:write"}, h.previewMigration)))
	mux.HandleFunc("POST /api/v1/campaigns/import/validate/jobs", limit(perm([]string{"campaigns:write"}, h.postCampaignImportValidateJob)))
	mux.HandleFunc("GET /api/v1/campaigns/import/validate/jobs/{id}", limit(perm([]string{"campaigns:write"}, h.getCampaignImportValidateJob)))
	mux.HandleFunc("POST /api/v1/campaigns/migrate/import", limit(perm([]string{"campaigns:write"}, h.importMigration)))
	h.registerMigrationPullRoutes(mux, limit, perm)
}

func (h *CampaignsHTTPHandlers) listMigrationSources(w http.ResponseWriter, r *http.Request) {
	httpresponse.JSON(w, http.StatusOK, migrationsource.SourcesResponse{
		Sources:         migrationsource.ListSources(),
		MaxPayloadBytes: migrationsource.MaxPayloadBytes,
	})
}

func (h *CampaignsHTTPHandlers) previewMigration(w http.ResponseWriter, r *http.Request) {
	req, ok := coldpath.DecodeRequestOrBadRequest[MigratePreviewRequest](w, r, migrationMaxBodyBytes)
	if !ok {
		return
	}
	kind := migrationsource.SourceKind(strings.TrimSpace(req.SourceKind))
	if kind == "" {
		httpresponse.Error(w, http.StatusBadRequest, "BAD_REQUEST", "source_kind is required")
		return
	}
	payload := bytesTrimSpaceJSON(req.Payload)
	if len(payload) == 0 {
		httpresponse.Error(w, http.StatusBadRequest, "BAD_REQUEST", "payload is required")
		return
	}
	if len(payload) > migrationMaxBodyBytes {
		httpresponse.Error(w, http.StatusBadRequest, "BAD_REQUEST", "payload too large")
		return
	}
	result, err := migrationsource.Preview(kind, payload, nil)
	if err != nil {
		if strings.Contains(err.Error(), "not implemented") || strings.Contains(err.Error(), "unsupported") {
			httpresponse.Error(w, http.StatusBadRequest, "BAD_REQUEST", err.Error())
			return
		}
		httpresponse.Error(w, http.StatusBadRequest, "BAD_REQUEST", err.Error())
		return
	}
	httpresponse.JSON(w, http.StatusOK, result)
}

func (h *CampaignsHTTPHandlers) importMigration(w http.ResponseWriter, r *http.Request) {
	if h.Campaigns == nil {
		httpresponse.Error(w, http.StatusServiceUnavailable, "UNAVAILABLE", "campaign service unavailable")
		return
	}
	idempotencyKey := strings.TrimSpace(r.Header.Get("Idempotency-Key"))
	if idempotencyKey == "" {
		httpresponse.Error(w, http.StatusBadRequest, "BAD_REQUEST", "Idempotency-Key header is required")
		return
	}
	req, ok := coldpath.DecodeRequestOrBadRequest[MigrateImportRequest](w, r, migrationMaxBodyBytes)
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
			h.writeServiceError(w, err)
			return
		}
	}
	kind := migrationsource.SourceKind(strings.TrimSpace(req.SourceKind))
	if kind == "" {
		httpresponse.Error(w, http.StatusBadRequest, "BAD_REQUEST", "source_kind is required")
		return
	}
	payload := bytesTrimSpaceJSON(req.Payload)
	if len(payload) == 0 {
		httpresponse.Error(w, http.StatusBadRequest, "BAD_REQUEST", "payload is required")
		return
	}
	if len(payload) > migrationMaxBodyBytes {
		httpresponse.Error(w, http.StatusBadRequest, "BAD_REQUEST", "payload too large")
		return
	}
	result, err := h.Campaigns.ImportMigrationCampaigns(r.Context(), ImportMigrationSpec{
		CustomerID:       customerID,
		IdempotencyKey:   idempotencyKey,
		SourceKind:       kind,
		Payload:          payload,
		NamePrefix:       req.NamePrefix,
		BudgetLimitMicro: req.BudgetLimitMicro,
	})
	if err != nil {
		h.writeServiceError(w, err)
		return
	}
	httpresponse.JSON(w, http.StatusCreated, result)
}

func (h *CampaignsHTTPHandlers) postCampaignImportValidateJob(w http.ResponseWriter, r *http.Request) {
	if h.ReportJobs == nil {
		httpresponse.Error(w, http.StatusServiceUnavailable, "UNAVAILABLE", "import validation jobs not configured")
		return
	}
	req, ok := coldpath.DecodeRequestOrBadRequest[ImportValidateJobRequest](w, r, migrationMaxBodyBytes)
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
			h.writeServiceError(w, err)
			return
		}
	}
	kind := migrationsource.SourceKind(strings.TrimSpace(req.SourceKind))
	if kind == "" {
		httpresponse.Error(w, http.StatusBadRequest, "BAD_REQUEST", "source_kind is required")
		return
	}
	payload := bytesTrimSpaceJSON(req.Payload)
	if len(payload) == 0 {
		httpresponse.Error(w, http.StatusBadRequest, "BAD_REQUEST", "payload is required")
		return
	}
	jobID, err := h.ReportJobs.CreateJob(r.Context(), reportjob.ReportJobSpec{
		CustomerID:       customerID.String(),
		ReportKey:        reportjob.CampaignImportValidationReportKey,
		Format:           "json",
		ImportSourceKind: string(kind),
		ImportPayload:    payload,
	}, strings.TrimSpace(r.Header.Get("Idempotency-Key")))
	if err != nil {
		httpresponse.Error(w, http.StatusBadRequest, "BAD_REQUEST", err.Error())
		return
	}
	status, _ := h.ReportJobs.GetJob(r.Context(), jobID)
	httpresponse.JSON(w, http.StatusCreated, status)
}

func (h *CampaignsHTTPHandlers) getCampaignImportValidateJob(w http.ResponseWriter, r *http.Request) {
	if h.ReportJobs == nil {
		httpresponse.Error(w, http.StatusServiceUnavailable, "UNAVAILABLE", "import validation jobs not configured")
		return
	}
	jobID := r.PathValue("id")
	status, ok := h.ReportJobs.GetJob(r.Context(), jobID)
	if !ok {
		httpresponse.Error(w, http.StatusNotFound, "NOT_FOUND", "job not found")
		return
	}
	if status.ReportKey != reportjob.CampaignImportValidationReportKey {
		httpresponse.Error(w, http.StatusNotFound, "NOT_FOUND", "job not found")
		return
	}
	if h.ResolveCustomerID != nil {
		customerID, err := uuid.Parse(status.CustomerID)
		if err != nil {
			httpresponse.Error(w, http.StatusBadRequest, "BAD_REQUEST", "invalid job customer")
			return
		}
		if _, err := h.ResolveCustomerID(r, &customerID); err != nil {
			h.writeServiceError(w, err)
			return
		}
	}
	httpresponse.JSON(w, http.StatusOK, status)
}

func bytesTrimSpaceJSON(raw json.RawMessage) []byte {
	return []byte(strings.TrimSpace(string(raw)))
}

type migrationPullService interface {
	PreviewMigrationPull(ctx context.Context, spec PullMigrationPreviewSpec) (migrationsource.PreviewResult, error)
	ImportMigrationPull(ctx context.Context, spec PullMigrationImportSpec) (ImportMigrationResult, error)
}

func (h *CampaignsHTTPHandlers) registerMigrationPullRoutes(
	mux *http.ServeMux,
	limit func(http.HandlerFunc) http.HandlerFunc,
	perm func([]string, http.HandlerFunc) http.HandlerFunc,
) {
	write := []string{"campaigns:write"}
	mux.HandleFunc("POST /api/v1/campaigns/migrate/pull/preview", limit(perm(write, h.previewMigrationPull)))
	mux.HandleFunc("POST /api/v1/campaigns/migrate/pull/import", limit(perm(write, h.importMigrationPull)))
}

func (h *CampaignsHTTPHandlers) previewMigrationPull(w http.ResponseWriter, r *http.Request) {
	req, ok := coldpath.DecodeRequestOrBadRequest[MigratePullRequest](w, r, coldpath.DefaultMaxBody)
	if !ok {
		return
	}
	svc, ok := h.Campaigns.(migrationPullService)
	if !ok {
		httpresponse.Error(w, http.StatusServiceUnavailable, "UNAVAILABLE", "migration pull not configured")
		return
	}
	kind := migrationsource.SourceKind(strings.TrimSpace(req.SourceKind))
	if !migrationsource.PullSupported(kind) {
		httpresponse.Error(w, http.StatusBadRequest, "BAD_REQUEST", "source_kind does not support live pull")
		return
	}
	result, err := svc.PreviewMigrationPull(r.Context(), PullMigrationPreviewSpec{
		SourceKind: kind,
		BaseURL:    req.BaseURL,
		APIToken:   req.APIToken,
		PullPath:   req.PullPath,
	})
	if err != nil {
		h.writeServiceError(w, err)
		return
	}
	httpresponse.JSON(w, http.StatusOK, result)
}

func (h *CampaignsHTTPHandlers) importMigrationPull(w http.ResponseWriter, r *http.Request) {
	if h.Campaigns == nil {
		httpresponse.Error(w, http.StatusServiceUnavailable, "UNAVAILABLE", "campaign service unavailable")
		return
	}
	idempotencyKey := strings.TrimSpace(r.Header.Get("Idempotency-Key"))
	if idempotencyKey == "" {
		httpresponse.Error(w, http.StatusBadRequest, "BAD_REQUEST", "Idempotency-Key header is required")
		return
	}
	req, ok := coldpath.DecodeRequestOrBadRequest[MigratePullRequest](w, r, coldpath.DefaultMaxBody)
	if !ok {
		return
	}
	svc, ok := h.Campaigns.(migrationPullService)
	if !ok {
		httpresponse.Error(w, http.StatusServiceUnavailable, "UNAVAILABLE", "migration pull not configured")
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
			h.writeServiceError(w, err)
			return
		}
	}
	kind := migrationsource.SourceKind(strings.TrimSpace(req.SourceKind))
	if !migrationsource.PullSupported(kind) {
		httpresponse.Error(w, http.StatusBadRequest, "BAD_REQUEST", "source_kind does not support live pull")
		return
	}
	result, err := svc.ImportMigrationPull(r.Context(), PullMigrationImportSpec{
		PullMigrationPreviewSpec: PullMigrationPreviewSpec{
			SourceKind: kind,
			BaseURL:    req.BaseURL,
			APIToken:   req.APIToken,
			PullPath:   req.PullPath,
		},
		CustomerID:       customerID,
		IdempotencyKey:   idempotencyKey,
		NamePrefix:       req.NamePrefix,
		BudgetLimitMicro: req.BudgetLimitMicro,
	})
	if err != nil {
		h.writeServiceError(w, err)
		return
	}
	httpresponse.JSON(w, http.StatusCreated, result)
}

func (h *CampaignsHTTPHandlers) writeCampaignRevisionConflict(w http.ResponseWriter, r *http.Request, campaignID uuid.UUID, current CampaignDTO, req PatchCampaignRequest) {
	if h.RecordRevisionConflict != nil && req.ExpectedRevision != nil {
		h.RecordRevisionConflict(r.Context(), campaignID, strings.TrimSpace(*req.ExpectedRevision))
	}
	httpresponse.JSON(w, http.StatusConflict, buildCampaignConflictResponse(current, req))
}
