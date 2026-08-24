package controlplane

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"
	"unicode"

	"ad-event-processor/internal/config"
	"ad-event-processor/internal/domain"
	"ad-event-processor/internal/metrics"
	"ad-event-processor/pkg/coldpath"
	"ad-event-processor/pkg/httpresponse"
	"ad-event-processor/pkg/supportbundle"
)

type OpsHTTPHandlers struct {
	OpsReader               ManagementOpsReader
	PaymentIntents          domain.PaymentAPI
	ConsentRecorder         ConsentRecorder
	ConsentVerifier         ConsentVerifier
	AuditLister             AuditLister
	RolesReloader           RolesReloader
	Blacklist               BlacklistAdmin
	Shard0Catchup           Shard0CatchupRunner
	FraudThreat             FraudThreatEnqueuer
	ApplyRateLimit          func(http.HandlerFunc) http.HandlerFunc
	RequirePermission       func(string, http.HandlerFunc) http.HandlerFunc
	WriteServiceError       func(http.ResponseWriter, error)
	AuthorizeCustomerAccess func(*http.Request, string) error
	SupportBundle           SupportBundleWriter
	RUMStore                RUMStore
	FraudPresets            FraudPresetsService
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
	mux.HandleFunc("GET /api/v1/ops/dlq/inbox", limit(perm("shards:read", ops.listDLQInbox)))
	mux.HandleFunc("POST /api/v1/ops/dlq/{id}/retry", limit(perm("shards:write", ops.retryDLQ)))
	mux.HandleFunc("POST /api/v1/ops/dlq/inbox/{id}/retry", limit(perm("shards:write", ops.retryDLQInbox)))
	mux.HandleFunc("GET /api/v1/ops/shards", limit(perm("shards:read", ops.getShards)))
	mux.HandleFunc("POST /api/v1/ops/shards/0/catchup", limit(perm("shards:write", ops.postShard0Catchup)))
	mux.HandleFunc("GET /api/v1/audit/export", limit(perm("audit:read", ops.exportAudit)))
	mux.HandleFunc("GET /api/v1/customers/{id}/payments", limit(perm("customers:read", ops.listCustomerPayments)))
	ops.registerReconRoutes(mux)
	ops.registerConsentRoutes(mux)
	ops.registerAuditRoutes(mux)
	ops.registerRolesRoutes(mux)
	ops.registerBlacklistRoutes(mux)
	ops.registerFraudThreatRoutes(mux)
	ops.registerDashboardRoutes(mux)
	ops.registerSupportBundleRoutes(mux)
	ops.registerRUMRoutes(mux)
	ops.registerMLModelRoutes(mux)
	ops.registerFraudPresetOpsRoutes(mux, limit, perm)
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
	limit, _ := coldpath.ParseAPIPagination(r)
	result, err := ops.OpsReader.ListDLQEntries(r.Context(), r.URL.Query().Get("cursor"), int(limit))
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

func (ops *OpsHTTPHandlers) listDLQInbox(w http.ResponseWriter, r *http.Request) {
	limit, _ := coldpath.ParseAPIPagination(r)
	result, err := ops.OpsReader.ListDLQInbox(r.Context(), r.URL.Query().Get("source"), r.URL.Query().Get("cursor"), int(limit))
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

type dlqInboxRetryRequest struct {
	Source string `json:"source"`
}

func (ops *OpsHTTPHandlers) retryDLQInbox(w http.ResponseWriter, r *http.Request) {
	idempotencyKey := r.Header.Get("Idempotency-Key")
	if idempotencyKey == "" {
		httpresponse.Error(w, http.StatusBadRequest, "BAD_REQUEST", "Idempotency-Key header is required")
		return
	}
	source := strings.TrimSpace(r.URL.Query().Get("source"))
	if stringsHasPrefixJSON(r) {
		body, err := coldpath.ReadLimitedBody(w, r, coldpath.DefaultMaxBody)
		if err != nil {
			return
		}
		if len(body) > 0 {
			decoded, decodeErr := coldpath.DecodeBody[dlqInboxRetryRequest](body)
			if decodeErr != nil {
				httpresponse.Error(w, http.StatusBadRequest, "BAD_REQUEST", "invalid request body")
				return
			}
			if source == "" {
				source = decoded.Source
			}
		}
	}
	if source == "" {
		httpresponse.Error(w, http.StatusBadRequest, "BAD_REQUEST", "source is required")
		return
	}
	dlqID := r.PathValue("id")
	dedup := sha256.Sum256([]byte(source + ":" + dlqID + ":" + idempotencyKey))
	if err := ops.OpsReader.RetryDLQInbox(r.Context(), source, dlqID, hex.EncodeToString(dedup[:])); err != nil {
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

func (ops *OpsHTTPHandlers) postShard0Catchup(w http.ResponseWriter, r *http.Request) {
	if ops.Shard0Catchup == nil {
		httpresponse.Error(w, http.StatusServiceUnavailable, "UNAVAILABLE", "shard 0 catch-up not configured")
		return
	}
	if err := ops.Shard0Catchup.RunShard0Catchup(r.Context()); err != nil {
		ops.writeServiceError(w, err)
		return
	}
	httpresponse.JSON(w, http.StatusOK, shard0CatchupResponse{Status: "ok"})
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
	redactPII := r.URL.Query().Get("redact_pii") == "true"
	w.Header().Set("Content-Type", "text/csv; charset=utf-8")
	w.Header().Set("Cache-Control", "no-store")

	var buf bytes.Buffer
	result, err := ops.OpsReader.ExportAuditCSV(r.Context(), cursor, redactPII, &buf)
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

	limit, offset := coldpath.ParseAPIPagination(r)
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
		}
		if !intent.CreatedAt.IsZero() {
			row.CreatedAt = intent.CreatedAt.UTC().Format(time.RFC3339)
		}
		if !intent.UpdatedAt.IsZero() {
			row.UpdatedAt = intent.UpdatedAt.UTC().Format(time.RFC3339)
		}
		if ledgerID, lerr := ops.OpsReader.LookupLedgerIDForPaymentIntent(r.Context(), intent.ID); lerr == nil {
			row.LedgerEntryID = ledgerID
		}
		rows = append(rows, row)
	}

	httpresponse.JSON(w, http.StatusOK, PaymentIntentListResponse{
		Items:  rows,
		Total:  resp.Total,
		Limit:  limit,
		Offset: offset,
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
	limit, _ := coldpath.ParseAPIPagination(r)
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

func (ops *OpsHTTPHandlers) registerReconRoutes(mux *http.ServeMux) {
	if ops.OpsReader == nil {
		return
	}
	limit := ops.ApplyRateLimit
	perm := ops.RequirePermission
	mux.HandleFunc("GET /api/v1/recon/runs", limit(perm("audit:read", ops.listReconRuns)))
}

func (ops *OpsHTTPHandlers) listReconRuns(w http.ResponseWriter, r *http.Request) {
	service := r.URL.Query().Get("service")
	limit, offset := coldpath.ParseAPIPagination(r)

	runs, total, err := ops.OpsReader.ListReconRuns(r.Context(), service, limit, offset)
	if err != nil {
		ops.writeServiceError(w, err)
		return
	}

	coldpath.WritePaginatedJSON(w, runs, total)
}

func (ops *OpsHTTPHandlers) registerAuditRoutes(mux *http.ServeMux) {
	if ops.AuditLister == nil {
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
	mux.HandleFunc("GET /api/v1/audit", limit(perm("audit:read", ops.listAudit)))
}

func (ops *OpsHTTPHandlers) listAudit(w http.ResponseWriter, r *http.Request) {
	limit, offset := coldpath.ParseAPIPagination(r)
	redact := r.URL.Query().Get("redact_pii") == "true"
	logs, total, err := ops.AuditLister.ListAuditLogs(r.Context(), limit, offset, redact)
	if err != nil {
		ops.writeServiceError(w, err)
		return
	}
	w.Header().Set("X-Total-Count", strconv.FormatInt(total, 10))
	httpresponse.JSON(w, http.StatusOK, logs)
}

func (ops *OpsHTTPHandlers) registerConsentRoutes(mux *http.ServeMux) {
	limit := ops.ApplyRateLimit
	if limit == nil {
		limit = func(next http.HandlerFunc) http.HandlerFunc { return next }
	}
	perm := ops.RequirePermission
	if perm == nil {
		perm = func(_ string, next http.HandlerFunc) http.HandlerFunc { return next }
	}
	if ops.ConsentRecorder != nil && ops.ConsentVerifier != nil {
		mux.HandleFunc("POST /api/v1/consent", limit(ops.postConsent))
	}
	if ops.OpsReader != nil {
		mux.HandleFunc("GET /api/v1/ops/consent/proofs", limit(perm("shards:read", ops.listConsentProofs)))
	}
}

func (ops *OpsHTTPHandlers) listConsentProofs(w http.ResponseWriter, r *http.Request) {
	if ops.OpsReader == nil {
		httpresponse.Error(w, http.StatusServiceUnavailable, "UNAVAILABLE", "consent proofs not configured")
		return
	}
	limit := int32(50)
	if raw := r.URL.Query().Get("limit"); raw != "" {
		if n, err := strconv.ParseInt(raw, 10, 32); err == nil && n > 0 {
			limit = int32(n)
		}
	}
	result, err := ops.OpsReader.ListConsentProofs(r.Context(), r.URL.Query().Get("user_id"), r.URL.Query().Get("cursor"), limit)
	if err != nil {
		ops.writeServiceError(w, err)
		return
	}
	httpresponse.JSON(w, http.StatusOK, result)
}

func (ops *OpsHTTPHandlers) postConsent(w http.ResponseWriter, r *http.Request) {
	body, err := coldpath.ReadLimitedBody(w, r, coldpath.DefaultMaxBody)
	if err != nil {
		httpresponse.Error(w, http.StatusBadRequest, "BAD_REQUEST", "invalid body")
		return
	}
	sig := r.Header.Get("X-Consent-Signature")
	if err := ops.ConsentVerifier.Verify(body, sig); err != nil {
		httpresponse.Error(w, http.StatusUnauthorized, "INVALID_SIGNATURE", "consent signature invalid")
		return
	}
	var in ConsentRecord
	if err := json.Unmarshal(body, &in); err != nil {
		httpresponse.Error(w, http.StatusBadRequest, "BAD_REQUEST", "invalid json")
		return
	}
	if err := ops.ConsentRecorder.RecordConsent(r.Context(), in); err != nil {
		ops.writeServiceError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (ops *OpsHTTPHandlers) registerRolesRoutes(mux *http.ServeMux) {
	if ops.RolesReloader == nil {
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
	mux.HandleFunc("POST /api/v1/ops/roles/reload", limit(perm("settings:write", ops.reloadRoles)))
}

func (ops *OpsHTTPHandlers) reloadRoles(w http.ResponseWriter, r *http.Request) {
	if ops.RolesReloader == nil {
		httpresponse.Error(w, http.StatusServiceUnavailable, "UNAVAILABLE", "roles reloader not configured")
		return
	}
	if err := ops.RolesReloader.ReloadRoles(); err != nil {
		slog.Error("roles reload failed", "err", err)
		httpresponse.Error(w, http.StatusInternalServerError, "INTERNAL", "failed to reload roles")
		return
	}
	httpresponse.JSON(w, http.StatusOK, map[string]string{"status": "reloaded", "path": ops.RolesReloader.RolesPath()})
}

func (ops *OpsHTTPHandlers) registerFraudThreatRoutes(mux *http.ServeMux) {
	if ops.FraudThreat == nil {
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
	mux.HandleFunc("POST /api/v1/ops/fraud-threat", limit(perm("shards:write", ops.enqueueFraudThreat)))
}

func (ops *OpsHTTPHandlers) enqueueFraudThreat(w http.ResponseWriter, r *http.Request) {
	req, err := coldpath.DecodeRequest[struct {
		Action     string                   `json:"action"`
		IP         string                   `json:"ip"`
		CampaignID string                   `json:"campaign_id"`
		Score      float64                  `json:"score"`
		Boost      int32                    `json:"boost"`
		TTLSeconds int64                    `json:"ttl_seconds"`
		Items      []FraudThreatEnqueueItem `json:"items"`
	}](w, r, coldpath.DefaultMaxBody)
	if err != nil {
		httpresponse.Error(w, http.StatusBadRequest, "BAD_REQUEST", "invalid request body")
		return
	}

	if len(req.Items) > 0 {
		n, err := ops.FraudThreat.EnqueueFraudThreatBatch(r.Context(), req.Items)
		if err != nil {
			ops.writeServiceError(w, err)
			return
		}
		httpresponse.JSON(w, http.StatusOK, map[string]int{"enqueued": n})
		return
	}

	if req.Action == "" || req.CampaignID == "" {
		httpresponse.Error(w, http.StatusBadRequest, "BAD_REQUEST", "invalid request body")
		return
	}
	if err := ops.FraudThreat.EnqueueFraudThreat(r.Context(), req.Action, req.IP, req.CampaignID, req.Score, req.Boost, req.TTLSeconds); err != nil {
		ops.writeServiceError(w, err)
		return
	}
	httpresponse.JSON(w, http.StatusOK, map[string]bool{"enqueued": true})
}

func (ops *OpsHTTPHandlers) registerBlacklistRoutes(mux *http.ServeMux) {
	if ops.Blacklist == nil {
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
	mux.HandleFunc("POST /api/v1/ops/blacklist", limit(perm("blacklist:write", ops.blockIP)))
	mux.HandleFunc("DELETE /api/v1/ops/blacklist", limit(perm("blacklist:write", ops.unblockIP)))
	mux.HandleFunc("GET /api/v1/ops/blacklist", limit(perm("blacklist:read", ops.listBlacklist)))
}

func (ops *OpsHTTPHandlers) blockIP(w http.ResponseWriter, r *http.Request) {
	req, err := coldpath.DecodeRequest[struct {
		IP         string `json:"ip"`
		Source     string `json:"source"`
		TTLSeconds *int64 `json:"ttl_seconds,omitempty"`
	}](w, r, coldpath.DefaultMaxBody)
	if err != nil || req.IP == "" {
		httpresponse.Error(w, http.StatusBadRequest, "BAD_REQUEST", "invalid request body")
		return
	}
	if r.Header.Get("X-Dry-Run") == "1" || r.URL.Query().Get("dry_run") == "1" {
		preview, err := ops.Blacklist.PreviewBlockIP(r.Context(), req.IP, req.Source, req.TTLSeconds)
		if err != nil {
			ops.writeServiceError(w, err)
			return
		}
		httpresponse.JSON(w, http.StatusOK, preview)
		return
	}
	if err := ops.Blacklist.BlockIPWithTTL(r.Context(), req.IP, req.Source, req.TTLSeconds); err != nil {
		ops.writeServiceError(w, err)
		return
	}
	w.WriteHeader(http.StatusCreated)
}

func (ops *OpsHTTPHandlers) unblockIP(w http.ResponseWriter, r *http.Request) {
	req, err := coldpath.DecodeRequest[struct {
		IP     string `json:"ip"`
		Source string `json:"source"`
	}](w, r, coldpath.DefaultMaxBody)
	if err != nil || req.IP == "" {
		httpresponse.Error(w, http.StatusBadRequest, "BAD_REQUEST", "invalid request body")
		return
	}
	if err := ops.Blacklist.UnblockIP(r.Context(), req.IP, req.Source); err != nil {
		ops.writeServiceError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (ops *OpsHTTPHandlers) listBlacklist(w http.ResponseWriter, r *http.Request) {
	limit, offset := coldpath.ParseAPIPagination(r)
	items, total, err := ops.Blacklist.ListBlacklist(r.Context(), limit, offset)
	if err != nil {
		ops.writeServiceError(w, err)
		return
	}
	w.Header().Set("X-Total-Count", strconv.FormatInt(total, 10))
	httpresponse.JSON(w, http.StatusOK, BlacklistListResponse{Items: items, Total: total})
}

func (ops *OpsHTTPHandlers) registerDashboardRoutes(mux *http.ServeMux) {
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
	mux.HandleFunc("GET /api/v1/ops/dashboard/summary", limit(perm("shards:read", ops.getDashboardSummary)))
	if ops.OpsReader != nil {
		mux.HandleFunc("GET /api/v1/ops/domains/rotation", limit(perm("settings:read", ops.listDomainRotation)))
	}
	mux.HandleFunc("GET /api/v1/ops/dashboard/metrics", limit(perm("shards:read", ops.getDashboardMetrics)))
	mux.HandleFunc("GET /api/v1/ops/dashboard/stream", limit(perm("shards:read", ops.streamDashboard)))
}

func (ops *OpsHTTPHandlers) getDashboardSummary(w http.ResponseWriter, r *http.Request) {
	summary, err := ops.OpsReader.GetDashboardSummary(r.Context())
	if err != nil {
		ops.writeServiceError(w, err)
		return
	}
	httpresponse.JSON(w, http.StatusOK, summary)
}

func (ops *OpsHTTPHandlers) listDomainRotation(w http.ResponseWriter, r *http.Request) {
	result, err := ops.OpsReader.ListDomainRotation(r.Context())
	if err != nil {
		ops.writeServiceError(w, err)
		return
	}
	httpresponse.JSON(w, http.StatusOK, result)
}

func (ops *OpsHTTPHandlers) getDashboardMetrics(w http.ResponseWriter, r *http.Request) {
	rangeHours := 24
	if raw := r.URL.Query().Get("range"); raw != "" {
		if len(raw) >= 2 && raw[len(raw)-1] == 'h' {
			if n, err := strconv.Atoi(raw[:len(raw)-1]); err == nil && n > 0 {
				rangeHours = n
			}
		}
	}
	metricName := r.URL.Query().Get("name")
	metrics, err := ops.OpsReader.GetDashboardMetrics(r.Context(), rangeHours, metricName)
	if err != nil {
		ops.writeServiceError(w, err)
		return
	}
	httpresponse.JSON(w, http.StatusOK, metrics)
}

func (ops *OpsHTTPHandlers) streamDashboard(w http.ResponseWriter, r *http.Request) {
	flusher, ok := w.(http.Flusher)
	if !ok {
		httpresponse.Error(w, http.StatusInternalServerError, "STREAM_UNSUPPORTED", "streaming not supported")
		return
	}
	ctx := r.Context()
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")

	ticker := time.NewTicker(10 * time.Second)
	defer ticker.Stop()

	writeEvent := func() bool {
		summary, err := ops.OpsReader.GetDashboardSummary(ctx)
		if err != nil {
			return true
		}
		payload, err := json.Marshal(map[string]any{
			"generated_at": summary.GeneratedAt,
			"data":         summary,
		})
		if err != nil {
			return true
		}
		if _, err := fmt.Fprintf(w, "event: dashboard\ndata: %s\n\n", payload); err != nil {
			return false
		}
		flusher.Flush()
		return true
	}

	if !writeEvent() {
		return
	}

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			if !writeEvent() {
				return
			}
		}
	}
}

func (ops *OpsHTTPHandlers) registerMLModelRoutes(mux *http.ServeMux) {
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
	mux.HandleFunc("GET /api/v1/ops/ml-model", limit(perm("shards:read", ops.getMLModelStatus)))
	mux.HandleFunc("GET /api/v1/ops/ml-model/eval", limit(perm("shards:read", ops.getMLEvalReport)))
	mux.HandleFunc("GET /api/v1/ops/ml-model/labels", limit(perm("shards:read", ops.listMLManualLabels)))
	mux.HandleFunc("POST /api/v1/ops/ml-model/labels", limit(perm("shards:write", ops.postMLManualLabel)))
}

func (ops *OpsHTTPHandlers) getMLModelStatus(w http.ResponseWriter, r *http.Request) {
	status, err := ops.OpsReader.GetMLModelStatus(r.Context())
	if err != nil {
		ops.writeServiceError(w, err)
		return
	}
	httpresponse.JSON(w, http.StatusOK, status)
}

func (ops *OpsHTTPHandlers) getMLEvalReport(w http.ResponseWriter, r *http.Request) {
	report, err := ops.OpsReader.GetMLEvalReport(r.Context())
	if err != nil {
		ops.writeServiceError(w, err)
		return
	}
	httpresponse.JSON(w, http.StatusOK, report)
}

func (ops *OpsHTTPHandlers) listMLManualLabels(w http.ResponseWriter, r *http.Request) {
	labels, err := ops.OpsReader.ListMLManualLabels(r.Context())
	if err != nil {
		ops.writeServiceError(w, err)
		return
	}
	if labels == nil {
		labels = []MLManualLabelDTO{}
	}
	httpresponse.JSON(w, http.StatusOK, labels)
}

func (ops *OpsHTTPHandlers) postMLManualLabel(w http.ResponseWriter, r *http.Request) {
	body, err := coldpath.ReadLimitedBody(w, r, coldpath.DefaultMaxBody)
	if err != nil {
		return
	}
	req, err := coldpath.DecodeBody[MLManualLabelRequest](body)
	if err != nil {
		httpresponse.Error(w, http.StatusBadRequest, "BAD_REQUEST", "invalid request body")
		return
	}
	if req.IPHash == "" {
		httpresponse.Error(w, http.StatusBadRequest, "BAD_REQUEST", "ip_hash required")
		return
	}
	if !validMLIPHashHex(req.IPHash) {
		httpresponse.Error(w, http.StatusBadRequest, "BAD_REQUEST", "ip_hash must be 32 hex characters")
		return
	}
	if req.Label != 0 && req.Label != 1 {
		httpresponse.Error(w, http.StatusBadRequest, "BAD_REQUEST", "label must be 0 or 1")
		return
	}
	if err := ops.OpsReader.AddMLManualLabel(r.Context(), req.IPHash, req.Label, req.Reason); err != nil {
		ops.writeServiceError(w, err)
		return
	}
	w.WriteHeader(http.StatusAccepted)
}

func validMLIPHashHex(s string) bool {
	if len(s) != 32 {
		return false
	}
	for _, c := range s {
		if unicode.IsDigit(c) {
			continue
		}
		if (c >= 'a' && c <= 'f') || (c >= 'A' && c <= 'F') {
			continue
		}
		return false
	}
	return true
}

func (ops *OpsHTTPHandlers) registerSupportBundleRoutes(mux *http.ServeMux) {
	if ops == nil || ops.SupportBundle == nil {
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
	mux.HandleFunc("POST /api/v1/ops/support/bundle", limit(perm("ops:write", ops.postSupportBundle)))
}

func (ops *OpsHTTPHandlers) postSupportBundle(w http.ResponseWriter, r *http.Request) {
	if ops.SupportBundle == nil {
		httpresponse.Error(w, http.StatusServiceUnavailable, "BUNDLE_UNAVAILABLE", "support bundle not configured")
		return
	}
	ctx, cancel := context.WithTimeout(r.Context(), supportbundle.DefaultTimeout)
	defer cancel()

	w.Header().Set("Content-Type", "application/gzip")
	w.Header().Set("Content-Disposition", `attachment; filename="ad-event-processor-support-bundle.tar.gz"`)
	w.Header().Set("Cache-Control", "no-store")

	if err := ops.SupportBundle.WriteSupportBundle(ctx, w); err != nil {
		ops.writeServiceError(w, err)
		return
	}
}

const (
	defaultFanOutMaxConcurrency = 8
	defaultFanOutPerSourceTO    = 2 * time.Second
)

type FanOutSourceError struct {
	Source string `json:"source"`
	Code   string `json:"code"`
}

type FanOutResult[T any] struct {
	Items      []T                 `json:"items"`
	Partial    bool                `json:"partial"`
	Errors     []FanOutSourceError `json:"errors,omitempty"`
	NextCursor string              `json:"next_cursor,omitempty"`
}

type FanOutSource[T any] struct {
	ID   string
	Poll func(ctx context.Context) ([]T, error)
}

type FanOutCollector struct {
	maxConcurrency int
	perSourceTO    time.Duration
	route          string
}

func NewFanOutCollector(cfg *config.Config, route string) *FanOutCollector {
	maxConcurrency := defaultFanOutMaxConcurrency
	if cfg != nil && cfg.Management.AdminFanoutMaxConcurrency > 0 {
		maxConcurrency = cfg.Management.AdminFanoutMaxConcurrency
	}
	return &FanOutCollector{
		maxConcurrency: maxConcurrency,
		perSourceTO:    defaultFanOutPerSourceTO,
		route:          route,
	}
}

type fanOutResultSlot[T any] struct {
	sourceID string
	items    []T
	err      error
}

func CollectFanOut[T any](ctx context.Context, c *FanOutCollector, sources []FanOutSource[T]) FanOutResult[T] {
	start := time.Now()
	defer func() {
		if c != nil && c.route != "" {
			metrics.AdminFanoutLatencySeconds.WithLabelValues(c.route).Observe(time.Since(start).Seconds())
		}
	}()

	if len(sources) == 0 {
		return FanOutResult[T]{Items: []T{}}
	}
	if c == nil {
		c = NewFanOutCollector(nil, "")
	}

	sem := make(chan struct{}, c.maxConcurrency)
	slots := make([]fanOutResultSlot[T], len(sources))
	var wg sync.WaitGroup

	for i, src := range sources {
		wg.Add(1)
		go func(idx int, source FanOutSource[T]) {
			defer wg.Done()
			sem <- struct{}{}
			defer func() { <-sem }()

			srcCtx, cancel := context.WithTimeout(ctx, c.perSourceTO)
			defer cancel()

			items, err := source.Poll(srcCtx)
			slots[idx] = fanOutResultSlot[T]{sourceID: source.ID, items: items, err: err}
		}(i, src)
	}
	wg.Wait()

	var (
		out    FanOutResult[T]
		ok     int
		failed int
	)
	for _, slot := range slots {
		if slot.err != nil {
			failed++
			code := "SOURCE_UNAVAILABLE"
			if errors.Is(slot.err, context.DeadlineExceeded) || errors.Is(slot.err, context.Canceled) {
				code = "TIMEOUT"
			}
			out.Errors = append(out.Errors, FanOutSourceError{Source: slot.sourceID, Code: code})
			continue
		}
		ok++
		if len(slot.items) > 0 {
			out.Items = append(out.Items, slot.items...)
		}
	}

	if failed > 0 && ok > 0 {
		out.Partial = true
	}
	if c.route != "" {
		metrics.AdminFanoutSourcesTotal.WithLabelValues(c.route).Add(float64(len(sources)))
		if out.Partial {
			metrics.AdminFanoutPartialTotal.WithLabelValues(c.route).Inc()
		}
	}
	return out
}

type fanOutCursorState struct {
	Sources map[string]string `json:"sources"`
}

func EncodeFanOutCursor(state map[string]string) (string, error) {
	if len(state) == 0 {
		return "", nil
	}
	raw, err := json.Marshal(fanOutCursorState{Sources: state})
	if err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(raw), nil
}

func DecodeFanOutCursor(raw string) (map[string]string, error) {
	if raw == "" {
		return map[string]string{}, nil
	}
	decoded, err := base64.RawURLEncoding.DecodeString(raw)
	if err != nil {
		return nil, fmt.Errorf("invalid cursor")
	}
	var state fanOutCursorState
	if err := json.Unmarshal(decoded, &state); err != nil {
		return nil, fmt.Errorf("invalid cursor")
	}
	if state.Sources == nil {
		state.Sources = map[string]string{}
	}
	return state.Sources, nil
}
