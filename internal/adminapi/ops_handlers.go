package adminapi

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"net/http"
	"strconv"
	"strings"

	"espx/pkg/coldpath"

	"espx/pkg/httpresponse"
)

type OpsHTTPHandlers struct {
	OpsReader               ManagementOpsReader
	PaymentIntents          PaymentLister
	ConsentRecorder         ConsentRecorder
	ConsentVerifier         ConsentVerifier
	AuditLister             AuditLister
	RolesReloader           RolesReloader
	PlansReloader           PlansReloader
	Blacklist               BlacklistAdmin
	ApplyRateLimit          func(http.HandlerFunc) http.HandlerFunc
	RequirePermission       func(string, http.HandlerFunc) http.HandlerFunc
	WriteServiceError       func(http.ResponseWriter, error)
	AuthorizeCustomerAccess func(*http.Request, string) error
	SupportBundle           SupportBundleWriter
}

func (ops *OpsHTTPHandlers) Register(mux *http.ServeMux) {
	if ops == nil || ops.OpsReader == nil {
		return
	}
	limit := ops.ApplyRateLimit
	perm := ops.RequirePermission
	if limit == nil {
		limit = func(next http.HandlerFunc) http.HandlerFunc { return next }
	}
	if perm == nil {
		perm = func(_ string, next http.HandlerFunc) http.HandlerFunc { return next }
	}

	mux.HandleFunc("GET /api/v1/ops/incidents", limit(perm("shards:read", ops.getIncidents)))
	mux.HandleFunc("GET /api/v1/ops/outbox", limit(perm("shards:read", ops.listOutbox)))
	mux.HandleFunc("GET /api/v1/ops/dlq", limit(perm("shards:read", ops.listDLQ)))
	mux.HandleFunc("POST /api/v1/ops/dlq/{id}/retry", limit(perm("shards:write", ops.retryDLQ)))
	mux.HandleFunc("GET /api/v1/ops/shards", limit(perm("shards:read", ops.getShards)))
	mux.HandleFunc("GET /api/v1/audit/export", limit(perm("audit:read", ops.exportAudit)))
	mux.HandleFunc("GET /api/v1/customers/{id}/payments", limit(perm("customers:read", ops.listCustomerPayments)))
	ops.registerReconRoutes(mux)
	ops.registerConsentRoutes(mux)
	ops.registerAuditRoutes(mux)
	ops.registerRolesRoutes(mux)
	ops.registerPlansRoutes(mux)
	ops.registerBlacklistRoutes(mux)
	ops.registerDashboardRoutes(mux)
	ops.registerSupportBundleRoutes(mux)
}

func (ops *OpsHTTPHandlers) getIncidents(w http.ResponseWriter, r *http.Request) {
	snap, err := ops.OpsReader.GetIncidentSnapshot(r.Context())
	if err != nil {
		ops.writeServiceError(w, err)
		return
	}
	if len(snap.Errors) > 0 && len(snap.Shards) == 0 && len(snap.StreamLag) == 0 {
		httpresponse.JSON(w, http.StatusServiceUnavailable, snap)
		return
	}
	httpresponse.JSON(w, http.StatusOK, snap)
}

func (ops *OpsHTTPHandlers) listOutbox(w http.ResponseWriter, r *http.Request) {
	limit := parsePaginationLimit(r)
	result, err := ops.OpsReader.ListOutboxEvents(r.Context(), r.URL.Query().Get("status"), r.URL.Query().Get("event_type"), r.URL.Query().Get("cursor"), limit)
	if err != nil {
		ops.writeServiceError(w, err)
		return
	}
	httpresponse.JSON(w, http.StatusOK, result)
}

func (ops *OpsHTTPHandlers) listDLQ(w http.ResponseWriter, r *http.Request) {
	limit := 50
	if v := r.URL.Query().Get("limit"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 {
			limit = n
		}
	}
	result, err := ops.OpsReader.ListDLQEntries(r.Context(), r.URL.Query().Get("cursor"), limit)
	if err != nil {
		ops.writeServiceError(w, err)
		return
	}
	if len(result.Errors) > 0 && len(result.Items) == 0 {
		httpresponse.JSON(w, http.StatusServiceUnavailable, result)
		return
	}
	httpresponse.JSON(w, http.StatusOK, result)
}

