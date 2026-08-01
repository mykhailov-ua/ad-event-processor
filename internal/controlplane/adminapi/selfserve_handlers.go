package adminapi

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"strconv"
	"strings"
	"time"

	"espx/internal/billing"
	"espx/internal/payment"
	"espx/pkg/coldpath"
	"espx/pkg/httpresponse"
	"espx/pkg/money"

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
	IdempotencyKey   string
}

type CampaignAdmin interface {
	EnforceSelfServeCreateLimits(ctx context.Context, customerID uuid.UUID, budgetMicro int64) error
	GenerateIdempotencyHash(customerID uuid.UUID, payload []byte) (string, error)
	CreateCampaign(ctx context.Context, spec CreateCampaignInput) (uuid.UUID, error)
	PauseCampaign(ctx context.Context, campaignID uuid.UUID, reason string) error
	ResumeCampaign(ctx context.Context, campaignID uuid.UUID, reason string) error
}

type PaymentIntents = payment.PaymentAPI

type APIKeyResult struct {
	ID         string
	Name       string
	RawKey     string
	ExpiresAt  string
	HasExpires bool
}

type APIKeyCreator interface {
	CreateAPIKey(ctx context.Context, accessToken, name string) (APIKeyResult, error)
}

type InvoiceLister = billing.BillingAPI

func parseMoneyMicro(micro *int64, legacy float64, hasLegacy bool, field string) (int64, error) {
	if micro != nil {
		if *micro < 0 {
			return 0, fmt.Errorf("invalid %s", field)
		}
		return *micro, nil
	}
	if hasLegacy {
		v, err := money.LegacyFloatToMicro(legacy)
		if err != nil {
			return 0, fmt.Errorf("invalid %s", field)
		}
		return v, nil
	}
	return 0, nil
}

func parseBudgetMicro(micro *int64, legacy float64, hasLegacy bool) (int64, error) {
	if micro != nil {
		if *micro <= 0 {
			return 0, fmt.Errorf("budget must be positive")
		}
		return *micro, nil
	}
	if hasLegacy {
		v, err := money.LegacyFloatToMicro(legacy)
		if err != nil || v <= 0 {
			return 0, fmt.Errorf("budget must be positive")
		}
		return v, nil
	}
	return 0, fmt.Errorf("budget is required")
}

type SelfServeHTTPHandlers struct {
	Campaigns                  CampaignAdmin
	PaymentIntents             PaymentIntents
	Invoices                   InvoiceLister
	APIKeys                    APIKeyCreator
	ApplyRateLimit             func(http.HandlerFunc) http.HandlerFunc
	RequireSelfServePermission func(string, http.HandlerFunc) http.HandlerFunc
	ResolveSelfServeCustomerID func(*http.Request, *uuid.UUID) (uuid.UUID, error)
	AuthorizeCampaignAccess    func(*http.Request, uuid.UUID) error
	WriteServiceError          func(http.ResponseWriter, error)
}

