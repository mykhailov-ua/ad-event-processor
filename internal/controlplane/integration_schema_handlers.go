package controlplane

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"strings"
	"time"

	db "ad-event-processor/internal/domain/db"
	"ad-event-processor/internal/integrationschema"
	"ad-event-processor/internal/postback"
	"ad-event-processor/pkg/coldpath"
	"ad-event-processor/pkg/httpresponse"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
)

type IntegrationSchemaDTO struct {
	ID        string          `json:"id"`
	Name      string          `json:"name"`
	Version   int             `json:"version"`
	Kind      string          `json:"kind"`
	Schema    json.RawMessage `json:"schema"`
	CreatedAt time.Time       `json:"created_at"`
	UpdatedAt time.Time       `json:"updated_at"`
}

type CreateIntegrationSchemaRequest struct {
	Name    string          `json:"name"`
	Version int             `json:"version"`
	Schema  json.RawMessage `json:"schema"`
}

type ApplyIntegrationSchemaRequest struct {
	CampaignID string `json:"campaign_id"`
}

type IntegrationSchemaHTTPHandlers struct {
	Pool                  *pgxpool.Pool
	EncryptionKey         []byte
	TemplateCatalog       any
	ResolveTrackingDomain func(ctx context.Context) string
	ApplyRateLimit        func(http.HandlerFunc) http.HandlerFunc
	RequirePermission     func(string, http.HandlerFunc) http.HandlerFunc
}

func (h *IntegrationSchemaHTTPHandlers) Register(mux *http.ServeMux) {
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
	mux.HandleFunc("GET /api/v1/integration/schemas", limit(perm("campaigns:read", h.listSchemas)))
	mux.HandleFunc("POST /api/v1/integration/schemas", limit(perm("campaigns:write", h.createSchema)))
	mux.HandleFunc("GET /api/v1/integration/schemas/{id}", limit(perm("campaigns:read", h.getSchema)))
	mux.HandleFunc("POST /api/v1/integration/schemas/{id}/apply", limit(perm("campaigns:write", h.applySchema)))
	h.RegisterTemplateRoutes(mux)
}

func (h *IntegrationSchemaHTTPHandlers) listSchemas(w http.ResponseWriter, r *http.Request) {
	rows, err := h.Pool.Query(r.Context(), `
		SELECT id, name, version, kind, body, created_at, updated_at
		FROM integration_schemas
		ORDER BY name, version DESC`)
	if err != nil {
		httpresponse.Error(w, http.StatusInternalServerError, "INTERNAL_ERROR", err.Error())
		return
	}
	defer rows.Close()

	var out []IntegrationSchemaDTO
	for rows.Next() {
		dto, err := scanIntegrationSchemaRow(rows.Scan)
		if err != nil {
			httpresponse.Error(w, http.StatusInternalServerError, "INTERNAL_ERROR", err.Error())
			return
		}
		out = append(out, dto)
	}
	if out == nil {
		out = []IntegrationSchemaDTO{}
	}
	httpresponse.JSON(w, http.StatusOK, out)
}

func (h *IntegrationSchemaHTTPHandlers) getSchema(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(r.PathValue("id"))
	if err != nil {
		httpresponse.Error(w, http.StatusBadRequest, "BAD_REQUEST", "invalid id")
		return
	}
	row := h.Pool.QueryRow(r.Context(), `
		SELECT id, name, version, kind, body, created_at, updated_at
		FROM integration_schemas WHERE id = $1`, id)
	dto, err := scanIntegrationSchemaRow(row.Scan)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			httpresponse.Error(w, http.StatusNotFound, "NOT_FOUND", "schema not found")
			return
		}
		httpresponse.Error(w, http.StatusInternalServerError, "INTERNAL_ERROR", err.Error())
		return
	}
	httpresponse.JSON(w, http.StatusOK, dto)
}

