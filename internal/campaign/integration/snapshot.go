package integration

import (
	"context"
	"errors"
	"net/http"
	"sync"

	"ad-event-processor/internal/integrationschema"
	"ad-event-processor/pkg/httpresponse"
)

type IntegrationSnapshotDTO struct {
	Schemas   []IntegrationSchemaDTO                   `json:"schemas"`
	Templates []integrationschema.TemplateCatalogEntry `json:"templates"`
}

func (h *IntegrationSchemaHTTPHandlers) listIntegrationSchemas(ctx context.Context) ([]IntegrationSchemaDTO, error) {
	if h == nil || h.Pool == nil {
		return nil, errors.New("integration schema handler not configured")
	}
	rows, err := h.Pool.Query(ctx, `
		SELECT id, name, version, kind, body, created_at, updated_at
		FROM integration_schemas
		ORDER BY name, version DESC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var out []IntegrationSchemaDTO
	for rows.Next() {
		dto, err := scanIntegrationSchemaRow(rows.Scan)
		if err != nil {
			return nil, err
		}
		out = append(out, dto)
	}
	if out == nil {
		out = []IntegrationSchemaDTO{}
	}
	return out, nil
}

func (h *IntegrationSchemaHTTPHandlers) listIntegrationTemplates(ctx context.Context) ([]integrationschema.TemplateCatalogEntry, error) {
	svc, ok := h.TemplateCatalog.(TemplateCatalogService)
	if !ok || svc == nil {
		return nil, errors.New("template catalog unavailable")
	}
	out := svc.ListBundledTemplates(ctx)
	if out == nil {
		out = []integrationschema.TemplateCatalogEntry{}
	}
	return out, nil
}

func (h *IntegrationSchemaHTTPHandlers) getIntegrationSnapshot(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	var snap IntegrationSnapshotDTO
	var schemasErr, templatesErr error
	var wg sync.WaitGroup
	wg.Add(2)

	go func() {
		defer wg.Done()
		snap.Schemas, schemasErr = h.listIntegrationSchemas(ctx)
	}()
	go func() {
		defer wg.Done()
		snap.Templates, templatesErr = h.listIntegrationTemplates(ctx)
	}()
	wg.Wait()

	if schemasErr != nil {
		httpresponse.Error(w, http.StatusInternalServerError, "INTERNAL_ERROR", schemasErr.Error())
		return
	}
	if templatesErr != nil {
		httpresponse.Error(w, http.StatusServiceUnavailable, "UNAVAILABLE", templatesErr.Error())
		return
	}
	httpresponse.JSON(w, http.StatusOK, snap)
}

func (h *IntegrationSchemaHTTPHandlers) registerSnapshotRoute(mux *http.ServeMux) {
	if h == nil || h.Pool == nil {
		return
	}
	limit := h.ApplyRateLimit
	perm := h.RequirePermission
	if limit == nil {
		limit = func(next http.HandlerFunc) http.HandlerFunc { return next }
	}
	if perm == nil {
		perm = func(_ string, next http.HandlerFunc) http.HandlerFunc { return next }
	}
	mux.HandleFunc("GET /api/v1/integration/snapshot", limit(perm("campaigns:read", h.getIntegrationSnapshot)))
}
