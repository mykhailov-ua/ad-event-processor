package selfserve

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"slices"
	"strings"
	"time"

	"ad-event-processor/internal/campaign"

	"ad-event-processor/internal/controlplane/authz"
	"ad-event-processor/internal/domain"
	"ad-event-processor/internal/identity"
	"ad-event-processor/pkg/coldpath"
	"ad-event-processor/pkg/httpresponse"

	"github.com/google/uuid"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

var selfServeAPIKeyAllowedScopes = []string{
	"campaigns:read",
	"campaigns:read:masked",
	"campaigns:write",
	"campaigns:pause",
	"customers:read",
}

var selfServeAPIKeyForbiddenScopes = []string{
	"audit:read",
	"ops:write",
	"blacklist:write",
	"shards:read",
	"rtb:write",
}

var selfServeAPIKeyForbiddenReportKeys = map[string]struct{}{
	"fraud-evidence-pack":  {},
	"filter-rejects":       {},
	"layer-desync-summary": {},
}

func ValidateSelfServeAPIKeyScopes(scopes []string) ([]string, error) {
	if len(scopes) == 0 {
		return []string{"campaigns:read"}, nil
	}
	if len(scopes) > 8 {
		return nil, campaign.ErrValidationf("too many scopes")
	}
	out := make([]string, 0, len(scopes))
	seen := make(map[string]struct{}, len(scopes))
	for _, scope := range scopes {
		scope = strings.TrimSpace(scope)
		if scope == "" {
			return nil, campaign.ErrValidationf("empty scope")
		}
		if slices.Contains(selfServeAPIKeyForbiddenScopes, scope) {
			return nil, campaign.ErrValidationf(fmt.Sprintf("scope %q not allowed for self-serve keys", scope))
		}
		if !slices.Contains(selfServeAPIKeyAllowedScopes, scope) {
			return nil, campaign.ErrValidationf(fmt.Sprintf("unsupported scope %q", scope))
		}
		if _, dup := seen[scope]; dup {
			continue
		}
		seen[scope] = struct{}{}
		out = append(out, scope)
	}
	return out, nil
}

func SelfServeRouteRequiresScope(routeScope string, keyScopes []string) bool {
	if len(keyScopes) == 0 {
		return true
	}
	return slices.Contains(keyScopes, routeScope)
}

func apiKeyScopesAllowReportKey(reportKey string, scopes []string) bool {
	if len(scopes) == 0 {
		return true
	}
	_, blocked := selfServeAPIKeyForbiddenReportKeys[reportKey]
	return !blocked
}

func RestrictSnapshotForAPIKeyScopes(base authz.Snapshot, scopes []string) authz.Snapshot {
	if len(scopes) == 0 {
		return base
	}
	allowed := make(map[string]struct{}, len(scopes))
	for _, scope := range scopes {
		allowed[scope] = struct{}{}
	}
	perms := make(map[string]struct{}, len(base.Permissions))
	for perm := range base.Permissions {
		if _, ok := allowed[perm]; ok {
			perms[perm] = struct{}{}
		}
	}
	return authz.Snapshot{
		Permissions: perms,
		Mask:        authz.MaskMasked,
	}
}

func DenyScopedAPIKeyOperatorReport(w http.ResponseWriter, r *http.Request, reportKey string) bool {
	user, ok := authz.GetUser(r.Context())
	if !ok || user.AuthSource != "api_key" || len(user.APIKeyScopes) == 0 {
		return false
	}
	if apiKeyScopesAllowReportKey(reportKey, user.APIKeyScopes) {
		return false
	}
	httpresponse.Error(w, http.StatusForbidden, "FORBIDDEN", "forbidden: report not allowed for api key scope")
	return true
}

type CampaignAdmin interface {
	EnforceSelfServeCreateLimits(ctx context.Context, customerID uuid.UUID, budgetMicro int64) error
	GenerateIdempotencyHash(customerID uuid.UUID, payload []byte) (string, error)
	CreateCampaign(ctx context.Context, spec campaign.CreateCampaignSpec) (uuid.UUID, error)
	PauseCampaign(ctx context.Context, campaignID uuid.UUID, reason string) error
	ResumeCampaign(ctx context.Context, campaignID uuid.UUID, reason string) error
}