type dlqRetryRequest struct {
	ShardID int    `json:"shard_id"`
	Stream  string `json:"stream"`
	EntryID string `json:"entry_id"`
}

func (ops *OpsHTTPHandlers) retryDLQ(w http.ResponseWriter, r *http.Request) {
	idempotencyKey := r.Header.Get("Idempotency-Key")
	if idempotencyKey == "" {
		httpresponse.Error(w, http.StatusBadRequest, "BAD_REQUEST", "Idempotency-Key header is required")
		return
	}

	dlqID := r.PathValue("id")
	var req dlqRetryRequest
	if stringsHasPrefixJSON(r) {
		body, err := coldpath.ReadLimitedBody(w, r, coldpath.DefaultMaxBody)
		if err != nil {
			return
		}
		if len(body) > 0 {
			decoded, decodeErr := coldpath.DecodeBody[dlqRetryRequest](body)
			if decodeErr != nil {
				httpresponse.Error(w, http.StatusBadRequest, "BAD_REQUEST", "invalid request body")
				return
			}
			req = decoded
		}
	}

	if req.EntryID == "" {
		req.EntryID = parseDLQEntryIDFromRoute(dlqID)
	}
	if req.ShardID == 0 {
		req.ShardID = parseDLQShardFromRoute(dlqID)
	}

	payload := DLQRetryPayload{
		ShardID: req.ShardID,
		Stream:  req.Stream,
		EntryID: req.EntryID,
		DLQID:   dlqID,
	}
	dedup := sha256.Sum256([]byte(dlqID + idempotencyKey))
	if err := ops.OpsReader.EnqueueDLQRetry(r.Context(), payload, hex.EncodeToString(dedup[:])); err != nil {
		ops.writeServiceError(w, err)
		return
	}
	w.WriteHeader(http.StatusAccepted)
}

func (ops *OpsHTTPHandlers) getShards(w http.ResponseWriter, r *http.Request) {
	report, err := ops.OpsReader.GetShardHealthFanOut(r.Context())
	if err != nil {
		ops.writeServiceError(w, err)
		return
	}
	if len(report.Errors) > 0 && len(report.Shards) == 0 {
		httpresponse.JSON(w, http.StatusServiceUnavailable, report)
		return
	}
	httpresponse.JSON(w, http.StatusOK, report)
}

func (ops *OpsHTTPHandlers) exportAudit(w http.ResponseWriter, r *http.Request) {
	if r.URL.Query().Get("format") != "csv" {
		httpresponse.Error(w, http.StatusBadRequest, "BAD_REQUEST", "format must be csv")
		return
	}

	customerRaw := r.URL.Query().Get("customer_id")
	if customerRaw != "" && ops.AuthorizeCustomerAccess != nil {
		if err := ops.AuthorizeCustomerAccess(r, customerRaw); err != nil {
			ops.writeServiceError(w, err)
			return
		}
	}

	cursor := r.URL.Query().Get("cursor")
	w.Header().Set("Content-Type", "text/csv; charset=utf-8")
	w.Header().Set("Cache-Control", "no-store")

	var buf bytes.Buffer
	result, err := ops.OpsReader.ExportAuditCSV(r.Context(), cursor, &buf)
	if err != nil {
		ops.writeServiceError(w, err)
		return
	}
	if result.Truncated {
		w.Header().Set("X-Export-Truncated", "true")
		w.Header().Set("X-Next-Cursor", result.NextCursor)
	}
	w.Header().Set("X-Export-Bytes", strconv.Itoa(result.Bytes))
	if _, err := w.Write(buf.Bytes()); err != nil {
		return
	}
}