func (h *IntegrationSchemaHTTPHandlers) createSchema(w http.ResponseWriter, r *http.Request) {
	body, err := coldpath.ReadLimitedBody(w, r, coldpath.DefaultMaxBody)
	if err != nil {
		return
	}
	var req CreateIntegrationSchemaRequest
	if err := json.Unmarshal(body, &req); err != nil {
		httpresponse.Error(w, http.StatusBadRequest, "BAD_REQUEST", "invalid json body")
		return
	}
	if err := integrationschema.ValidateName(req.Name); err != nil {
		httpresponse.Error(w, http.StatusBadRequest, "BAD_REQUEST", err.Error())
		return
	}
	if req.Version <= 0 {
		httpresponse.Error(w, http.StatusBadRequest, "BAD_REQUEST", "version must be positive")
		return
	}
	kind, _, err := integrationschema.ParseDocument(req.Schema)
	if err != nil {
		httpresponse.Error(w, http.StatusBadRequest, "BAD_REQUEST", err.Error())
		return
	}

	var id uuid.UUID
	err = h.Pool.QueryRow(r.Context(), `
		INSERT INTO integration_schemas (name, version, kind, body)
		VALUES ($1, $2, $3, $4::jsonb)
		RETURNING id`,
		strings.TrimSpace(req.Name), req.Version, string(kind), []byte(req.Schema),
	).Scan(&id)
	if err != nil {
		if isPgUniqueViolation(err) {
			httpresponse.Error(w, http.StatusConflict, "CONFLICT", "schema name/version already exists")
			return
		}
		httpresponse.Error(w, http.StatusInternalServerError, "INTERNAL_ERROR", err.Error())
		return
	}

	h.getSchemaByID(w, r, id)
}

func (h *IntegrationSchemaHTTPHandlers) getSchemaByID(w http.ResponseWriter, r *http.Request, id uuid.UUID) {
	row := h.Pool.QueryRow(r.Context(), `
		SELECT id, name, version, kind, body, created_at, updated_at
		FROM integration_schemas WHERE id = $1`, id)
	dto, err := scanIntegrationSchemaRow(row.Scan)
	if err != nil {
		httpresponse.Error(w, http.StatusInternalServerError, "INTERNAL_ERROR", err.Error())
		return
	}
	httpresponse.JSON(w, http.StatusCreated, dto)
}