type SelfServeTemplates interface {
	ListCampaignTemplates(ctx context.Context, customerID uuid.UUID, limit, offset int32) ([]campaign.CampaignTemplateDTO, int64, error)
	CreateCampaignFromTemplate(ctx context.Context, templateID, customerID uuid.UUID, name string, budgetLimit *int64, idempotencyKey string) (uuid.UUID, error)
}

type PaymentIntents = domain.PaymentAPI

type APIKeyCreator interface {
	CreateAPIKey(ctx context.Context, accessToken, name string, scopes []string) (identity.CreateAPIKeyResult, error)
}
type InvoiceLister = domain.BillingAPI

type SelfServeHTTPHandlers struct {
	Campaigns                  CampaignAdmin
	Templates                  SelfServeTemplates
	PaymentIntents             PaymentIntents
	Invoices                   InvoiceLister
	APIKeys                    APIKeyCreator
	ApplyRateLimit             func(http.HandlerFunc) http.HandlerFunc
	RequireSelfServePermission func(string, http.HandlerFunc) http.HandlerFunc
	RequireAnyPermission       func([]string, http.HandlerFunc) http.HandlerFunc
	ResolveSelfServeCustomerID func(*http.Request, *uuid.UUID) (uuid.UUID, error)
	AuthorizeCampaignAccess    func(*http.Request, uuid.UUID) error
	WriteServiceError          func(http.ResponseWriter, error)
	DefaultPaymentProvider     string
	CryptoSubProvider          string
	WriteBillingError          func(http.ResponseWriter, error)
}

type selfServeIDCreatedResponse struct {
	ID string `json:"id"`
}

type selfServePaymentIntentCreatedResponse struct {
	IntentID       string `json:"intent_id"`
	Status         string `json:"status"`
	CheckoutURL    string `json:"checkout_url"`
	ProviderRef    string `json:"provider_ref"`
	DepositAddress string `json:"deposit_address,omitempty"`
	DepositNetwork string `json:"deposit_network,omitempty"`
	DepositQRSVG   string `json:"deposit_qr_svg,omitempty"`
}

type selfServeAPIKeyCreatedResponse struct {
	ID        string   `json:"id"`
	Name      string   `json:"name"`
	RawKey    string   `json:"raw_key"`
	Scopes    []string `json:"scopes,omitempty"`
	ExpiresAt string   `json:"expires_at,omitempty"`
}

type selfServeInvoiceListResponse struct {
	Invoices []domain.Invoice `json:"invoices"`
	Total    int64            `json:"total"`
}

func (h *SelfServeHTTPHandlers) Register(mux *http.ServeMux) {
	if h == nil {
		return
	}
	limit := h.ApplyRateLimit
	perm := h.RequireSelfServePermission
	permAny := h.RequireAnyPermission
	if limit == nil {
		limit = func(next http.HandlerFunc) http.HandlerFunc { return next }
	}
	if perm == nil {
		perm = func(_ string, next http.HandlerFunc) http.HandlerFunc { return next }
	}
	if permAny == nil {
		permAny = func(perms []string, next http.HandlerFunc) http.HandlerFunc {
			if len(perms) == 0 {
				return next
			}
			return perm(perms[0], next)
		}
	}
	pausePerms := []string{"campaigns:write", "campaigns:pause"}

	mux.HandleFunc("POST /api/v1/selfserve/campaigns", limit(perm("campaigns:write", h.createCampaign)))
	mux.HandleFunc("GET /api/v1/selfserve/templates", limit(permAny([]string{"campaigns:read", "campaigns:read:masked"}, h.listTemplates)))
	mux.HandleFunc("POST /api/v1/selfserve/campaigns/{id}/pause", limit(permAny(pausePerms, h.pauseCampaign)))
	mux.HandleFunc("POST /api/v1/selfserve/campaigns/{id}/resume", limit(permAny(pausePerms, h.resumeCampaign)))
	mux.HandleFunc("POST /api/v1/selfserve/payment-intents", limit(perm("customers:read", h.createPaymentIntent)))
	mux.HandleFunc("GET /api/v1/selfserve/invoices", limit(perm("customers:read", h.listInvoices)))
	mux.HandleFunc("POST /api/v1/selfserve/api-keys", limit(perm("campaigns:write", h.createAPIKey)))
}