func (selfServe *SelfServeHTTPHandlers) Register(mux *http.ServeMux) {
	if selfServe == nil {
		return
	}
	limit := selfServe.ApplyRateLimit
	perm := selfServe.RequireSelfServePermission
	if limit == nil {
		limit = func(next http.HandlerFunc) http.HandlerFunc { return next }
	}
	if perm == nil {
		perm = func(_ string, next http.HandlerFunc) http.HandlerFunc { return next }
	}

	mux.HandleFunc("POST /api/v1/selfserve/campaigns", limit(perm("campaigns:write", selfServe.createCampaign)))
	mux.HandleFunc("POST /api/v1/selfserve/campaigns/{id}/pause", limit(perm("campaigns:write", selfServe.pauseCampaign)))
	mux.HandleFunc("POST /api/v1/selfserve/campaigns/{id}/resume", limit(perm("campaigns:write", selfServe.resumeCampaign)))
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
		BrandID          *uuid.UUID `json:"brand_id,omitempty"`
		Name             string     `json:"name"`
		BudgetLimitMicro *int64     `json:"budget_limit_micro"`
		BudgetLimit      *float64   `json:"budget_limit"`
		PacingMode       string     `json:"pacing_mode"`
		DailyBudgetMicro *int64     `json:"daily_budget_micro"`
		DailyBudget      *float64   `json:"daily_budget"`
		Timezone         string     `json:"timezone"`
		FreqLimit        int32      `json:"freq_limit"`
		FreqWindow       int32      `json:"freq_window"`
		TargetCountries  []string   `json:"target_countries"`
		StartAt          *time.Time `json:"start_at,omitempty"`
		EndAt            *time.Time `json:"end_at,omitempty"`
		DaypartHours     []int16    `json:"daypart_hours,omitempty"`
	}](body)
	if err != nil {
		httpresponse.Error(w, http.StatusBadRequest, "BAD_REQUEST", "invalid request body")
		return
	}
	if req.Name == "" {
		httpresponse.Error(w, http.StatusBadRequest, "BAD_REQUEST", "name is required")
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

	pacing := "ASAP"
	if req.PacingMode == "EVEN" {
		pacing = "EVEN"
	}
	if req.Timezone == "" {
		req.Timezone = "UTC"
	}
	if req.FreqWindow == 0 {
		req.FreqWindow = 86400
	}

	budgetLegacy := 0.0
	hasBudgetLegacy := req.BudgetLimit != nil
	if hasBudgetLegacy {
		budgetLegacy = *req.BudgetLimit
	}
	budgetLimitMicro, err := parseBudgetMicro(req.BudgetLimitMicro, budgetLegacy, hasBudgetLegacy)
	if err != nil {
		httpresponse.Error(w, http.StatusBadRequest, "BAD_REQUEST", err.Error())
		return
	}

	dailyLegacy := 0.0
	hasDailyLegacy := req.DailyBudget != nil
	if hasDailyLegacy {
		dailyLegacy = *req.DailyBudget
	}
	dailyBudgetMicro, err := parseMoneyMicro(req.DailyBudgetMicro, dailyLegacy, hasDailyLegacy, "daily_budget")
	if err != nil {
		httpresponse.Error(w, http.StatusBadRequest, "BAD_REQUEST", err.Error())
		return
	}

	if selfServe.Campaigns == nil {
		httpresponse.Error(w, http.StatusServiceUnavailable, "UNAVAILABLE", "campaign service not configured")
		return
	}
	if err := selfServe.Campaigns.EnforceSelfServeCreateLimits(r.Context(), customerID, budgetLimitMicro); err != nil {
		selfServe.writeServiceError(w, err)
		return
	}

	hash, err := selfServe.Campaigns.GenerateIdempotencyHash(customerID, append(body, []byte(idempotencyKey)...))
	if err != nil {
		selfServe.writeServiceError(w, err, slog.String("customer_id", customerID.String()))
		return
	}

	id, err := selfServe.Campaigns.CreateCampaign(r.Context(), CreateCampaignInput{
		CustomerID:       customerID,
		BrandID:          req.BrandID,
		Name:             req.Name,
		BudgetLimitMicro: budgetLimitMicro,
		PacingMode:       pacing,
		DailyBudgetMicro: dailyBudgetMicro,
		Timezone:         req.Timezone,
		FreqLimit:        req.FreqLimit,
		FreqWindow:       req.FreqWindow,
		TargetCountries:  req.TargetCountries,
		StartAt:          req.StartAt,
		EndAt:            req.EndAt,
		DaypartHours:     req.DaypartHours,
		IdempotencyKey:   hash,
	})
	if err != nil {
		selfServe.writeServiceError(w, err, slog.String("customer_id", customerID.String()))
		return
	}
	httpresponse.JSON(w, http.StatusCreated, map[string]any{"id": id})
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
	body, err := coldpath.ReadLimitedBody(w, r, 16*1024)
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

	httpresponse.JSON(w, http.StatusOK, map[string]any{
		"intent_id":    resp.IntentID,
		"status":       resp.Status,
		"checkout_url": resp.CheckoutURL,
		"provider_ref": resp.ProviderRef,
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

	limit := int32(20)
	offset := int32(0)
	if v := r.URL.Query().Get("limit"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 {
			limit = int32(n)
		}
	}
	if v := r.URL.Query().Get("offset"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n >= 0 {
			offset = int32(n)
		}
	}
	if limit > 100 {
		limit = 100
	}

	resp, err := selfServe.Invoices.ListInvoices(r.Context(), customerID.String(), limit, offset)
	if err != nil {
		WriteBillingError(w, err)
		return
	}

	invoices := make([]map[string]any, 0, len(resp.Invoices))
	for i := range resp.Invoices {
		invoices = append(invoices, InvoiceToJSON(&resp.Invoices[i]))
	}
	httpresponse.JSON(w, http.StatusOK, map[string]any{
		"invoices": invoices,
		"total":    resp.Total,
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

	out := map[string]any{
		"id":      resp.ID,
		"name":    resp.Name,
		"raw_key": resp.RawKey,
	}
	if resp.HasExpires {
		out["expires_at"] = resp.ExpiresAt
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
