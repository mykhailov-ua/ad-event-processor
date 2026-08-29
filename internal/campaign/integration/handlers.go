package integration

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"sort"
	"strings"
	"time"

	"ad-event-processor/internal/campaign"

	"ad-event-processor/internal/controlplane/authz"
	"ad-event-processor/internal/domain"
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

type (
	IntegrationSchemaDTO           = campaign.IntegrationSchemaDTO
	CreateIntegrationSchemaRequest struct {
		Name    string          `json:"name"`
		Version int             `json:"version"`
		Schema  json.RawMessage `json:"schema"`
	}
)

type ApplyIntegrationSchemaRequest struct {
	CampaignID string `json:"campaign_id"`
}

type AffiliateStatusPresetDTO = campaign.AffiliateStatusPresetDTO

type AffiliateStatusPresetEntryDTO = campaign.AffiliateStatusPresetEntryDTO

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
	mux.HandleFunc("GET /api/v1/integration/affiliate-status-presets", limit(perm("campaigns:read", h.listAffiliateStatusPresets)))
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
		if campaign.IsPgUniqueViolation(err) {
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

type IntegrationHealthInput struct {
	CampaignID                uuid.UUID
	IntegrationSchemaBound    bool
	TrafficTemplateID         string
	TargetURL                 string
	ClickQueryParams          map[string]string
	IngressCostConfigured     bool
	IngressCostParam          string
	PostbackConfigured        bool
	CostSyncNetwork           string
	CostSyncCredentialPresent bool
}

var trafficTemplateCostSyncNetwork = map[string]string{
	"meta-facebook":  "facebook",
	"meta-instagram": "facebook",
	"google-ads":     "google",
	"google-display": "google",
	"youtube-ads":    "google",
	"tiktok-ads":     "tiktok",
}

var costSyncRequiredKeys = map[string][]string{
	"facebook": {"ad_campaign_id", "sub2", "sub4", "fbclid"},
	"google":   {"ad_campaign_id", "sub2", "sub3", "gclid"},
	"tiktok":   {"ad_campaign_id", "sub2", "sub3", "ttclid"},
}

func BuildCampaignIntegrationHealth(input IntegrationHealthInput) campaign.IntegrationHealthDTO {
	rows := make([]campaign.IntegrationHealthRow, 0, 6)
	trackingRoute := fmt.Sprintf("/campaigns/%s?tab=tracking", input.CampaignID)
	configRoute := fmt.Sprintf("/campaigns/%s?tab=config", input.CampaignID)
	postbackRoute := fmt.Sprintf("/campaigns/%s?tab=postbacks", input.CampaignID)
	costSyncRoute := "/integrations/cost-sync"

	if input.IntegrationSchemaBound {
		rows = append(rows, campaign.IntegrationHealthRow{
			Slug:    "integration_schema",
			Status:  string(campaign.IntegrationHealthOK),
			Message: "Integration schema is bound on the campaign.",
		})
	} else if template := strings.TrimSpace(input.TrafficTemplateID); template != "" && template != "direct-custom" {
		rows = append(rows, campaign.IntegrationHealthRow{
			Slug:    "traffic_template",
			Status:  string(campaign.IntegrationHealthOK),
			Message: "Traffic template preset is saved on the campaign.",
		})
	} else {
		rows = append(rows, campaign.IntegrationHealthRow{
			Slug:     "integration_schema",
			Status:   string(campaign.IntegrationHealthWarn),
			Message:  "No integration schema or traffic template preset; apply a bundled template on Integration.",
			FixRoute: trackingRoute,
		})
	}

	if input.TargetURL == "" {
		rows = append(rows, campaign.IntegrationHealthRow{
			Slug:     "target_url",
			Status:   string(campaign.IntegrationHealthWarn),
			Message:  "Target URL is empty; clicks have nowhere to land after tracking.",
			FixRoute: configRoute,
		})
	} else {
		rows = append(rows, campaign.IntegrationHealthRow{
			Slug:    "target_url",
			Status:  string(campaign.IntegrationHealthOK),
			Message: "Target URL is configured.",
		})
	}

	if network := input.CostSyncNetwork; network != "" {
		required := costSyncRequiredKeys[network]
		missing := missingClickJoinKeys(input.ClickQueryParams, required)
		if len(missing) > 0 {
			rows = append(rows, campaign.IntegrationHealthRow{
				Slug:     "click_join_keys",
				Status:   string(campaign.IntegrationHealthFail),
				Message:  fmt.Sprintf("Click preset missing Cost Sync join keys: %s", strings.Join(missing, ", ")),
				FixRoute: trackingRoute,
			})
		} else {
			rows = append(rows, campaign.IntegrationHealthRow{
				Slug:    "click_join_keys",
				Status:  string(campaign.IntegrationHealthOK),
				Message: "Required Cost Sync join keys are present in the click preset.",
			})
		}
		if !input.CostSyncCredentialPresent {
			rows = append(rows, campaign.IntegrationHealthRow{
				Slug:     "cost_sync_credential",
				Status:   string(campaign.IntegrationHealthWarn),
				Message:  fmt.Sprintf("No Cost Sync credential for network %s; spend join stays empty.", network),
				FixRoute: costSyncRoute,
			})
		} else {
			rows = append(rows, campaign.IntegrationHealthRow{
				Slug:    "cost_sync_credential",
				Status:  string(campaign.IntegrationHealthOK),
				Message: fmt.Sprintf("Cost Sync credential configured for %s.", network),
			})
		}
	}

	if input.PostbackConfigured {
		rows = append(rows, campaign.IntegrationHealthRow{
			Slug:    "postback_config",
			Status:  string(campaign.IntegrationHealthOK),
			Message: "Postback or CAPI template is configured.",
		})
	} else {
		rows = append(rows, campaign.IntegrationHealthRow{
			Slug:     "postback_config",
			Status:   string(campaign.IntegrationHealthWarn),
			Message:  "No postback config; affiliate conversions will not forward until CAPI & Postbacks is set.",
			FixRoute: postbackRoute,
		})
	}

	if ingressMacroInPreset(input.ClickQueryParams) {
		if input.IngressCostConfigured {
			rows = append(rows, campaign.IntegrationHealthRow{
				Slug:    "ingress_cost_config",
				Status:  string(campaign.IntegrationHealthOK),
				Message: fmt.Sprintf("Ingress cost parsing enabled for param %s.", input.IngressCostParam),
			})
		} else {
			rows = append(rows, campaign.IntegrationHealthRow{
				Slug:     "ingress_cost_config",
				Status:   string(campaign.IntegrationHealthWarn),
				Message:  "Click preset includes an ingress cost macro but ingress_cost_config is unset.",
				FixRoute: trackingRoute,
			})
		}
	}

	return campaign.IntegrationHealthDTO{
		CampaignID: input.CampaignID.String(),
		Summary:    summarizeIntegrationHealth(rows),
		Rows:       rows,
	}
}

func missingClickJoinKeys(params map[string]string, required []string) []string {
	if len(required) == 0 {
		return nil
	}
	var missing []string
	for _, key := range required {
		if strings.TrimSpace(params[key]) == "" {
			missing = append(missing, key)
		}
	}
	return missing
}

func ingressMacroInPreset(params map[string]string) bool {
	for _, key := range []string{"cost", "cpc", "bid"} {
		val := strings.TrimSpace(params[key])
		if val == "" {
			continue
		}
		if strings.Contains(val, "{cost}") || strings.Contains(val, "{cpc}") || strings.Contains(val, "{bid}") {
			return true
		}
	}
	return false
}

func summarizeIntegrationHealth(rows []campaign.IntegrationHealthRow) string {
	summary := string(campaign.IntegrationHealthOK)
	for _, row := range rows {
		switch campaign.IntegrationHealthStatus(row.Status) {
		case campaign.IntegrationHealthFail:
			return string(campaign.IntegrationHealthFail)
		case campaign.IntegrationHealthWarn:
			summary = string(campaign.IntegrationHealthWarn)
		}
	}
	return summary
}

func CostSyncNetworkForTrafficTemplate(templateID string) string {
	return trafficTemplateCostSyncNetwork[strings.TrimSpace(templateID)]
}

func getCampaignIntegrationHealth(h *campaign.CampaignsHTTPHandlers, w http.ResponseWriter, r *http.Request) {
	campaignID, ok := h.ParseCampaignID(w, r)
	if !ok {
		return
	}
	if h.AuthorizeCampaignAccess != nil {
		if err := h.AuthorizeCampaignAccess(r, campaignID); err != nil {
			h.WriteHandlerError(w, err)
			return
		}
	}
	if h.Campaigns == nil {
		httpresponse.Error(w, http.StatusServiceUnavailable, "UNAVAILABLE", "campaign service unavailable")
		return
	}
	out, err := h.Campaigns.GetCampaignIntegrationHealth(r.Context(), campaignID)
	if err != nil {
		h.WriteHandlerError(w, err)
		return
	}
	httpresponse.JSON(w, http.StatusOK, out)
}

func GetCampaignIntegrationHealth(ctx context.Context, pool *pgxpool.Pool, fx campaign.Effects, campaignID uuid.UUID) (campaign.IntegrationHealthDTO, error) {
	if pool == nil || fx == nil {
		return campaign.IntegrationHealthDTO{}, campaign.ErrServiceUnavailable()
	}
	row, err := fx.GetCampaignRow(ctx, campaignID)
	if err != nil {
		return campaign.IntegrationHealthDTO{}, err
	}
	if err := campaign.AssertMediaBuyerCampaignAccess(ctx, row); err != nil {
		return campaign.IntegrationHealthDTO{}, err
	}

	input := IntegrationHealthInput{
		CampaignID:             campaignID,
		IntegrationSchemaBound: row.IntegrationSchemaID.Valid,
		TrafficTemplateID:      campaign.FormatOptionalText(row.TrafficTemplateID),
		TargetURL:              strings.TrimSpace(row.TargetUrl),
		ClickQueryParams:       campaign.ClickQueryParamsFromRaw(row.ClickQueryParams),
	}
	if len(row.IngressCostConfig) > 0 {
		parsed := domain.ParseIngressCostConfigJSON(row.IngressCostConfig)
		if parsed.Enabled() {
			input.IngressCostConfigured = true
			switch parsed.Param {
			case domain.IngressCostParamCost:
				input.IngressCostParam = "cost"
			case domain.IngressCostParamCPC:
				input.IngressCostParam = "cpc"
			case domain.IngressCostParamBid:
				input.IngressCostParam = "bid"
			}
		}
	}
	if input.TrafficTemplateID != "" {
		input.CostSyncNetwork = CostSyncNetworkForTrafficTemplate(input.TrafficTemplateID)
	}
	if input.CostSyncNetwork != "" {
		q := db.New(pool)
		cred, err := q.GetCostSyncCredential(ctx, db.GetCostSyncCredentialParams{
			CustomerID: row.CustomerID,
			Network:    input.CostSyncNetwork,
		})
		if err == nil && strings.TrimSpace(cred.Network) != "" {
			input.CostSyncCredentialPresent = true
		} else if err != nil && !isPgNoRows(err) {
			return campaign.IntegrationHealthDTO{}, err
		}
	}

	q := db.New(pool)
	if pb, err := q.GetPostbackConfig(ctx, domain.ToUUID(campaignID)); err == nil {
		if strings.TrimSpace(pb.UrlTemplate) != "" {
			input.PostbackConfigured = true
		}
	} else if !isPgNoRows(err) {
		return campaign.IntegrationHealthDTO{}, err
	}

	return BuildCampaignIntegrationHealth(input), nil
}

func AuditCampaignRevisionConflict(ctx context.Context, pool *pgxpool.Pool, fx campaign.Effects, campaignID uuid.UUID, expectedRevision string) {
	if pool == nil || fx == nil {
		return
	}
	row, err := fx.GetCampaignRow(ctx, campaignID)
	if err != nil {
		return
	}
	expectedRevision = strings.TrimSpace(expectedRevision)
	_ = pgx.BeginFunc(ctx, pool, func(tx pgx.Tx) error {
		q := db.New(tx)
		var uid uuid.UUID
		if u, ok := authz.GetUser(ctx); ok {
			uid = u.UserID
		}
		fx.AuditLog(ctx, q, uid, "CAMPAIGN_REVISION_CONFLICT", "campaign", &campaignID, auditRevisionConflictChange{
			ExpectedRevision: expectedRevision,
			ServerRevision:   campaign.CampaignRevision(row.UpdatedAt.Time.Format(time.RFC3339)),
		}, nil)
		return nil
	})
}

type auditRevisionConflictChange struct {
	ExpectedRevision string `json:"expected_revision"`
	ServerRevision   string `json:"server_revision"`
}

func isPgNoRows(err error) bool {
	return err != nil && (errors.Is(err, pgx.ErrNoRows) || strings.Contains(err.Error(), "no rows"))
}

func (h *IntegrationSchemaHTTPHandlers) listAffiliateStatusPresets(w http.ResponseWriter, r *http.Request) {
	out := make([]AffiliateStatusPresetDTO, 0, 8)
	for _, entry := range integrationschema.BundledIntegrationTemplateCatalog {
		if entry.Kind != integrationschema.KindStatusMapping {
			continue
		}
		_, kind, parsed, err := integrationschema.LoadBundledTemplate(entry)
		if err != nil || kind != integrationschema.KindStatusMapping {
			continue
		}
		statusSchema := parsed.(*integrationschema.StatusMappingSchema)
		statuses := make([]AffiliateStatusPresetEntryDTO, 0, len(statusSchema.StatusMap))
		for inbound, goal := range statusSchema.StatusMap {
			statuses = append(statuses, AffiliateStatusPresetEntryDTO{
				InboundStatus: inbound,
				GoalName:      goal,
			})
		}
		sort.Slice(statuses, func(i, j int) bool {
			return statuses[i].InboundStatus < statuses[j].InboundStatus
		})
		out = append(out, AffiliateStatusPresetDTO{
			Name:     entry.Name,
			Statuses: statuses,
		})
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Name < out[j].Name })
	httpresponse.JSON(w, http.StatusOK, out)
}

