package opsadmin

import (
	"bytes"
	"net/http"
	"strconv"
	"time"

	"ad-event-processor/pkg/coldpath"
	"ad-event-processor/pkg/httpresponse"
)

func (h *HTTPHandlers) exportAudit(w http.ResponseWriter, r *http.Request) {
	if r.URL.Query().Get("format") != "csv" {
		httpresponse.Error(w, http.StatusBadRequest, "BAD_REQUEST", "format must be csv")
		return
	}

	customerRaw := r.URL.Query().Get("customer_id")
	if customerRaw != "" && h.AuthorizeCustomerAccess != nil {
		if err := h.AuthorizeCustomerAccess(r, customerRaw); err != nil {
			h.writeServiceError(w, err)
			return
		}
	}

	cursor := r.URL.Query().Get("cursor")
	redactPII := r.URL.Query().Get("redact_pii") == "true"
	w.Header().Set("Content-Type", "text/csv; charset=utf-8")
	w.Header().Set("Cache-Control", "no-store")

	var buf bytes.Buffer
	result, err := h.OpsReader.ExportAuditCSV(r.Context(), cursor, redactPII, &buf)
	if err != nil {
		h.writeServiceError(w, err)
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

func (h *HTTPHandlers) listCustomerPayments(w http.ResponseWriter, r *http.Request) {
	idStr := r.PathValue("id")
	if h.AuthorizeCustomerAccess != nil {
		if err := h.AuthorizeCustomerAccess(r, idStr); err != nil {
			h.writeServiceError(w, err)
			return
		}
	}

	if h.PaymentIntents == nil {
		httpresponse.Error(w, http.StatusServiceUnavailable, "PAYMENT_UNAVAILABLE", "payment service not configured")
		return
	}

	limit, offset := coldpath.ParseAPIPagination(r)
	resp, err := h.PaymentIntents.ListPaymentIntents(r.Context(), idStr, limit, offset)
	if err != nil {
		h.writeServiceError(w, err)
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
		if ledgerID, lerr := h.OpsReader.LookupLedgerIDForPaymentIntent(r.Context(), intent.ID); lerr == nil {
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

func (h *HTTPHandlers) registerAuditRoutes(mux *http.ServeMux) {
	if h.AuditLister == nil {
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
	mux.HandleFunc("GET /api/v1/audit", limit(perm("audit:read", h.listAudit)))
}

func (h *HTTPHandlers) listAudit(w http.ResponseWriter, r *http.Request) {
	limit, offset := coldpath.ParseAPIPagination(r)
	redact := r.URL.Query().Get("redact_pii") == "true"
	logs, total, err := h.AuditLister.ListAuditLogs(r.Context(), limit, offset, redact)
	if err != nil {
		h.writeServiceError(w, err)
		return
	}
	w.Header().Set("X-Total-Count", strconv.FormatInt(total, 10))
	httpresponse.JSON(w, http.StatusOK, logs)
}