func (h *SelfServeHTTPHandlers) createCampaign(w http.ResponseWriter, r *http.Request) {
	body, err := coldpath.ReadLimitedBody(w, r, coldpath.DefaultMaxBody)
	if err != nil {
		httpresponse.Error(w, http.StatusBadRequest, "BAD_REQUEST", "failed to read request body")
		return
	}

	req, err := coldpath.DecodeBody[struct {
		CustomerID       *uuid.UUID `json:"customer_id,omitempty"`
		TemplateID       string     `json:"template_id"`
		Name             string     `json:"name"`
		BudgetLimitMicro *int64     `json:"budget_limit_micro"`
	}](body)
	if err != nil {
		httpresponse.Error(w, http.StatusBadRequest, "BAD_REQUEST", "invalid request body")
		return
	}
	templateID, err := uuid.Parse(strings.TrimSpace(req.TemplateID))
	if err != nil || templateID == uuid.Nil {
		httpresponse.Error(w, http.StatusBadRequest, "BAD_REQUEST", "template_id is required")
		return
	}

	customerID, err := h.resolveCustomerID(r, req.CustomerID)
	if err != nil {
		h.WriteHandlerError(w, err)
		return
	}

	idempotencyKey := r.Header.Get("Idempotency-Key")
	if idempotencyKey == "" {
		httpresponse.Error(w, http.StatusBadRequest, "BAD_REQUEST", "Idempotency-Key header is required")
		return
	}

	if h.Templates == nil || h.Campaigns == nil {
		httpresponse.Error(w, http.StatusServiceUnavailable, "UNAVAILABLE", "campaign service not configured")
		return
	}

	budgetMicro := int64(0)
	if req.BudgetLimitMicro != nil {
		if *req.BudgetLimitMicro <= 0 {
			httpresponse.Error(w, http.StatusBadRequest, "BAD_REQUEST", "budget must be positive")
			return
		}
		budgetMicro = *req.BudgetLimitMicro
		if err := h.Campaigns.EnforceSelfServeCreateLimits(r.Context(), customerID, budgetMicro); err != nil {
			h.WriteHandlerError(w, err)
			return
		}
	}

	hash, err := h.Campaigns.GenerateIdempotencyHash(customerID, append(body, []byte(idempotencyKey)...))
	if err != nil {
		h.WriteHandlerError(w, err, slog.String("customer_id", customerID.String()))
		return
	}

	var budgetOverride *int64
	if req.BudgetLimitMicro != nil {
		budgetOverride = req.BudgetLimitMicro
	}

	id, err := h.Templates.CreateCampaignFromTemplate(
		r.Context(), templateID, customerID, strings.TrimSpace(req.Name), budgetOverride, hash,
	)
	if err != nil {
		h.WriteHandlerError(w, err, slog.String("customer_id", customerID.String()))
		return
	}
	httpresponse.JSON(w, http.StatusCreated, selfServeIDCreatedResponse{ID: id.String()})
}

func (h *SelfServeHTTPHandlers) listTemplates(w http.ResponseWriter, r *http.Request) {
	if h.Templates == nil {
		httpresponse.Error(w, http.StatusServiceUnavailable, "UNAVAILABLE", "template service not configured")
		return
	}
	var bodyCustomerID *uuid.UUID
	if raw := r.URL.Query().Get("customer_id"); raw != "" {
		parsed, err := uuid.Parse(raw)
		if err != nil {
			httpresponse.Error(w, http.StatusBadRequest, "BAD_REQUEST", "invalid customer_id")
			return
		}
		bodyCustomerID = &parsed
	}
	customerID, err := h.resolveCustomerID(r, bodyCustomerID)
	if err != nil {
		h.WriteHandlerError(w, err)
		return
	}
	limit, offset := coldpath.ParseAPIPagination(r)
	items, total, err := h.Templates.ListCampaignTemplates(r.Context(), customerID, limit, offset)
	if err != nil {
		h.WriteHandlerError(w, err)
		return
	}
	httpresponse.JSON(w, http.StatusOK, campaign.CampaignTemplateListResponse{Items: items, Total: total})
}

