package controlplane

import (
	"context"
	"errors"
	"net/http"
	"strconv"

	"ad-event-processor/internal/billingadmin"
	"ad-event-processor/internal/campaign"
	"ad-event-processor/internal/config"
	ctrlhttp "ad-event-processor/internal/control/http"
	"ad-event-processor/internal/controlplane/authz"
	"ad-event-processor/internal/dedup"
	db "ad-event-processor/internal/domain/db"
	"ad-event-processor/internal/identity"
	"ad-event-processor/internal/ledger"
	"ad-event-processor/internal/payment"
	"ad-event-processor/internal/regionproxy"
	"ad-event-processor/internal/reports"
	"ad-event-processor/internal/shardadmin"
	"ad-event-processor/pkg/coldpath"
	"ad-event-processor/pkg/httpresponse"

	"github.com/google/uuid"
)

type Handler struct {
	svc                  *Service
	cfg                  *config.Config
	ipLimiter            *ctrlhttp.IPRateLimiter
	licenseApplyLimiter  *ctrlhttp.IPRateLimiter
	customerLimiter      *ctrlhttp.CustomerRateLimiter
	fraudDecisionLimiter *ctrlhttp.CustomerRateLimiter
	fraudPreviewLimiter  *ctrlhttp.CustomerRateLimiter
	authMiddleware       *AuthMiddleware
	authClient           *identity.AuthClient
	payment              *payment.APIClient
	billing              *ledger.BillingClient
	invoiceDelivery      billingadmin.InvoiceRetryer
}

func NewHandler(svc *Service, cfg *config.Config, authMiddleware *AuthMiddleware, authClient *identity.AuthClient, paymentClient *payment.APIClient, billingClient *ledger.BillingClient) *Handler {
	rps := 10.0
	burst := 50
	if cfg != nil {
		rps = cfg.Management.RateLimitRPS
		burst = cfg.Management.RateLimitBurst
	}
	h := &Handler{
		svc:                  svc,
		cfg:                  cfg,
		ipLimiter:            ctrlhttp.NewIPRateLimiter(rps, burst),
		licenseApplyLimiter:  ctrlhttp.NewIPRateLimiter(ctrlhttp.LicenseApplyRPS, ctrlhttp.LicenseApplyBurst),
		customerLimiter:      ctrlhttp.NewCustomerRateLimiter(),
		fraudDecisionLimiter: ctrlhttp.NewFraudDecisionLimiter(),
		fraudPreviewLimiter:  ctrlhttp.NewFraudPreviewLimiter(),
		authMiddleware:       authMiddleware,
		authClient:           authClient,
		payment:              paymentClient,
		billing:              billingClient,
	}
	if paymentClient != nil {
		svc.SetPayment(paymentClient)
	}
	return h
}

func (h *Handler) RegisterRoutes(mux *http.ServeMux) {
	if h.svc != nil && h.svc.GetPool() != nil {
		RegisterRoutes(mux, h.BuildAdminAPIRegistry(h.svc.GetPool(), h.svc.RedisShards()))
	}

	registerAdminGoneRoutes(mux)
	registerRootRoute(mux, NewAdminUIGate(h.authMiddleware))
	RegisterRegionIngestRoutes(mux, h)
}

func (h *Handler) limit(next http.HandlerFunc) http.HandlerFunc {
	return h.limitByIP(h.pgHigh(next))
}

func (h *Handler) limitByIP(next http.HandlerFunc) http.HandlerFunc {
	return ctrlhttp.LimitByIP(h.ipLimiter, next)
}

func (h *Handler) limitLicenseApply(next http.HandlerFunc) http.HandlerFunc {
	return ctrlhttp.LimitLicenseApply(h.licenseApplyLimiter, next)
}

func (h *Handler) limitExportByCustomer(next http.HandlerFunc) http.HandlerFunc {
	return ctrlhttp.LimitExportByCustomer(h.customerLimiter, next)
}

func (h *Handler) allowFraudPreview(campaignID string) bool {
	return ctrlhttp.AllowFraudPreview(h.fraudPreviewLimiter, campaignID)
}

