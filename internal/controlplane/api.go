package controlplane

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"strconv"

	"espx/internal/controlplane/adminapi"
	"espx/internal/dedup"
	"espx/internal/domain"
	db "espx/internal/domain/db"
	"espx/pkg/coldpath"
	"espx/pkg/dedupkey"
	"espx/pkg/httpresponse"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

type invalidQueryError string

func errInvalidQuery(msg string) error {
	return invalidQueryError(msg)
}

func (e invalidQueryError) Error() string { return string(e) }

type validationError string

func (e validationError) Error() string { return string(e) }

func errValidation(msg string) error { return validationError(msg) }

var errForbidden = &forbiddenError{}

type forbiddenError struct{}

func (e *forbiddenError) Error() string { return "forbidden" }

type RegionIngestBatchInput struct {
	RegionCode  uint8
	NodeID      string
	SourceEpoch uint32
	Seq         uint64
	FactorU     uuid.UUID
	Payload     []byte
	OpID        uuid.UUID
}

type RegionIngestBatchResult struct {
	Outcome  dedup.Outcome
	DedupKey string
}

type regionIngestBatchJSON struct {
	RegionCode  uint8  `json:"region_code"`
	NodeID      string `json:"node_id"`
	SourceEpoch uint32 `json:"source_epoch"`
	Seq         uint64 `json:"seq"`
	FactorU     string `json:"factor_u"`
	Payload     []byte `json:"payload"`
	OpID        string `json:"op_id,omitempty"`
}

func parseAPIPagination(r *http.Request) (int32, int32) {
	limit := int32(50)
	if l, err := strconv.ParseInt(r.URL.Query().Get("limit"), 10, 32); err == nil && l > 0 {
		limit = int32(l)
	}
	offset := int32(0)
	if o, err := strconv.ParseInt(r.URL.Query().Get("offset"), 10, 32); err == nil && o > 0 {
		offset = int32(o)
	}
	return coldpath.ClampLimitOffset(limit, offset, 50, 1000)
}

func (h *Handler) ensureCampaignAccess(r *http.Request, campaignID uuid.UUID) error {
	u, ok := GetUser(r.Context())
	if !ok || !u.IsUser() {
		return nil
	}
	camp, err := h.svc.GetCampaignRow(r.Context(), campaignID)
	if err != nil {
		return err
	}
	if uuid.UUID(camp.CustomerID.Bytes) != u.CustomerID {
		return errForbidden
	}
	return nil
}