func (h *IntegrationSchemaHTTPHandlers) applySchema(w http.ResponseWriter, r *http.Request) {
	schemaID, err := uuid.Parse(r.PathValue("id"))
	if err != nil {
		httpresponse.Error(w, http.StatusBadRequest, "BAD_REQUEST", "invalid id")
		return
	}
	body, err := coldpath.ReadLimitedBody(w, r, coldpath.DefaultMaxBody)
	if err != nil {
		return
	}
	var req ApplyIntegrationSchemaRequest
	if err := json.Unmarshal(body, &req); err != nil {
		httpresponse.Error(w, http.StatusBadRequest, "BAD_REQUEST", "invalid json body")
		return
	}
	campaignID, err := uuid.Parse(strings.TrimSpace(req.CampaignID))
	if err != nil {
		httpresponse.Error(w, http.StatusBadRequest, "BAD_REQUEST", "invalid campaign_id")
		return
	}

	var kind string
	var schemaBody []byte
	err = h.Pool.QueryRow(r.Context(), `
		SELECT kind, body FROM integration_schemas WHERE id = $1`, schemaID,
	).Scan(&kind, &schemaBody)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			httpresponse.Error(w, http.StatusNotFound, "NOT_FOUND", "schema not found")
			return
		}
		httpresponse.Error(w, http.StatusInternalServerError, "INTERNAL_ERROR", err.Error())
		return
	}

	q := db.New(h.Pool)
	_, err = q.GetCampaign(r.Context(), pgtype.UUID{Bytes: campaignID, Valid: true})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			httpresponse.Error(w, http.StatusNotFound, "NOT_FOUND", "campaign not found")
			return
		}
		httpresponse.Error(w, http.StatusInternalServerError, "INTERNAL_ERROR", err.Error())
		return
	}

	tx, err := h.Pool.Begin(r.Context())
	if err != nil {
		httpresponse.Error(w, http.StatusInternalServerError, "INTERNAL_ERROR", err.Error())
		return
	}
	applyCtx := r.Context()
	defer func() { _ = tx.Rollback(applyCtx) }()

	if _, err := tx.Exec(r.Context(), `
		UPDATE campaigns SET integration_schema_id = $2, updated_at = NOW() WHERE id = $1`,
		campaignID, schemaID); err != nil {
		httpresponse.Error(w, http.StatusInternalServerError, "INTERNAL_ERROR", err.Error())
		return
	}

	applied := map[string]string{"status": "ok", "kind": kind}
	switch integrationschema.Kind(kind) {
	case integrationschema.KindOutboundPostback:
		tpl, err := integrationschema.OutboundURLTemplateFromBody(schemaBody)
		if err != nil {
			httpresponse.Error(w, http.StatusBadRequest, "BAD_REQUEST", err.Error())
			return
		}
		key := h.EncryptionKey
		if len(key) == 0 {
			key = []byte("postback-encryption-secret-key32")
		}
		encToken, err := postback.EncryptAESGCM([]byte("integration-schema"), key)
		if err != nil {
			httpresponse.Error(w, http.StatusInternalServerError, "INTERNAL_ERROR", "encryption failed")
			return
		}
		txQ := db.New(tx)
		if err := txQ.UpsertPostbackConfig(r.Context(), db.UpsertPostbackConfigParams{
			CampaignID:        pgtype.UUID{Bytes: campaignID, Valid: true},
			Provider:          "webhook",
			UrlTemplate:       tpl,
			ApiTokenEncrypted: encToken,
			TargetEvent:       "conversion",
		}); err != nil {
			httpresponse.Error(w, http.StatusInternalServerError, "INTERNAL_ERROR", err.Error())
			return
		}
		applied["url_template"] = tpl
	case integrationschema.KindAffiliateReceivePostback:
		parsedKind, parsed, err := integrationschema.ParseDocument(schemaBody)
		if err != nil || parsedKind != integrationschema.KindAffiliateReceivePostback {
			httpresponse.Error(w, http.StatusBadRequest, "BAD_REQUEST", "invalid affiliate receive schema")
			return
		}
		recv := parsed.(*integrationschema.AffiliateReceivePostbackSchema)
		trackingDomain := ""
		if h.ResolveTrackingDomain != nil {
			trackingDomain = h.ResolveTrackingDomain(r.Context())
		}
		panelURL := integrationschema.BuildAffiliateReceivePanelURL(trackingDomain, recv)
		applied["panel_postback_url"] = panelURL
		if suffix := strings.TrimSpace(recv.OfferURLSuffix); suffix != "" {
			applied["offer_url_suffix"] = suffix
		}
	case integrationschema.KindInboundTokens:
		parsedKind, parsed, err := integrationschema.ParseDocument(schemaBody)
		if err != nil || parsedKind != integrationschema.KindInboundTokens {
			httpresponse.Error(w, http.StatusBadRequest, "BAD_REQUEST", "invalid inbound schema")
			return
		}
		inbound := parsed.(*integrationschema.InboundTokensSchema)
		trackingDomain := ""
		if h.ResolveTrackingDomain != nil {
			trackingDomain = h.ResolveTrackingDomain(r.Context())
		}
		trackingURL := integrationschema.BuildInboundTrackingURL(trackingDomain, inbound)
		if _, err := tx.Exec(r.Context(), `
			UPDATE campaigns
			SET integration_schema_id = $2, target_url = $3, updated_at = NOW()
			WHERE id = $1`, campaignID, schemaID, trackingURL); err != nil {
			httpresponse.Error(w, http.StatusInternalServerError, "INTERNAL_ERROR", err.Error())
			return
		}
		applied["target_url"] = trackingURL
	case integrationschema.KindStatusMapping:
		if _, err := tx.Exec(r.Context(), `
			UPDATE campaigns SET status_integration_schema_id = $2, updated_at = NOW() WHERE id = $1`,
			campaignID, schemaID); err != nil {
			httpresponse.Error(w, http.StatusInternalServerError, "INTERNAL_ERROR", err.Error())
			return
		}
	default:
		httpresponse.Error(w, http.StatusBadRequest, "BAD_REQUEST", "unsupported schema kind")
		return
	}

	if err := tx.Commit(r.Context()); err != nil {
		httpresponse.Error(w, http.StatusInternalServerError, "INTERNAL_ERROR", err.Error())
		return
	}
	httpresponse.JSON(w, http.StatusOK, applied)
}

func scanIntegrationSchemaRow(scan func(dest ...any) error) (IntegrationSchemaDTO, error) {
	var dto IntegrationSchemaDTO
	var id uuid.UUID
	var kind string
	var body []byte
	var created, updated time.Time
	if err := scan(&id, &dto.Name, &dto.Version, &kind, &body, &created, &updated); err != nil {
		return IntegrationSchemaDTO{}, err
	}
	dto.ID = id.String()
	dto.Kind = kind
	dto.Schema = json.RawMessage(body)
	dto.CreatedAt = created
	dto.UpdatedAt = updated
	return dto, nil
}