func (h *Handler) allowFraudDecision(customerID string) bool {
	return ctrlhttp.AllowFraudDecision(h.fraudDecisionLimiter, customerID)
}

func (h *Handler) pgHigh(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if h.svc == nil || h.svc.postgresGate == nil {
			next(w, r)
			return
		}
		if err := h.svc.postgresGate.AcquireHigh(r.Context()); err != nil {
			httpresponse.Error(w, http.StatusServiceUnavailable, "SERVICE_UNAVAILABLE", "database busy")
			return
		}
		defer h.svc.postgresGate.ReleaseHigh()
		next(w, r)
	}
}

func (h *Handler) perm(next http.HandlerFunc, permission string) http.HandlerFunc {
	if h.authMiddleware != nil {
		return h.authMiddleware.RequirePermission(permission)(next)
	}
	return h.authFallback(next)
}

func (h *Handler) authFallback(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		key := r.Header.Get("X-Admin-API-Key")
		if key == "" || h.cfg == nil || key != string(h.cfg.AdminAPIKey) {
			httpresponse.Error(w, http.StatusUnauthorized, "UNAUTHORIZED", "unauthorized")
			return
		}
		user := authz.AuthenticatedUser{
			UserID:     apiKeyPrincipalID(key),
			Role:       ctrlhttp.RoleAdmin,
			AuthSource: "api_key",
		}
		ctx := authz.WithAuthenticatedUser(context.WithValue(r.Context(), UserContextKey, user), user)
		next(w, r.WithContext(ctx))
	}
}

func (h *Handler) ensureCampaignAccess(r *http.Request, campaignID uuid.UUID) error {
	u, ok := GetUser(r.Context())
	if !ok || !u.HasBoundCustomer() {
		return nil
	}
	camp, err := h.svc.GetCampaignRow(r.Context(), campaignID)
	if err != nil {
		return err
	}
	if uuid.UUID(camp.CustomerID.Bytes) != u.CustomerID {
		return errForbidden
	}
	if err := campaign.AssertMediaBuyerCampaignAccess(r.Context(), camp); err != nil {
		return err
	}
	return nil
}

func (h *Handler) ensureCustomerAccess(r *http.Request, customerID string) error {
	u, ok := GetUser(r.Context())
	if !ok || !u.HasBoundCustomer() {
		return nil
	}
	cid, err := uuid.Parse(customerID)
	if err != nil {
		return err
	}
	if u.CustomerID != cid {
		return errForbidden
	}
	return nil
}

func writeForecastError(w http.ResponseWriter, err error) {
	if errors.Is(err, campaign.ErrForecastClickHouseTimeout) || errors.Is(err, campaign.ErrForecastUnavailable) {
		w.Header().Set("Retry-After", strconv.Itoa(campaign.ForecastRetryAfterSec()))
		httpresponse.JSON(w, http.StatusServiceUnavailable, reports.ForecastUnavailableResponse{
			Error: reports.ForecastErrorDetail{
				Code:    "FORECAST_UNAVAILABLE",
				Message: err.Error(),
			},
			RetryAfter: campaign.ForecastRetryAfterSec(),
		})
		return
	}
	if errors.Is(err, ErrClickHouseNotConfigured) {
		httpresponse.Error(w, http.StatusServiceUnavailable, "CLICKHOUSE_UNAVAILABLE", "clickhouse not configured")
		return
	}
	writeServiceError(w, err)
}

func (h *Handler) resolveForecastCustomerID(r *http.Request, bodyCustomerID *uuid.UUID) (*uuid.UUID, error) {
	u, ok := GetUser(r.Context())
	if !ok {
		return nil, errForbidden
	}
	if u.HasBoundCustomer() {
		if bodyCustomerID != nil && *bodyCustomerID != uuid.Nil && *bodyCustomerID != u.CustomerID {
			return nil, errForbidden
		}
		cid := u.CustomerID
		return &cid, nil
	}
	if bodyCustomerID != nil && *bodyCustomerID != uuid.Nil {
		return bodyCustomerID, nil
	}
	return nil, nil
}