func (h *SelfServeHTTPHandlers) pauseCampaign(w http.ResponseWriter, r *http.Request) {
	campaignID, err := uuid.Parse(r.PathValue("id"))
	if err != nil {
		httpresponse.Error(w, http.StatusBadRequest, "BAD_REQUEST", "invalid campaign id")
		return
	}
	if err := h.authorizeCampaign(r, campaignID); err != nil {
		h.WriteHandlerError(w, err)
		return
	}
	req, err := coldpath.DecodeRequest[struct {
		Reason string `json:"reason"`
	}](w, r, coldpath.DefaultMaxBody)
	if err != nil {
		slog.Warn("failed to decode pause campaign request", "error", err)
	}
	if h.Campaigns == nil {
		httpresponse.Error(w, http.StatusServiceUnavailable, "UNAVAILABLE", "campaign service not configured")
		return
	}
	if err := h.Campaigns.PauseCampaign(r.Context(), campaignID, req.Reason); err != nil {
		h.WriteHandlerError(w, err, slog.String("campaign_id", campaignID.String()))
		return
	}
	w.WriteHeader(http.StatusAccepted)
}

func (h *SelfServeHTTPHandlers) resumeCampaign(w http.ResponseWriter, r *http.Request) {
	campaignID, err := uuid.Parse(r.PathValue("id"))
	if err != nil {
		httpresponse.Error(w, http.StatusBadRequest, "BAD_REQUEST", "invalid campaign id")
		return
	}
	if err := h.authorizeCampaign(r, campaignID); err != nil {
		h.WriteHandlerError(w, err)
		return
	}
	req, err := coldpath.DecodeRequest[struct {
		Reason string `json:"reason"`
	}](w, r, coldpath.DefaultMaxBody)
	if err != nil {
		slog.Warn("failed to decode resume campaign request", "error", err)
	}
	if h.Campaigns == nil {
		httpresponse.Error(w, http.StatusServiceUnavailable, "UNAVAILABLE", "campaign service not configured")
		return
	}
	if err := h.Campaigns.ResumeCampaign(r.Context(), campaignID, req.Reason); err != nil {
		h.WriteHandlerError(w, err, slog.String("campaign_id", campaignID.String()))
		return
	}
	w.WriteHeader(http.StatusAccepted)
}