type TemplateCatalogService interface {
	ListBundledTemplates(ctx context.Context) []integrationschema.TemplateCatalogEntry
	ImportBundledTemplates(ctx context.Context, names []string) ([]IntegrationSchemaDTO, error)
	ApplyCampaignTemplates(ctx context.Context, campaignID uuid.UUID, req campaign.ApplyCampaignTemplatesRequest) (campaign.ApplyCampaignTemplatesResult, error)
}

type ImportTemplatesRequest struct {
	Names []string `json:"names,omitempty"`
}

type ApplyCampaignTemplatesRequest = campaign.ApplyCampaignTemplatesRequest

type ApplyCampaignTemplatesResult = campaign.ApplyCampaignTemplatesResult

func (h *IntegrationSchemaHTTPHandlers) RegisterTemplateRoutes(mux *http.ServeMux) {
	if h == nil {
		return
	}
	svc, ok := h.TemplateCatalog.(TemplateCatalogService)
	if !ok || svc == nil {
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
	mux.HandleFunc("GET /api/v1/integration/templates", limit(perm("campaigns:read", h.listBundledTemplates)))
	mux.HandleFunc("POST /api/v1/integration/templates/import", limit(perm("campaigns:write", h.importBundledTemplates)))
	mux.HandleFunc("POST /api/v1/campaigns/{id}/apply-templates", limit(perm("campaigns:write", h.applyCampaignTemplates)))
}

func (h *IntegrationSchemaHTTPHandlers) listBundledTemplates(w http.ResponseWriter, r *http.Request) {
	svc, ok := h.TemplateCatalog.(TemplateCatalogService)
	if !ok || svc == nil {
		httpresponse.Error(w, http.StatusServiceUnavailable, "UNAVAILABLE", "template catalog unavailable")
		return
	}
	out := svc.ListBundledTemplates(r.Context())
	if out == nil {
		out = []integrationschema.TemplateCatalogEntry{}
	}
	httpresponse.JSON(w, http.StatusOK, out)
}

func (h *IntegrationSchemaHTTPHandlers) importBundledTemplates(w http.ResponseWriter, r *http.Request) {
	svc, ok := h.TemplateCatalog.(TemplateCatalogService)
	if !ok || svc == nil {
		httpresponse.Error(w, http.StatusServiceUnavailable, "UNAVAILABLE", "template catalog unavailable")
		return
	}
	req, ok := coldpath.DecodeRequestOrBadRequest[ImportTemplatesRequest](w, r, coldpath.DefaultMaxBody)
	if !ok {
		return
	}
	imported, err := svc.ImportBundledTemplates(r.Context(), req.Names)
	if err != nil {
		httpresponse.Error(w, http.StatusBadRequest, "BAD_REQUEST", err.Error())
		return
	}
	if imported == nil {
		imported = []IntegrationSchemaDTO{}
	}
	httpresponse.JSON(w, http.StatusOK, imported)
}

func (h *IntegrationSchemaHTTPHandlers) applyCampaignTemplates(w http.ResponseWriter, r *http.Request) {
	svc, ok := h.TemplateCatalog.(TemplateCatalogService)
	if !ok || svc == nil {
		httpresponse.Error(w, http.StatusServiceUnavailable, "UNAVAILABLE", "template catalog unavailable")
		return
	}
	campaignID, err := uuid.Parse(r.PathValue("id"))
	if err != nil {
		httpresponse.Error(w, http.StatusBadRequest, "BAD_REQUEST", "invalid campaign id")
		return
	}
	req, ok := coldpath.DecodeRequestOrBadRequest[ApplyCampaignTemplatesRequest](w, r, coldpath.DefaultMaxBody)
	if !ok {
		return
	}
	result, err := svc.ApplyCampaignTemplates(r.Context(), campaignID, req)
	if err != nil {
		httpresponse.Error(w, http.StatusBadRequest, "BAD_REQUEST", err.Error())
		return
	}
	httpresponse.JSON(w, http.StatusOK, result)
}