func (h *Handler) selfServePerm(next http.HandlerFunc, permission string) http.HandlerFunc {
	if h.authMiddleware != nil {
		return h.authMiddleware.RequireSelfServe(permission)(next)
	}
	return h.perm(next, permission)
}

func (h *Handler) resolveSelfServeCustomerID(r *http.Request, bodyCustomerID *uuid.UUID) (uuid.UUID, error) {
	u, ok := GetUser(r.Context())
	if !ok {
		return uuid.Nil, errForbidden
	}
	if u.HasBoundCustomer() {
		if bodyCustomerID != nil && *bodyCustomerID != uuid.Nil && *bodyCustomerID != u.CustomerID {
			return uuid.Nil, errForbidden
		}
		return u.CustomerID, nil
	}
	if bodyCustomerID == nil || *bodyCustomerID == uuid.Nil {
		return uuid.Nil, errValidation("customer_id is required")
	}
	return *bodyCustomerID, nil
}

type invalidQueryError string

func (e invalidQueryError) Error() string { return string(e) }

type validationError string

func (e validationError) Error() string { return string(e) }

func (s *Service) MultiRegionGlobal() bool {
	return s != nil && s.cfg != nil && s.cfg.MultiRegionGlobal()
}

func (s *Service) ApplyRegionSpendSyncBatch(ctx context.Context, batchDedupKey string, payload []byte) error {
	return s.applyRegionSpendSyncBatch(ctx, batchDedupKey, payload)
}

func (s *Service) EnsureProxyBatchBookAndExecute(ctx context.Context, in regionproxy.BatchInput, apply func(ctx context.Context, claim dedup.ClaimResult) error) (regionproxy.BatchResult, error) {
	worker := s.OperationLeaseWorker()
	if worker == nil {
		worker = NewOperationLeaseWorker(s)
	}
	bookReq := shardadmin.ProxyBatchBookRequest(ctx, s, shardadmin.ProxyBatchBookInput{
		RegionCode:  in.RegionCode,
		NodeID:      in.NodeID,
		SourceEpoch: in.SourceEpoch,
		Seq:         in.Seq,
		FactorU:     in.FactorU,
		OpID:        in.OpID,
	}, 1)
	if _, err := worker.EnsureBook(ctx, bookReq); err != nil {
		return regionproxy.BatchResult{}, err
	}
	var result regionproxy.BatchResult
	err := worker.ExecuteOp(ctx, bookReq.OpID, func(ctx context.Context, _ db.OperationLease, claim dedup.ClaimResult) error {
		result.Outcome = claim.Outcome
		result.DedupKey = claim.DedupKey
		return apply(ctx, claim)
	})
	return result, err
}

func (s *Service) IngestRegionProxyBatch(ctx context.Context, in regionproxy.BatchInput) (regionproxy.BatchResult, error) {
	return regionproxy.IngestBatch(ctx, s, in)
}

func (h *Handler) postRegionIngestBatch(w http.ResponseWriter, r *http.Request) {
	key := r.Header.Get("X-Admin-API-Key")
	if key == "" || h.cfg == nil || key != string(h.cfg.AdminAPIKey) {
		httpresponse.Error(w, http.StatusUnauthorized, "UNAUTHORIZED", "unauthorized")
		return
	}
	body, err := coldpath.ReadLimitedBody(w, r, coldpath.RegionIngestMaxBody)
	if err != nil {
		httpresponse.Error(w, http.StatusBadRequest, "BAD_REQUEST", "invalid body")
		return
	}
	in, err := regionproxy.DecodeBatchJSON(body)
	if err != nil {
		httpresponse.Error(w, http.StatusBadRequest, "BAD_REQUEST", err.Error())
		return
	}
	result, err := h.svc.IngestRegionProxyBatch(r.Context(), in)
	if err != nil {
		writeServiceError(w, err)
		return
	}
	httpresponse.JSON(w, http.StatusOK, map[string]string{
		"outcome":   string(result.Outcome),
		"dedup_key": result.DedupKey,
	})
}
