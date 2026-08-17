package adminapi

import (
	"context"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"github.com/bidshard/ad-event-processor/internal/domain"
	"github.com/bidshard/ad-event-processor/internal/identity"
	"github.com/bidshard/ad-event-processor/pkg/coldpath"
	"github.com/bidshard/ad-event-processor/pkg/httpresponse"

	"github.com/google/uuid"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type CreateCampaignInput struct {
	CustomerID       uuid.UUID
	BrandID          *uuid.UUID
	Name             string
	BudgetLimitMicro int64
	PacingMode       string
	DailyBudgetMicro int64
	Timezone         string
	FreqLimit        int32
	FreqWindow       int32
	TargetCountries  []string
	StartAt          *time.Time
	EndAt            *time.Time
	DaypartHours     []int16
	TemplateID       *uuid.UUID
	IdempotencyKey   string
}

type CampaignAdmin interface {
	EnforceSelfServeCreateLimits(ctx context.Context, customerID uuid.UUID, budgetMicro int64) error
	GenerateIdempotencyHash(customerID uuid.UUID, payload []byte) (string, error)
	CreateCampaign(ctx context.Context, spec CreateCampaignInput) (uuid.UUID, error)
	PauseCampaign(ctx context.Context, campaignID uuid.UUID, reason string) error
	ResumeCampaign(ctx context.Context, campaignID uuid.UUID, reason string) error
}

type SelfServeTemplates interface {
	ListCampaignTemplates(ctx context.Context, customerID uuid.UUID, limit, offset int32) ([]CampaignTemplateDTO, int64, error)
	CreateCampaignFromTemplate(ctx context.Context, templateID, customerID uuid.UUID, name string, budgetLimit *int64, idempotencyKey string) (uuid.UUID, error)
}

type PaymentIntents = domain.PaymentAPI

type APIKeyCreator interface {
	CreateAPIKey(ctx context.Context, accessToken, name string) (identity.CreateAPIKeyResult, error)
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
}

func (selfServe *SelfServeHTTPHandlers) Register(mux *http.ServeMux) {
	if selfServe == nil {
		return
	}
	limit := selfServe.ApplyRateLimit
	perm := selfServe.RequireSelfServePermission
	permAny := selfServe.RequireAnyPermission
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

	mux.HandleFunc("POST /api/v1/selfserve/campaigns", limit(perm("campaigns:write", selfServe.createCampaign)))
	mux.HandleFunc("GET /api/v1/selfserve/templates", limit(permAny([]string{"campaigns:read", "campaigns:read:masked"}, selfServe.listTemplates)))
	mux.HandleFunc("POST /api/v1/selfserve/campaigns/{id}/pause", limit(permAny(pausePerms, selfServe.pauseCampaign)))
	mux.HandleFunc("POST /api/v1/selfserve/campaigns/{id}/resume", limit(permAny(pausePerms, selfServe.resumeCampaign)))
	mux.HandleFunc("POST /api/v1/selfserve/payment-intents", limit(perm("customers:read", selfServe.createPaymentIntent)))
	mux.HandleFunc("GET /api/v1/selfserve/invoices", limit(perm("customers:read", selfServe.listInvoices)))
	mux.HandleFunc("POST /api/v1/selfserve/api-keys", limit(perm("campaigns:write", selfServe.createAPIKey)))
}

func (selfServe *SelfServeHTTPHandlers) createCampaign(w http.ResponseWriter, r *http.Request) {
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

	customerID, err := selfServe.resolveCustomerID(r, req.CustomerID)
	if err != nil {
		selfServe.writeServiceError(w, err)
		return
	}

	idempotencyKey := r.Header.Get("Idempotency-Key")
	if idempotencyKey == "" {
		httpresponse.Error(w, http.StatusBadRequest, "BAD_REQUEST", "Idempotency-Key header is required")
		return
	}

	if selfServe.Templates == nil || selfServe.Campaigns == nil {
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
		if err := selfServe.Campaigns.EnforceSelfServeCreateLimits(r.Context(), customerID, budgetMicro); err != nil {
			selfServe.writeServiceError(w, err)
			return
		}
	}

	hash, err := selfServe.Campaigns.GenerateIdempotencyHash(customerID, append(body, []byte(idempotencyKey)...))
	if err != nil {
		selfServe.writeServiceError(w, err, slog.String("customer_id", customerID.String()))
		return
	}

	var budgetOverride *int64
	if req.BudgetLimitMicro != nil {
		budgetOverride = req.BudgetLimitMicro
	}

	id, err := selfServe.Templates.CreateCampaignFromTemplate(
		r.Context(), templateID, customerID, strings.TrimSpace(req.Name), budgetOverride, hash,
	)
	if err != nil {
		selfServe.writeServiceError(w, err, slog.String("customer_id", customerID.String()))
		return
	}
	httpresponse.JSON(w, http.StatusCreated, IDCreatedResponse{ID: id.String()})
}

func (selfServe *SelfServeHTTPHandlers) listTemplates(w http.ResponseWriter, r *http.Request) {
	if selfServe.Templates == nil {
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
	customerID, err := selfServe.resolveCustomerID(r, bodyCustomerID)
	if err != nil {
		selfServe.writeServiceError(w, err)
		return
	}
	limit, offset := coldpath.ParseAPIPagination(r)
	items, total, err := selfServe.Templates.ListCampaignTemplates(r.Context(), customerID, limit, offset)
	if err != nil {
		selfServe.writeServiceError(w, err)
		return
	}
	httpresponse.JSON(w, http.StatusOK, CampaignTemplateListResponse{Items: items, Total: total})
}

func (selfServe *SelfServeHTTPHandlers) pauseCampaign(w http.ResponseWriter, r *http.Request) {
	campaignID, err := uuid.Parse(r.PathValue("id"))
	if err != nil {
		httpresponse.Error(w, http.StatusBadRequest, "BAD_REQUEST", "invalid campaign id")
		return
	}
	if err := selfServe.authorizeCampaign(r, campaignID); err != nil {
		selfServe.writeServiceError(w, err)
		return
	}
	req, err := coldpath.DecodeRequest[struct {
		Reason string `json:"reason"`
	}](w, r, coldpath.DefaultMaxBody)
	if err != nil {
		slog.Warn("failed to decode pause campaign request", "error", err)
	}
	if selfServe.Campaigns == nil {
		httpresponse.Error(w, http.StatusServiceUnavailable, "UNAVAILABLE", "campaign service not configured")
		return
	}
	if err := selfServe.Campaigns.PauseCampaign(r.Context(), campaignID, req.Reason); err != nil {
		selfServe.writeServiceError(w, err, slog.String("campaign_id", campaignID.String()))
		return
	}
	w.WriteHeader(http.StatusAccepted)
}

func (selfServe *SelfServeHTTPHandlers) resumeCampaign(w http.ResponseWriter, r *http.Request) {
	campaignID, err := uuid.Parse(r.PathValue("id"))
	if err != nil {
		httpresponse.Error(w, http.StatusBadRequest, "BAD_REQUEST", "invalid campaign id")
		return
	}
	if err := selfServe.authorizeCampaign(r, campaignID); err != nil {
		selfServe.writeServiceError(w, err)
		return
	}
	req, err := coldpath.DecodeRequest[struct {
		Reason string `json:"reason"`
	}](w, r, coldpath.DefaultMaxBody)
	if err != nil {
		slog.Warn("failed to decode resume campaign request", "error", err)
	}
	if selfServe.Campaigns == nil {
		httpresponse.Error(w, http.StatusServiceUnavailable, "UNAVAILABLE", "campaign service not configured")
		return
	}
	if err := selfServe.Campaigns.ResumeCampaign(r.Context(), campaignID, req.Reason); err != nil {
		selfServe.writeServiceError(w, err, slog.String("campaign_id", campaignID.String()))
		return
	}
	w.WriteHeader(http.StatusAccepted)
}

func (selfServe *SelfServeHTTPHandlers) createPaymentIntent(w http.ResponseWriter, r *http.Request) {
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

	if selfServe.PaymentIntents == nil {
		httpresponse.Error(w, http.StatusServiceUnavailable, "PAYMENT_UNAVAILABLE", "payment service not configured")
		return
	}

	customerID, err := selfServe.resolveCustomerID(r, req.CustomerID)
	if err != nil {
		selfServe.writeServiceError(w, err)
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
	if p := strings.TrimSpace(selfServe.DefaultPaymentProvider); p != "" {
		meta["provider"] = p
	}
	if cp := strings.TrimSpace(selfServe.CryptoSubProvider); cp != "" {
		meta["crypto_provider"] = cp
	}

	resp, err := selfServe.PaymentIntents.CreatePaymentIntent(r.Context(), customerID.String(), req.AmountMicro, currency, idempotencyKey, meta)
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

	httpresponse.JSON(w, http.StatusOK, PaymentIntentCreatedResponse{
		IntentID:       resp.IntentID,
		Status:         resp.Status,
		CheckoutURL:    resp.CheckoutURL,
		ProviderRef:    resp.ProviderRef,
		DepositAddress: resp.DepositAddress,
		DepositNetwork: resp.DepositNetwork,
		DepositQRSVG:   resp.DepositQRSVG,
	})
}

func (selfServe *SelfServeHTTPHandlers) listInvoices(w http.ResponseWriter, r *http.Request) {
	if selfServe.Invoices == nil {
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

	customerID, err := selfServe.resolveCustomerID(r, bodyCustomerID)
	if err != nil {
		selfServe.writeServiceError(w, err)
		return
	}

	limit, offset := coldpath.ParseAPIPaginationWith(r, 20, 100)

	resp, err := selfServe.Invoices.ListInvoices(r.Context(), customerID.String(), limit, offset)
	if err != nil {
		WriteBillingError(w, err)
		return
	}

	httpresponse.JSON(w, http.StatusOK, SelfServeInvoiceListResponse{
		Invoices: resp.Invoices,
		Total:    resp.Total,
	})
}

func (selfServe *SelfServeHTTPHandlers) createAPIKey(w http.ResponseWriter, r *http.Request) {
	if selfServe.APIKeys == nil {
		httpresponse.Error(w, http.StatusServiceUnavailable, "AUTH_UNAVAILABLE", "auth service not configured")
		return
	}

	req, err := coldpath.DecodeRequest[struct {
		Name string `json:"name"`
	}](w, r, coldpath.DefaultMaxBody)
	if err != nil {
		return
	}
	if strings.TrimSpace(req.Name) == "" {
		httpresponse.Error(w, http.StatusBadRequest, "BAD_REQUEST", "name is required")
		return
	}

	cookie, err := r.Cookie("accessToken")
	if err != nil || cookie.Value == "" {
		httpresponse.Error(w, http.StatusUnauthorized, "UNAUTHORIZED", "session required to create api keys")
		return
	}

	resp, err := selfServe.APIKeys.CreateAPIKey(r.Context(), cookie.Value, req.Name)
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

	out := APIKeyCreatedResponse{
		ID:     resp.ID,
		Name:   resp.Name,
		RawKey: resp.RawKey,
	}
	if resp.ExpiresAt != nil {
		out.ExpiresAt = resp.ExpiresAt.UTC().Format(time.RFC3339)
	}
	httpresponse.JSON(w, http.StatusCreated, out)
}

func (selfServe *SelfServeHTTPHandlers) resolveCustomerID(r *http.Request, bodyCustomerID *uuid.UUID) (uuid.UUID, error) {
	if selfServe.ResolveSelfServeCustomerID == nil {
		return uuid.Nil, ErrForbidden
	}
	return selfServe.ResolveSelfServeCustomerID(r, bodyCustomerID)
}

func (selfServe *SelfServeHTTPHandlers) authorizeCampaign(r *http.Request, campaignID uuid.UUID) error {
	if selfServe.AuthorizeCampaignAccess == nil {
		return ErrForbidden
	}
	return selfServe.AuthorizeCampaignAccess(r, campaignID)
}

func (selfServe *SelfServeHTTPHandlers) writeServiceError(w http.ResponseWriter, err error, logAttrs ...any) {
	if selfServe.WriteServiceError != nil {
		selfServe.WriteServiceError(w, err)
		return
	}
	_ = logAttrs
	httpresponse.Error(w, http.StatusInternalServerError, "INTERNAL", "internal error")
}
