package controlplane

import (
	"encoding/json"
	"net/http"
	"strings"

	"ad-event-processor/internal/migrationsource"
	"ad-event-processor/pkg/coldpath"
	"ad-event-processor/pkg/httpresponse"

	"github.com/google/uuid"
)

const migrationMaxBodyBytes = migrationsource.MaxPayloadBytes

type MigratePreviewRequest struct {
	SourceKind string          `json:"source_kind"`
	Payload    json.RawMessage `json:"payload"`
}

func (campaigns *CampaignsHTTPHandlers) registerMigrationRoutes(
	mux *http.ServeMux,
	limit func(http.HandlerFunc) http.HandlerFunc,
	perm func([]string, http.HandlerFunc) http.HandlerFunc,
) {
	mux.HandleFunc("GET /api/v1/campaigns/migrate/sources", limit(perm([]string{"campaigns:read"}, campaigns.listMigrationSources)))
	mux.HandleFunc("POST /api/v1/campaigns/migrate/preview", limit(perm([]string{"campaigns:write"}, campaigns.previewMigration)))
	mux.HandleFunc("POST /api/v1/campaigns/migrate/import", limit(perm([]string{"campaigns:write"}, campaigns.importMigration)))
}

func (campaigns *CampaignsHTTPHandlers) listMigrationSources(w http.ResponseWriter, r *http.Request) {
	httpresponse.JSON(w, http.StatusOK, migrationsource.SourcesResponse{
		Sources:         migrationsource.ListSources(),
		MaxPayloadBytes: migrationsource.MaxPayloadBytes,
	})
}

func (campaigns *CampaignsHTTPHandlers) previewMigration(w http.ResponseWriter, r *http.Request) {
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

type MigrateImportRequest struct {
	CustomerID       string          `json:"customer_id"`
	SourceKind       string          `json:"source_kind"`
	Payload          json.RawMessage `json:"payload"`
	NamePrefix       string          `json:"name_prefix,omitempty"`
	BudgetLimitMicro *int64          `json:"budget_limit_micro,omitempty"`
}

func (campaigns *CampaignsHTTPHandlers) importMigration(w http.ResponseWriter, r *http.Request) {
	if campaigns.Campaigns == nil {
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
	if campaigns.ResolveCustomerID != nil {
		customerID, err = campaigns.ResolveCustomerID(r, nonNilUUID(customerID))
		if err != nil {
			campaigns.writeServiceError(w, err)
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
	result, err := campaigns.Campaigns.ImportMigrationCampaigns(r.Context(), ImportMigrationSpec{
		CustomerID:       customerID,
		IdempotencyKey:   idempotencyKey,
		SourceKind:       kind,
		Payload:          payload,
		NamePrefix:       req.NamePrefix,
		BudgetLimitMicro: req.BudgetLimitMicro,
	})
	if err != nil {
		campaigns.writeServiceError(w, err)
		return
	}
	httpresponse.JSON(w, http.StatusCreated, result)
}

func bytesTrimSpaceJSON(raw json.RawMessage) []byte {
	return []byte(strings.TrimSpace(string(raw)))
}