func (h *Handler) ensureCustomerAccess(r *http.Request, customerID string) error {
	u, ok := GetUser(r.Context())
	if !ok || !u.IsUser() {
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
	if errors.Is(err, adminapi.ErrForecastClickHouseTimeout) || errors.Is(err, adminapi.ErrForecastUnavailable) {
		w.Header().Set("Retry-After", strconv.Itoa(adminapi.ForecastRetryAfterSec()))
		httpresponse.JSON(w, http.StatusServiceUnavailable, adminapi.ForecastUnavailableResponse{
			Error: adminapi.ForecastErrorDetail{
				Code:    "FORECAST_UNAVAILABLE",
				Message: err.Error(),
			},
			RetryAfter: adminapi.ForecastRetryAfterSec(),
		})
		return
	}
	if errors.Is(err, adminapi.ErrClickHouseNotConfigured) {
		httpresponse.Error(w, http.StatusServiceUnavailable, "CLICKHOUSE_UNAVAILABLE", "clickhouse not configured")
		return
	}
	writeServiceError(w, err)
}

func (handler *Handler) resolveForecastCustomerID(r *http.Request, bodyCustomerID *uuid.UUID) (*uuid.UUID, error) {
	u, ok := GetUser(r.Context())
	if !ok {
		return nil, errForbidden
	}
	if u.IsUser() {
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

func (handler *Handler) selfServePerm(next http.HandlerFunc, permission string) http.HandlerFunc {
	if handler.authMiddleware != nil {
		return handler.authMiddleware.RequireSelfServe(permission)(next)
	}
	return handler.perm(next, permission)
}

func (handler *Handler) resolveSelfServeCustomerID(r *http.Request, bodyCustomerID *uuid.UUID) (uuid.UUID, error) {
	u, ok := GetUser(r.Context())
	if !ok {
		return uuid.Nil, errForbidden
	}
	if u.IsUser() {
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

func (s *Service) IngestRegionProxyBatch(ctx context.Context, in RegionIngestBatchInput) (RegionIngestBatchResult, error) {
	if in.RegionCode == 0 {
		return RegionIngestBatchResult{}, fmt.Errorf("region ingest batch region=%d: invalid region code", in.RegionCode)
	}
	if in.NodeID == "" {
		return RegionIngestBatchResult{}, fmt.Errorf("region ingest batch region=%d: node_id required", in.RegionCode)
	}
	if len(in.Payload) == 0 {
		return RegionIngestBatchResult{}, fmt.Errorf("region ingest batch region=%d: empty payload", in.RegionCode)
	}

	var canonBuf [4096 + 64]byte
	canon := dedupkey.WriteCanonicalProxyBatchPayload(canonBuf[:0], in.Seq, in.Payload)
	expected := dedupkey.FactorU(canon)
	if expected != in.FactorU {
		return RegionIngestBatchResult{}, fmt.Errorf("region ingest batch region=%d: factor_u mismatch", in.RegionCode)
	}

	if s != nil && s.cfg != nil && s.cfg.MultiRegionGlobal() {
		return s.ingestRegionProxyBatchLeased(ctx, in)
	}
	return s.ingestRegionProxyBatchDirect(ctx, in)
}

func (s *Service) ingestRegionProxyBatchDirect(ctx context.Context, in RegionIngestBatchInput) (RegionIngestBatchResult, error) {
	epoch := in.SourceEpoch
	if epoch == 0 && s.pool != nil {
		epoch = dedup.LoadRoutingEpoch(ctx, s.pool)
	}
	adapter := dedup.NewAdapter(s.pool, in.RegionCode, epoch)
	if adapter == nil {
		return RegionIngestBatchResult{}, fmt.Errorf("region ingest batch region=%d: dedup adapter unavailable", in.RegionCode)
	}

	seq := int64(in.Seq)
	scope := adapter.RegionScope(dedupkey.ProxySourceID(in.RegionCode, in.NodeID), seq, seq)
	claim, err := adapter.ClaimConfirm(ctx, scope, in.FactorU)
	if err != nil {
		return RegionIngestBatchResult{}, fmt.Errorf("region ingest batch region=%d: %w", in.RegionCode, err)
	}
	if guardErr := dedup.GuardOutcome(claim); guardErr != nil {
		return RegionIngestBatchResult{}, fmt.Errorf("region ingest batch region=%d: %w", in.RegionCode, guardErr)
	}
	if claim.ShouldApply() {
		if err := adapter.RecordApply(ctx, claim.DedupKey); err != nil {
			return RegionIngestBatchResult{}, fmt.Errorf("region ingest batch region=%d: %w", in.RegionCode, err)
		}
		if err := s.applyRegionSpendSyncBatch(ctx, claim.DedupKey, in.Payload); err != nil {
			return RegionIngestBatchResult{}, fmt.Errorf("region ingest batch region=%d: %w", in.RegionCode, err)
		}
	}
	return RegionIngestBatchResult{
		Outcome:  claim.Outcome,
		DedupKey: claim.DedupKey,
	}, nil
}

func (s *Service) ingestRegionProxyBatchLeased(ctx context.Context, in RegionIngestBatchInput) (RegionIngestBatchResult, error) {
	worker := s.OperationLeaseWorker()
	if worker == nil {
		worker = NewOperationLeaseWorker(s)
	}
	bookReq := ProxyBatchBookRequest(ctx, s, in, 1)
	if _, err := worker.EnsureBook(ctx, bookReq); err != nil {
		return RegionIngestBatchResult{}, fmt.Errorf("region ingest batch region=%d: %w", in.RegionCode, err)
	}

	var result RegionIngestBatchResult
	err := worker.ExecuteOp(ctx, bookReq.OpID, func(ctx context.Context, _ db.OperationLease, claim dedup.ClaimResult) error {
		result = RegionIngestBatchResult{
			Outcome:  claim.Outcome,
			DedupKey: claim.DedupKey,
		}
		if claim.ShouldApply() {
			return s.applyRegionSpendSyncBatch(ctx, claim.DedupKey, in.Payload)
		}
		return nil
	})
	if err != nil {
		return RegionIngestBatchResult{}, fmt.Errorf("region ingest batch region=%d: %w", in.RegionCode, err)
	}
	return result, nil
}

func (h *Handler) registerRegionIngestRoutes(mux *http.ServeMux) {
	if h.cfg == nil || !h.cfg.MultiRegionGlobal() {
		return
	}
	mux.HandleFunc("POST /api/v1/region/ingest/batch", h.pgHigh(h.postRegionIngestBatch))
}

func (h *Handler) postRegionIngestBatch(w http.ResponseWriter, r *http.Request) {
	key := r.Header.Get("X-Admin-API-Key")
	if key == "" || h.cfg == nil || key != string(h.cfg.AdminAPIKey) {
		httpresponse.Error(w, http.StatusUnauthorized, "UNAUTHORIZED", "unauthorized")
		return
	}
	body, err := io.ReadAll(io.LimitReader(r.Body, 4*1024*1024))
	if err != nil {
		httpresponse.Error(w, http.StatusBadRequest, "BAD_REQUEST", "invalid body")
		return
	}
	var in regionIngestBatchJSON
	if err := json.Unmarshal(body, &in); err != nil {
		httpresponse.Error(w, http.StatusBadRequest, "BAD_REQUEST", "invalid json")
		return
	}
	factorU, err := uuid.Parse(in.FactorU)
	if err != nil {
		httpresponse.Error(w, http.StatusBadRequest, "BAD_REQUEST", "invalid factor_u")
		return
	}
	var opID uuid.UUID
	if in.OpID != "" {
		opID, err = uuid.Parse(in.OpID)
		if err != nil {
			httpresponse.Error(w, http.StatusBadRequest, "BAD_REQUEST", "invalid op_id")
			return
		}
	}
	result, err := h.svc.IngestRegionProxyBatch(r.Context(), RegionIngestBatchInput{
		RegionCode:  in.RegionCode,
		NodeID:      in.NodeID,
		SourceEpoch: in.SourceEpoch,
		Seq:         in.Seq,
		FactorU:     factorU,
		Payload:     in.Payload,
		OpID:        opID,
	})
	if err != nil {
		writeServiceError(w, err)
		return
	}
	httpresponse.JSON(w, http.StatusOK, map[string]string{
		"outcome":   string(result.Outcome),
		"dedup_key": result.DedupKey,
	})
}

func mapServiceError(err error) (status int, code, message string) {
	if err == nil {
		return http.StatusOK, "", ""
	}
	if errors.Is(err, errForbidden) {
		return http.StatusForbidden, "FORBIDDEN", "forbidden"
	}
	if errors.Is(err, ErrInstallTokenInvalid) {
		return http.StatusUnauthorized, "UNAUTHORIZED", ErrInstallTokenInvalid.Error()
	}

	if errors.Is(err, ErrSelfServeActiveCampaignLimit) || errors.Is(err, ErrSelfServeDailyCreateLimit) {
		return http.StatusTooManyRequests, "LIMIT_EXCEEDED", err.Error()
	}

	var q invalidQueryError
	if errors.As(err, &q) {
		return http.StatusBadRequest, "BAD_REQUEST", string(q)
	}

	var ve validationError
	if errors.As(err, &ve) {
		return http.StatusBadRequest, "BAD_REQUEST", string(ve)
	}

	if isNotFoundError(err) {
		return http.StatusNotFound, "NOT_FOUND", "resource not found"
	}

	if isConflictError(err) {
		return http.StatusConflict, "CONFLICT", conflictMessage(err)
	}

	if errors.Is(err, ErrSellersJSONInvalid) {
		return http.StatusServiceUnavailable, "SUPPLY_INVALID", ErrSellersJSONInvalid.Error()
	}

	if msg, ok := badRequestMessage(err); ok {
		return http.StatusBadRequest, "BAD_REQUEST", msg
	}

	return http.StatusInternalServerError, "INTERNAL_ERROR", "internal error"
}

func isNotFoundError(err error) bool {
	return errors.Is(err, pgx.ErrNoRows) ||
		errors.Is(err, ErrCustomerNotFound) ||
		errors.Is(err, ErrPaymentTopupNotFound) ||
		errors.Is(err, ErrCampaignNotFound) ||
		errors.Is(err, ErrBrandNotFound) ||
		errors.Is(err, ErrCreativeNotFound) ||
		errors.Is(err, ErrTemplateNotFound) ||
		errors.Is(err, ErrRtbDealNotFound) ||
		errors.Is(err, ErrDealCustomerMissing) ||
		errors.Is(err, ErrSellerNotFound) ||
		errors.Is(err, ErrAdsTxtEntryNotFound) ||
		errors.Is(err, domain.ErrSlotMapVersionNotFound)
}

func isConflictError(err error) bool {
	return errors.Is(err, ErrSlotMigrationNotReady) ||
		errors.Is(err, domain.ErrSlotMapAlreadyActive) ||
		errors.Is(err, ErrPlatformConfigBootstrapped)
}

func conflictMessage(err error) string {
	switch {
	case errors.Is(err, ErrSlotMigrationNotReady):
		return ErrSlotMigrationNotReady.Error()
	case errors.Is(err, domain.ErrSlotMapAlreadyActive):
		return domain.ErrSlotMapAlreadyActive.Error()
	default:
		return "conflict"
	}
}

func badRequestMessage(err error) (string, bool) {
	switch {
	case errors.Is(err, ErrFeedbackInvalidType),
		errors.Is(err, ErrFeedbackInvalidEmail),
		errors.Is(err, ErrFeedbackEmptyMessage):
		return err.Error(), true
	case errors.Is(err, ErrInsufficientBalance):
		return ErrInsufficientBalance.Error(), true
	case errors.Is(err, ErrSelfServeActiveCampaignLimit),
		errors.Is(err, ErrSelfServeDailyCreateLimit),
		errors.Is(err, ErrSelfServeBudgetOutOfRange):
		return err.Error(), true
	case errors.Is(err, ErrBrandBelongsToAnotherCustomer),
		errors.Is(err, ErrTemplateBelongsToAnotherCustomer),
		errors.Is(err, ErrCampaignCannotBePaused),
		errors.Is(err, ErrCampaignNotPaused),
		errors.Is(err, ErrCampaignOutsideSchedule),
		errors.Is(err, ErrInvalidPacingMode),
		errors.Is(err, ErrWeightMustBePositive),
		errors.Is(err, ErrCreativeStatusInvalid),
		errors.Is(err, ErrIncompleteIdempotency),
		errors.Is(err, ErrUnsupportedGranularity),
		errors.Is(err, ErrInvalidTimeRange),
		errors.Is(err, ErrInvalidServiceFilter),
		errors.Is(err, ErrInvalidDealPacing),
		errors.Is(err, ErrDuplicateDealID),
		errors.Is(err, ErrInvalidDealSeats),
		errors.Is(err, ErrInvalidSellerType),
		errors.Is(err, ErrInvalidRelationship),
		errors.Is(err, ErrSupplyChainTooLong),
		errors.Is(err, ErrRefundExceedsTopup),
		errors.Is(err, ErrChargebackExceedsTopup),
		errors.Is(err, ErrChargebackReversalExceedsWithdrawn),
		errors.Is(err, errExportLimit),
		errors.Is(err, domain.ErrSlotMapIncomplete),
		errors.Is(err, domain.ErrSlotMapInvalidSlot),
		errors.Is(err, domain.ErrSlotMapInvalidShard),
		errors.Is(err, ErrPlatformConfigNotBootstrapped):
		return err.Error(), true
	default:
		return "", false
	}
}

func writeServiceError(w http.ResponseWriter, err error, logAttrs ...any) {
	status, code, message := mapServiceError(err)
	if status >= http.StatusInternalServerError {
		attrs := append([]any{slog.Any("err", err)}, logAttrs...)
		slog.Error("management request failed", attrs...)
	}
	httpresponse.Error(w, status, code, message)
}