type PaymentHistoryRow struct {
	IntentID       string `json:"intent_id"`
	CustomerID     string `json:"customer_id"`
	AmountMicro    int64  `json:"amount_micro"`
	Currency       string `json:"currency"`
	Status         string `json:"status"`
	Provider       string `json:"provider,omitempty"`
	ProviderRef    string `json:"provider_ref,omitempty"`
	IdempotencyKey string `json:"idempotency_key,omitempty"`
	LedgerEntryID  string `json:"ledger_entry_id,omitempty"`
	CreatedAt      string `json:"created_at"`
	UpdatedAt      string `json:"updated_at"`
}

type PaymentLister interface {
	ListPaymentIntents(ctx context.Context, customerID string, limit, offset int32) (PaymentIntentList, error)
}

type PaymentIntentList struct {
	Intents []PaymentIntentRow
	Total   int64
}

type PaymentIntentRow struct {
	ID             string
	CustomerID     string
	AmountMicro    int64
	Currency       string
	Status         string
	Provider       string
	ProviderRef    string
	IdempotencyKey string
	CreatedAt      string
	UpdatedAt      string
}

func (ops *OpsHTTPHandlers) listCustomerPayments(w http.ResponseWriter, r *http.Request) {
	idStr := r.PathValue("id")
	if ops.AuthorizeCustomerAccess != nil {
		if err := ops.AuthorizeCustomerAccess(r, idStr); err != nil {
			ops.writeServiceError(w, err)
			return
		}
	}

	if ops.PaymentIntents == nil {
		httpresponse.Error(w, http.StatusServiceUnavailable, "PAYMENT_UNAVAILABLE", "payment service not configured")
		return
	}

	limit, offset := parsePagination(r)
	resp, err := ops.PaymentIntents.ListPaymentIntents(r.Context(), idStr, limit, offset)
	if err != nil {
		ops.writeServiceError(w, err)
		return
	}

	rows := make([]PaymentHistoryRow, 0, len(resp.Intents))
	for _, intent := range resp.Intents {
		row := PaymentHistoryRow{
			IntentID:       intent.ID,
			CustomerID:     intent.CustomerID,
			AmountMicro:    intent.AmountMicro,
			Currency:       intent.Currency,
			Status:         intent.Status,
			Provider:       intent.Provider,
			ProviderRef:    intent.ProviderRef,
			IdempotencyKey: intent.IdempotencyKey,
			CreatedAt:      intent.CreatedAt,
			UpdatedAt:      intent.UpdatedAt,
		}
		if ledgerID, lerr := ops.OpsReader.LookupLedgerIDForPaymentIntent(r.Context(), intent.ID); lerr == nil {
			row.LedgerEntryID = ledgerID
		}
		rows = append(rows, row)
	}

	httpresponse.JSON(w, http.StatusOK, map[string]any{
		"items":  rows,
		"total":  resp.Total,
		"limit":  limit,
		"offset": offset,
	})
}

func (ops *OpsHTTPHandlers) writeServiceError(w http.ResponseWriter, err error) {
	if ops.WriteServiceError != nil {
		ops.WriteServiceError(w, err)
		return
	}
	httpresponse.Error(w, http.StatusInternalServerError, "INTERNAL", "request failed")
}

func parsePaginationLimit(r *http.Request) int32 {
	limit, _ := parsePagination(r)
	return limit
}

func stringsHasPrefixJSON(r *http.Request) bool {
	ct := r.Header.Get("Content-Type")
	return len(ct) >= 16 && ct[:16] == "application/json"
}

func parseDLQShardFromRoute(dlqID string) int {
	const prefix = "shard-"
	if len(dlqID) < len(prefix)+2 {
		return 0
	}
	if dlqID[:6] != prefix {
		return 0
	}
	rest := dlqID[6:]
	for i, ch := range rest {
		if ch == '-' {
			n, err := strconv.Atoi(rest[:i])
			if err == nil {
				return n
			}
			break
		}
	}
	return 0
}

func parseDLQEntryIDFromRoute(dlqID string) string {
	const prefix = "shard-"
	if !strings.HasPrefix(dlqID, prefix) {
		return ""
	}
	rest := dlqID[len(prefix):]
	dash := strings.Index(rest, "-")
	if dash < 0 || dash+1 >= len(rest) {
		return ""
	}
	return rest[dash+1:]
}