func (h *SelfServeHTTPHandlers) createPaymentIntent(w http.ResponseWriter, r *http.Request) {
	body, err := coldpath.ReadLimitedBody(w, r, coldpath.SelfServePaymentIntentMaxBody)
	if err != nil {
		httpresponse.Error(w, http.StatusBadRequest, "BAD_REQUEST", "invalid request body")
		return
	}

	req, err := coldpath.DecodeBody[struct {
		CustomerID  *uuid.UUID `json:"customer_id,omitempty"`
		AmountMicro int64      `json:"amount_micro"`
		Currency    string     `json:"currency"`
	}](body)
	if err != nil {
		httpresponse.Error(w, http.StatusBadRequest, "BAD_REQUEST", "invalid request body")
		return
	}
	if req.AmountMicro <= 0 {
		httpresponse.Error(w, http.StatusBadRequest, "BAD_REQUEST", "amount_micro must be greater than zero")
		return
	}

	idempotencyKey := r.Header.Get("Idempotency-Key")
	if idempotencyKey == "" {
		httpresponse.Error(w, http.StatusBadRequest, "BAD_REQUEST", "Idempotency-Key header is required")
		return
	}

	if h.PaymentIntents == nil {
		httpresponse.Error(w, http.StatusServiceUnavailable, "PAYMENT_UNAVAILABLE", "payment service not configured")
		return
	}

	customerID, err := h.resolveCustomerID(r, req.CustomerID)
	if err != nil {
		h.WriteHandlerError(w, err)
		return
	}

	currency := req.Currency
	if currency == "" {
		currency = "USD"
	}

	meta := map[string]string{
		"customer_id": customerID.String(),
		"source":      "selfserve",
	}
	if p := strings.TrimSpace(h.DefaultPaymentProvider); p != "" {
		meta["provider"] = p
	}
	if cp := strings.TrimSpace(h.CryptoSubProvider); cp != "" {
		meta["crypto_provider"] = cp
	}

	resp, err := h.PaymentIntents.CreatePaymentIntent(r.Context(), customerID.String(), req.AmountMicro, currency, idempotencyKey, meta)
	if err != nil {
		if st, ok := status.FromError(err); ok {
			switch st.Code() {
			case codes.InvalidArgument:
				httpresponse.Error(w, http.StatusBadRequest, "BAD_REQUEST", st.Message())
				return
			case codes.AlreadyExists:
				httpresponse.Error(w, http.StatusConflict, "CONFLICT", st.Message())
				return
			case codes.FailedPrecondition:
				httpresponse.Error(w, http.StatusServiceUnavailable, "PAYMENT_UNAVAILABLE", st.Message())
				return
			}
		}
		httpresponse.Error(w, http.StatusInternalServerError, "INTERNAL", "failed to create payment intent")
		return
	}
	if resp == nil {
		httpresponse.Error(w, http.StatusInternalServerError, "INTERNAL", "failed to create payment intent")
		return
	}

	httpresponse.JSON(w, http.StatusOK, selfServePaymentIntentCreatedResponse{
		IntentID:       resp.IntentID,
		Status:         resp.Status,
		CheckoutURL:    resp.CheckoutURL,
		ProviderRef:    resp.ProviderRef,
		DepositAddress: resp.DepositAddress,
		DepositNetwork: resp.DepositNetwork,
		DepositQRSVG:   resp.DepositQRSVG,
	})
}

func (h *SelfServeHTTPHandlers) listInvoices(w http.ResponseWriter, r *http.Request) {
	if h.Invoices == nil {
		httpresponse.Error(w, http.StatusServiceUnavailable, "BILLING_UNAVAILABLE", "billing service not configured")
		return
	}

	var bodyCustomerID *uuid.UUID
	if raw := r.URL.Query().Get("customer_id"); raw != "" {
		parsed, err := uuid.Parse(raw)
		if err != nil {
			httpresponse.Error(w, http.StatusBadRequest, "BAD_REQUEST", "invalid customer_id")
			return
		}
		bodyCustomerID = &parsed
	}

	customerID, err := h.resolveCustomerID(r, bodyCustomerID)
	if err != nil {
		h.WriteHandlerError(w, err)
		return
	}

	limit, offset := coldpath.ParseAPIPaginationWith(r, 20, 100)

	resp, err := h.Invoices.ListInvoices(r.Context(), customerID.String(), limit, offset)
	if err != nil {
		if h.WriteBillingError != nil {
			h.WriteBillingError(w, err)
		} else {
			httpresponse.Error(w, http.StatusInternalServerError, "INTERNAL", "failed to list invoices")
		}
		return
	}

	httpresponse.JSON(w, http.StatusOK, selfServeInvoiceListResponse{
		Invoices: resp.Invoices,
		Total:    resp.Total,
	})
}

func (h *SelfServeHTTPHandlers) createAPIKey(w http.ResponseWriter, r *http.Request) {
	if h.APIKeys == nil {
		httpresponse.Error(w, http.StatusServiceUnavailable, "AUTH_UNAVAILABLE", "auth service not configured")
		return
	}

	req, err := coldpath.DecodeRequest[struct {
		Name   string   `json:"name"`
		Scopes []string `json:"scopes"`
	}](w, r, coldpath.DefaultMaxBody)
	if err != nil {
		return
	}
	if strings.TrimSpace(req.Name) == "" {
		httpresponse.Error(w, http.StatusBadRequest, "BAD_REQUEST", "name is required")
		return
	}
	scopes, err := ValidateSelfServeAPIKeyScopes(req.Scopes)
	if err != nil {
		h.WriteHandlerError(w, err)
		return
	}

	cookie, err := r.Cookie("accessToken")
	if err != nil || cookie.Value == "" {
		httpresponse.Error(w, http.StatusUnauthorized, "UNAUTHORIZED", "session required to create api keys")
		return
	}

	resp, err := h.APIKeys.CreateAPIKey(r.Context(), cookie.Value, req.Name, scopes)
	if err != nil {
		if st, ok := status.FromError(err); ok {
			switch st.Code() {
			case codes.InvalidArgument:
				httpresponse.Error(w, http.StatusBadRequest, "BAD_REQUEST", st.Message())
				return
			case codes.Unauthenticated:
				httpresponse.Error(w, http.StatusUnauthorized, "UNAUTHORIZED", st.Message())
				return
			}
		}
		httpresponse.Error(w, http.StatusInternalServerError, "INTERNAL", "failed to create api key")
		return
	}

	out := selfServeAPIKeyCreatedResponse{
		ID:     resp.ID,
		Name:   resp.Name,
		RawKey: resp.RawKey,
		Scopes: resp.Scopes,
	}
	if resp.ExpiresAt != nil {
		out.ExpiresAt = resp.ExpiresAt.UTC().Format(time.RFC3339)
	}
	httpresponse.JSON(w, http.StatusCreated, out)
}

func (h *SelfServeHTTPHandlers) resolveCustomerID(r *http.Request, bodyCustomerID *uuid.UUID) (uuid.UUID, error) {
	if h.ResolveSelfServeCustomerID == nil {
		return uuid.Nil, campaign.ErrForbidden
	}
	return h.ResolveSelfServeCustomerID(r, bodyCustomerID)
}

func (h *SelfServeHTTPHandlers) authorizeCampaign(r *http.Request, campaignID uuid.UUID) error {
	if h.AuthorizeCampaignAccess == nil {
		return campaign.ErrForbidden
	}
	return h.AuthorizeCampaignAccess(r, campaignID)
}

func (h *SelfServeHTTPHandlers) WriteHandlerError(w http.ResponseWriter, err error, logAttrs ...any) {
	h.writeServiceError(w, err, logAttrs...)
}

func (h *SelfServeHTTPHandlers) writeServiceError(w http.ResponseWriter, err error, logAttrs ...any) {
	if h.WriteServiceError != nil {
		h.WriteServiceError(w, err)
		return
	}
	_ = logAttrs
	httpresponse.Error(w, http.StatusInternalServerError, "INTERNAL", "internal error")
}

type SelfServeTemplateHost interface {
	ListCampaignTemplates(ctx context.Context, customerID uuid.UUID, limit, offset int32) ([]campaign.CampaignTemplateDTO, int64, error)
	CreateCampaignFromTemplate(ctx context.Context, templateID, customerID uuid.UUID, name string, budgetLimit *int64, idempotencyKey string) (uuid.UUID, error)
}

type selfServeTemplatesAdapter struct {
	host SelfServeTemplateHost
}

func NewSelfServeTemplatesAdapter(host SelfServeTemplateHost) SelfServeTemplates {
	if host == nil {
		return nil
	}
	return selfServeTemplatesAdapter{host: host}
}

func (a selfServeTemplatesAdapter) ListCampaignTemplates(
	ctx context.Context,
	customerID uuid.UUID,
	limit, offset int32,
) ([]campaign.CampaignTemplateDTO, int64, error) {
	return a.host.ListCampaignTemplates(ctx, customerID, limit, offset)
}

func (a selfServeTemplatesAdapter) CreateCampaignFromTemplate(
	ctx context.Context,
	templateID, customerID uuid.UUID,
	name string,
	budgetLimit *int64,
	idempotencyKey string,
) (uuid.UUID, error) {
	return a.host.CreateCampaignFromTemplate(ctx, templateID, customerID, name, budgetLimit, idempotencyKey)
}
