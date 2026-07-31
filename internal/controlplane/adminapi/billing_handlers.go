package adminapi

import (
	"errors"
	"espx/internal/billing"
	"net/http"
	"strconv"
	"time"

	"espx/pkg/coldpath"

	"espx/pkg/httpresponse"

	"github.com/google/uuid"
)

type BillingHTTPHandlers struct {
	Billing                 billing.BillingAPI
	InProcessInvoices       InProcessInvoiceService
	CompositeReads          *CompositeReadService
	InvoiceDelivery         InvoiceRetryer
	VoidAuditor             VoidAuditor
	ApplyRateLimit          func(http.HandlerFunc) http.HandlerFunc
	RequirePermission       func(string, http.HandlerFunc) http.HandlerFunc
	AuthorizeCustomerAccess func(*http.Request, string) error
	WriteServiceError       func(http.ResponseWriter, error)
	RequestIsFromAdmin      func(*http.Request) bool

	ApplySelfServeRateLimit    func(http.HandlerFunc) http.HandlerFunc
	RequireSelfServePermission func(string, http.HandlerFunc) http.HandlerFunc
	ResolveSelfServeCustomerID func(*http.Request) (uuid.UUID, error)

	CustomerBalance              CustomerBalanceReader
	Disputes                     DisputeLister
	LimitExportByCustomer        func(http.HandlerFunc) http.HandlerFunc
	ResolveDisputeCustomerFilter func(*http.Request) (string, error)
}

func (billHandlers *BillingHTTPHandlers) Register(mux *http.ServeMux) {
	if billHandlers == nil {
		return
	}
	limit := billHandlers.ApplyRateLimit
	perm := billHandlers.RequirePermission
	if limit == nil {
		limit = func(next http.HandlerFunc) http.HandlerFunc { return next }
	}
	if perm == nil {
		perm = func(_ string, next http.HandlerFunc) http.HandlerFunc { return next }
	}

	mux.HandleFunc("GET /api/v1/billing/invoices", limit(perm("customers:read", billHandlers.listInvoices)))
	mux.HandleFunc("GET /api/v1/billing/invoices/{id}", limit(perm("customers:read", billHandlers.getInvoice)))
	mux.HandleFunc("GET /api/v1/billing/invoices/{id}/pdf", limit(perm("customers:read", billHandlers.getInvoicePDF)))
	mux.HandleFunc("GET /api/v1/customers/{id}/billing/statement", limit(perm("customers:read", billHandlers.getStatement)))
	mux.HandleFunc("GET /api/v1/billing/invoices/{id}/ledger-lines", limit(perm("customers:read", billHandlers.getLedgerLines)))
	mux.HandleFunc("POST /api/v1/billing/invoices/preview", limit(perm("customers:read", billHandlers.previewInvoice)))
	mux.HandleFunc("GET /api/v1/customers/{id}/wallet", limit(perm("customers:read", billHandlers.getWallet)))
	mux.HandleFunc("GET /api/v1/billing/invoices/{id}/deliveries", limit(perm("customers:read", billHandlers.listDeliveries)))
	mux.HandleFunc("POST /api/v1/billing/invoices/{id}/deliveries/retry", limit(perm("customers:write", billHandlers.retryDelivery)))
	mux.HandleFunc("GET /api/v1/billing/invariant", limit(perm("customers:read", billHandlers.getInvariant)))
	mux.HandleFunc("GET /api/v1/billing/summary", limit(perm("shards:read", billHandlers.getSummary)))
	mux.HandleFunc("GET /api/v1/customers/{id}/tax-profile", limit(perm("customers:read", billHandlers.getTaxProfile)))
	mux.HandleFunc("PUT /api/v1/customers/{id}/tax-profile", limit(perm("customers:write", billHandlers.putTaxProfile)))
	mux.HandleFunc("POST /api/v1/billing/invoices/{id}/void", limit(perm("customers:write", billHandlers.voidInvoice)))
	mux.HandleFunc("GET /api/v1/customers/{id}/billing/forecast", limit(perm("customers:read", billHandlers.getForecast)))

	if billHandlers.RequireSelfServePermission != nil && billHandlers.ResolveSelfServeCustomerID != nil {
		ssLimit := billHandlers.ApplySelfServeRateLimit
		if ssLimit == nil {
			ssLimit = limit
		}
		mux.HandleFunc("GET /api/v1/selfserve/billing/statement", ssLimit(billHandlers.RequireSelfServePermission("customers:read", billHandlers.getSelfServeStatement)))
	}

	billHandlers.registerBalanceRoutes(mux)
	billHandlers.registerDisputeRoutes(mux)
}

func (billHandlers *BillingHTTPHandlers) listInvoices(w http.ResponseWriter, r *http.Request) {
	if billHandlers.Billing == nil {
		httpresponse.Error(w, http.StatusServiceUnavailable, "BILLING_UNAVAILABLE", "billing service not configured")
		return
	}

	customerRaw := r.URL.Query().Get("customer_id")
	adminList := billHandlers.RequestIsFromAdmin != nil && billHandlers.RequestIsFromAdmin(r) && customerRaw == ""
	if !adminList {
		if customerRaw == "" {
			httpresponse.Error(w, http.StatusBadRequest, "BAD_REQUEST", "customer_id is required")
			return
		}
		if err := billHandlers.authorizeCustomerAccess(r, customerRaw); err != nil {
			billHandlers.writeServiceError(w, err)
			return
		}
	} else if customerRaw != "" {
		if err := billHandlers.authorizeCustomerAccess(r, customerRaw); err != nil {
			billHandlers.writeServiceError(w, err)
			return
		}
	}

	limit, offset := parsePagination(r)
	if adminList {
		billHandlers.listInvoicesAdmin(w, r, limit, offset)
		return
	}

	resp, err := billHandlers.Billing.ListInvoices(r.Context(), customerRaw, limit, offset)
	if err != nil {
		WriteBillingError(w, err)
		return
	}
	items := make([]map[string]any, 0, len(resp.Invoices))
	for i := range resp.Invoices {
		items = append(items, invoiceToJSON(&resp.Invoices[i]))
	}
	httpresponse.JSON(w, http.StatusOK, map[string]any{
		"items":  items,
		"total":  resp.Total,
		"limit":  limit,
		"offset": offset,
	})
}

func (billHandlers *BillingHTTPHandlers) listInvoicesAdmin(w http.ResponseWriter, r *http.Request, limit, offset int32) {
	if billHandlers.CompositeReads == nil {
		httpresponse.Error(w, http.StatusServiceUnavailable, "BILLING_UNAVAILABLE", "billing composite reads not configured")
		return
	}
	filters := AdminInvoiceFilters{}
	if raw := r.URL.Query().Get("customer_id"); raw != "" {
		id, err := uuid.Parse(raw)
		if err != nil {
			httpresponse.Error(w, http.StatusBadRequest, "BAD_REQUEST", "invalid customer_id")
			return
		}
		filters.CustomerID = &id
	}
	if raw := r.URL.Query().Get("month"); raw != "" {
		month, err := time.Parse("2006-01", raw)
		if err != nil {
			httpresponse.Error(w, http.StatusBadRequest, "BAD_REQUEST", "month must be YYYY-MM")
			return
		}
		filters.Month = &month
	}
	filters.Status = r.URL.Query().Get("status")
	if raw := r.URL.Query().Get("min_total"); raw != "" {
		n, err := strconv.ParseInt(raw, 10, 64)
		if err != nil || n < 0 {
			httpresponse.Error(w, http.StatusBadRequest, "BAD_REQUEST", "invalid min_total")
			return
		}
		filters.MinTotal = n
	}
	result, err := billHandlers.CompositeReads.ListInvoicesAdmin(r.Context(), filters, limit, offset)
	if err != nil {
		billHandlers.writeServiceError(w, err)
		return
	}
	httpresponse.JSON(w, http.StatusOK, result)
}

func (billHandlers *BillingHTTPHandlers) getForecast(w http.ResponseWriter, r *http.Request) {
	if billHandlers.CompositeReads == nil {
		httpresponse.Error(w, http.StatusServiceUnavailable, "BILLING_UNAVAILABLE", "billing composite reads not configured")
		return
	}
	customerID, err := uuid.Parse(r.PathValue("id"))
	if err != nil {
		httpresponse.Error(w, http.StatusBadRequest, "BAD_REQUEST", "invalid customer id")
		return
	}
	if err := billHandlers.authorizeCustomerAccess(r, customerID.String()); err != nil {
		billHandlers.writeServiceError(w, err)
		return
	}
	forecast, err := billHandlers.CompositeReads.BuildForecast(r.Context(), customerID)
	if err != nil {
		billHandlers.writeServiceError(w, err)
		return
	}
	httpresponse.JSON(w, http.StatusOK, forecast)
}

func (billHandlers *BillingHTTPHandlers) getSelfServeStatement(w http.ResponseWriter, r *http.Request) {
	if billHandlers.CompositeReads == nil || billHandlers.ResolveSelfServeCustomerID == nil {
		httpresponse.Error(w, http.StatusServiceUnavailable, "BILLING_UNAVAILABLE", "billing not configured")
		return
	}
	customerID, err := billHandlers.ResolveSelfServeCustomerID(r)
	if err != nil {
		billHandlers.writeServiceError(w, err)
		return
	}
	from, to, err := ParseStatementPeriod(r.URL.Query().Get("from"), r.URL.Query().Get("to"), r.URL.Query().Get("month"))
	if err != nil {
		httpresponse.Error(w, http.StatusBadRequest, "BAD_REQUEST", err.Error())
		return
	}
	stmt, err := billHandlers.CompositeReads.BuildStatement(r.Context(), customerID, from, to)
	if err != nil {
		billHandlers.writeServiceError(w, err)
		return
	}
	httpresponse.JSON(w, http.StatusOK, stmt)
}

func (billHandlers *BillingHTTPHandlers) getInvoice(w http.ResponseWriter, r *http.Request) {
	if billHandlers.Billing == nil {
		httpresponse.Error(w, http.StatusServiceUnavailable, "BILLING_UNAVAILABLE", "billing service not configured")
		return
	}
	invoiceID := r.PathValue("id")
	if _, err := uuid.Parse(invoiceID); err != nil {
		httpresponse.Error(w, http.StatusBadRequest, "BAD_REQUEST", "invalid invoice id")
		return
	}
	invoice, err := billHandlers.Billing.GetInvoice(r.Context(), invoiceID)
	if err != nil {
		WriteBillingError(w, err)
		return
	}
	if err := billHandlers.authorizeCustomerAccess(r, invoice.CustomerID); err != nil {
		billHandlers.writeServiceError(w, err)
		return
	}
	body := invoiceToJSON(invoice)
	body["pdf_url"] = invoicePDFPath(invoiceID)
	httpresponse.JSON(w, http.StatusOK, body)
}

func (billHandlers *BillingHTTPHandlers) getInvoicePDF(w http.ResponseWriter, r *http.Request) {
	if billHandlers.Billing == nil {
		httpresponse.Error(w, http.StatusServiceUnavailable, "BILLING_UNAVAILABLE", "billing service not configured")
		return
	}
	invoiceID := r.PathValue("id")
	if _, err := uuid.Parse(invoiceID); err != nil {
		httpresponse.Error(w, http.StatusBadRequest, "BAD_REQUEST", "invalid invoice id")
		return
	}
	invoice, err := billHandlers.Billing.GetInvoice(r.Context(), invoiceID)
	if err != nil {
		WriteBillingError(w, err)
		return
	}
	if err := billHandlers.authorizeCustomerAccess(r, invoice.CustomerID); err != nil {
		billHandlers.writeServiceError(w, err)
		return
	}
	pdf := billing.RenderInvoicePDF(invoice)
	if len(pdf) == 0 {
		httpresponse.Error(w, http.StatusInternalServerError, "INTERNAL", "failed to render invoice pdf")
		return
	}
	w.Header().Set("Content-Type", "application/pdf")
	w.Header().Set("Cache-Control", "no-store")
	if _, err := w.Write(pdf); err != nil {
		return
	}
}

func (billHandlers *BillingHTTPHandlers) getStatement(w http.ResponseWriter, r *http.Request) {
	if billHandlers.CompositeReads == nil {
		httpresponse.Error(w, http.StatusServiceUnavailable, "BILLING_UNAVAILABLE", "billing composite reads not configured")
		return
	}
	customerID, err := uuid.Parse(r.PathValue("id"))
	if err != nil {
		httpresponse.Error(w, http.StatusBadRequest, "BAD_REQUEST", "invalid customer id")
		return
	}
	if err := billHandlers.authorizeCustomerAccess(r, customerID.String()); err != nil {
		billHandlers.writeServiceError(w, err)
		return
	}
	from, to, err := ParseStatementPeriod(r.URL.Query().Get("from"), r.URL.Query().Get("to"), r.URL.Query().Get("month"))
	if err != nil {
		httpresponse.Error(w, http.StatusBadRequest, "BAD_REQUEST", err.Error())
		return
	}
	stmt, err := billHandlers.CompositeReads.BuildStatement(r.Context(), customerID, from, to)
	if err != nil {
		billHandlers.writeServiceError(w, err)
		return
	}
	httpresponse.JSON(w, http.StatusOK, stmt)
}

func (billHandlers *BillingHTTPHandlers) getLedgerLines(w http.ResponseWriter, r *http.Request) {
	if billHandlers.CompositeReads == nil || billHandlers.Billing == nil {
		httpresponse.Error(w, http.StatusServiceUnavailable, "BILLING_UNAVAILABLE", "billing not configured")
		return
	}
	invoiceID := r.PathValue("id")
	invoice, err := billHandlers.Billing.GetInvoice(r.Context(), invoiceID)
	if err != nil {
		WriteBillingError(w, err)
		return
	}
	if err := billHandlers.authorizeCustomerAccess(r, invoice.CustomerID); err != nil {
		billHandlers.writeServiceError(w, err)
		return
	}
	customerID, _ := uuid.Parse(invoice.CustomerID)
	month := invoice.BillingMonth.UTC()
	if month.IsZero() {
		month = time.Now().UTC()
	}
	var cursorID int64
	if c := r.URL.Query().Get("cursor"); c != "" {
		cursorID, _ = strconv.ParseInt(c, 10, 64)
	}
	limit, _ := parsePagination(r)
	lines, nextCursor, total, err := billHandlers.CompositeReads.ListLedgerLines(r.Context(), customerID, month, cursorID, limit)
	if err != nil {
		billHandlers.writeServiceError(w, err)
		return
	}
	httpresponse.JSON(w, http.StatusOK, map[string]any{
		"items":       lines,
		"total":       total,
		"next_cursor": nextCursor,
		"limit":       limit,
	})
}

type previewRequest struct {
	CustomerID   string `json:"customer_id"`
	BillingMonth string `json:"billing_month"`
}

func (billHandlers *BillingHTTPHandlers) previewInvoice(w http.ResponseWriter, r *http.Request) {
	if billHandlers.InProcessInvoices == nil {
		httpresponse.Error(w, http.StatusServiceUnavailable, "BILLING_UNAVAILABLE", "billing service not configured")
		return
	}
	body, err := coldpath.ReadLimitedBody(w, r, coldpath.DefaultMaxBody)
	if err != nil {
		return
	}
	req, err := coldpath.DecodeBody[previewRequest](body)
	if err != nil {
		httpresponse.Error(w, http.StatusBadRequest, "BAD_REQUEST", "invalid request body")
		return
	}
	if req.CustomerID == "" {
		httpresponse.Error(w, http.StatusBadRequest, "BAD_REQUEST", "customer_id is required")
		return
	}
	if err := billHandlers.authorizeCustomerAccess(r, req.CustomerID); err != nil {
		billHandlers.writeServiceError(w, err)
		return
	}
	customerID, err := uuid.Parse(req.CustomerID)
	if err != nil {
		httpresponse.Error(w, http.StatusBadRequest, "BAD_REQUEST", "invalid customer_id")
		return
	}
	month, err := time.Parse("2006-01", req.BillingMonth)
	if err != nil {
		httpresponse.Error(w, http.StatusBadRequest, "BAD_REQUEST", "billing_month must be YYYY-MM")
		return
	}
	preview, err := billHandlers.InProcessInvoices.PreviewInvoice(r.Context(), customerID, month)
	if err != nil {
		writeBillingLocalError(w, err)
		return
	}
	httpresponse.JSON(w, http.StatusOK, preview)
}

func (billHandlers *BillingHTTPHandlers) getWallet(w http.ResponseWriter, r *http.Request) {
	if billHandlers.CompositeReads == nil {
		httpresponse.Error(w, http.StatusServiceUnavailable, "BILLING_UNAVAILABLE", "billing composite reads not configured")
		return
	}
	customerID, err := uuid.Parse(r.PathValue("id"))
	if err != nil {
		httpresponse.Error(w, http.StatusBadRequest, "BAD_REQUEST", "invalid customer id")
		return
	}
	if err := billHandlers.authorizeCustomerAccess(r, customerID.String()); err != nil {
		billHandlers.writeServiceError(w, err)
		return
	}
	wallet, err := billHandlers.CompositeReads.GetWallet(r.Context(), customerID)
	if err != nil {
		writeBillingLocalError(w, err)
		return
	}
	httpresponse.JSON(w, http.StatusOK, wallet)
}

func (billHandlers *BillingHTTPHandlers) listDeliveries(w http.ResponseWriter, r *http.Request) {
	if billHandlers.CompositeReads == nil || billHandlers.Billing == nil {
		httpresponse.Error(w, http.StatusServiceUnavailable, "BILLING_UNAVAILABLE", "billing not configured")
		return
	}
	invoiceID := r.PathValue("id")
	invoice, err := billHandlers.Billing.GetInvoice(r.Context(), invoiceID)
	if err != nil {
		WriteBillingError(w, err)
		return
	}
	if err := billHandlers.authorizeCustomerAccess(r, invoice.CustomerID); err != nil {
		billHandlers.writeServiceError(w, err)
		return
	}
	rows, err := billHandlers.CompositeReads.ListDeliveries(r.Context(), invoiceID)
	if err != nil {
		billHandlers.writeServiceError(w, err)
		return
	}
	httpresponse.JSON(w, http.StatusOK, map[string]any{"items": rows})
}

func (billHandlers *BillingHTTPHandlers) retryDelivery(w http.ResponseWriter, r *http.Request) {
	if billHandlers.Billing == nil || billHandlers.InvoiceDelivery == nil {
		httpresponse.Error(w, http.StatusServiceUnavailable, "BILLING_UNAVAILABLE", "invoice retry not configured")
		return
	}
	idempotencyKey := r.Header.Get("Idempotency-Key")
	if idempotencyKey == "" {
		httpresponse.Error(w, http.StatusBadRequest, "BAD_REQUEST", "Idempotency-Key header is required")
		return
	}
	invoiceID := r.PathValue("id")
	invoice, err := billHandlers.Billing.GetInvoice(r.Context(), invoiceID)
	if err != nil {
		WriteBillingError(w, err)
		return
	}
	if err := billHandlers.authorizeCustomerAccess(r, invoice.CustomerID); err != nil {
		billHandlers.writeServiceError(w, err)
		return
	}
	if err := billHandlers.InvoiceDelivery.RetryInvoiceDelivery(r.Context(), invoice, idempotencyKey); err != nil {
		billHandlers.writeServiceError(w, err)
		return
	}
	w.WriteHeader(http.StatusAccepted)
}

func (billHandlers *BillingHTTPHandlers) getInvariant(w http.ResponseWriter, r *http.Request) {
	if billHandlers.CompositeReads == nil {
		httpresponse.Error(w, http.StatusServiceUnavailable, "BILLING_UNAVAILABLE", "billing composite reads not configured")
		return
	}
	var customerID *uuid.UUID
	if raw := r.URL.Query().Get("customer_id"); raw != "" {
		if err := billHandlers.authorizeCustomerAccess(r, raw); err != nil {
			billHandlers.writeServiceError(w, err)
			return
		}
		id, err := uuid.Parse(raw)
		if err != nil {
			httpresponse.Error(w, http.StatusBadRequest, "BAD_REQUEST", "invalid customer_id")
			return
		}
		customerID = &id
	} else if billHandlers.RequestIsFromAdmin == nil || !billHandlers.RequestIsFromAdmin(r) {
		httpresponse.Error(w, http.StatusForbidden, "FORBIDDEN", "customer_id required for tenant users")
		return
	}
	result, err := billHandlers.CompositeReads.GetInvariant(r.Context(), customerID)
	if err != nil {
		billHandlers.writeServiceError(w, err)
		return
	}
	httpresponse.JSON(w, http.StatusOK, result)
}

func (billHandlers *BillingHTTPHandlers) getSummary(w http.ResponseWriter, r *http.Request) {
	if billHandlers.CompositeReads == nil {
		httpresponse.Error(w, http.StatusServiceUnavailable, "BILLING_UNAVAILABLE", "billing composite reads not configured")
		return
	}
	if billHandlers.RequestIsFromAdmin != nil && !billHandlers.RequestIsFromAdmin(r) {
		httpresponse.Error(w, http.StatusForbidden, "FORBIDDEN", "admin only")
		return
	}
	summary, err := billHandlers.CompositeReads.GetSummary(r.Context())
	if err != nil {
		billHandlers.writeServiceError(w, err)
		return
	}
	httpresponse.JSON(w, http.StatusOK, summary)
}

func (billHandlers *BillingHTTPHandlers) getTaxProfile(w http.ResponseWriter, r *http.Request) {
	if billHandlers.CompositeReads == nil {
		httpresponse.Error(w, http.StatusServiceUnavailable, "BILLING_UNAVAILABLE", "billing composite reads not configured")
		return
	}
	customerID, err := uuid.Parse(r.PathValue("id"))
	if err != nil {
		httpresponse.Error(w, http.StatusBadRequest, "BAD_REQUEST", "invalid customer id")
		return
	}
	if err := billHandlers.authorizeCustomerAccess(r, customerID.String()); err != nil {
		billHandlers.writeServiceError(w, err)
		return
	}
	profile, err := billHandlers.CompositeReads.GetTaxProfile(r.Context(), customerID)
	if err != nil {
		billHandlers.writeServiceError(w, err)
		return
	}
	httpresponse.JSON(w, http.StatusOK, profile)
}

func (billHandlers *BillingHTTPHandlers) putTaxProfile(w http.ResponseWriter, r *http.Request) {
	if billHandlers.CompositeReads == nil {
		httpresponse.Error(w, http.StatusServiceUnavailable, "BILLING_UNAVAILABLE", "billing composite reads not configured")
		return
	}
	customerID, err := uuid.Parse(r.PathValue("id"))
	if err != nil {
		httpresponse.Error(w, http.StatusBadRequest, "BAD_REQUEST", "invalid customer id")
		return
	}
	if err := billHandlers.authorizeCustomerAccess(r, customerID.String()); err != nil {
		billHandlers.writeServiceError(w, err)
		return
	}
	body, err := coldpath.ReadLimitedBody(w, r, coldpath.DefaultMaxBody)
	if err != nil {
		return
	}
	dto, err := coldpath.DecodeBody[TaxProfileDTO](body)
	if err != nil {
		httpresponse.Error(w, http.StatusBadRequest, "BAD_REQUEST", "invalid request body")
		return
	}
	profile, err := billHandlers.CompositeReads.UpsertTaxProfile(r.Context(), customerID, dto)
	if err != nil {
		billHandlers.writeServiceError(w, err)
		return
	}
	httpresponse.JSON(w, http.StatusOK, profile)
}

func (billHandlers *BillingHTTPHandlers) voidInvoice(w http.ResponseWriter, r *http.Request) {
	if billHandlers.InProcessInvoices == nil {
		httpresponse.Error(w, http.StatusServiceUnavailable, "BILLING_UNAVAILABLE", "billing service not configured")
		return
	}
	invoiceID, err := uuid.Parse(r.PathValue("id"))
	if err != nil {
		httpresponse.Error(w, http.StatusBadRequest, "BAD_REQUEST", "invalid invoice id")
		return
	}
	var customerID string
	if billHandlers.Billing != nil {
		if inv, gerr := billHandlers.Billing.GetInvoice(r.Context(), invoiceID.String()); gerr == nil {
			customerID = inv.CustomerID
			if err := billHandlers.authorizeCustomerAccess(r, customerID); err != nil {
				billHandlers.writeServiceError(w, err)
				return
			}
		}
	}
	if err := billHandlers.InProcessInvoices.VoidInvoice(r.Context(), invoiceID); err != nil {
		writeBillingLocalError(w, err)
		return
	}
	if billHandlers.VoidAuditor != nil && customerID != "" {
		_ = billHandlers.VoidAuditor.AuditInvoiceVoid(r.Context(), invoiceID.String(), customerID)
	}
	w.WriteHeader(http.StatusNoContent)
}

func (billHandlers *BillingHTTPHandlers) authorizeCustomerAccess(r *http.Request, customerID string) error {
	if billHandlers.AuthorizeCustomerAccess == nil {
		return nil
	}
	return billHandlers.AuthorizeCustomerAccess(r, customerID)
}

func (billHandlers *BillingHTTPHandlers) writeServiceError(w http.ResponseWriter, err error) {
	var cur invalidExportCursorError
	if errors.As(err, &cur) {
		httpresponse.Error(w, http.StatusBadRequest, "BAD_REQUEST", string(cur))
		return
	}
	if errors.Is(err, ErrForbidden) {
		httpresponse.Error(w, http.StatusForbidden, "FORBIDDEN", "forbidden")
		return
	}
	if billHandlers.WriteServiceError != nil {
		billHandlers.WriteServiceError(w, err)
		return
	}
	httpresponse.Error(w, http.StatusInternalServerError, "INTERNAL", "request failed")
}

func invoicePDFPath(invoiceID string) string {
	return "/api/v1/billing/invoices/" + invoiceID + "/pdf"
}

func InvoiceToJSON(invoice *billing.Invoice) map[string]any {
	return invoiceToJSON(invoice)
}

func invoiceToJSON(invoice *billing.Invoice) map[string]any {
	if invoice == nil {
		return nil
	}
	month := ""
	if !invoice.BillingMonth.IsZero() {
		month = invoice.BillingMonth.UTC().Format("2006-01")
	}
	lines := make([]map[string]any, 0, len(invoice.Lines))
	for _, line := range invoice.Lines {
		lines = append(lines, map[string]any{
			"ledger_type":  line.LedgerType,
			"amount_micro": line.AmountMicro,
			"entry_count":  line.EntryCount,
		})
	}
	return map[string]any{
		"id":             invoice.ID,
		"customer_id":    invoice.CustomerID,
		"billing_month":  month,
		"subtotal_micro": invoice.SubtotalMicro,
		"tax_micro":      invoice.TaxMicro,
		"total_micro":    invoice.TotalMicro,
		"currency":       invoice.Currency,
		"tax_scheme":     invoice.TaxScheme,
		"tax_rate_bps":   invoice.TaxRateBps,
		"lines":          lines,
	}
}

func parsePagination(r *http.Request) (int32, int32) {
	limit := int32(50)
	offset := int32(0)
	if v := r.URL.Query().Get("limit"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 {
			limit = int32(n)
		}
	}
	if limit > 1000 {
		limit = 1000
	}
	if v := r.URL.Query().Get("offset"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n >= 0 {
			offset = int32(n)
		}
	}
	return limit, offset
}

func writeBillingLocalError(w http.ResponseWriter, err error) {
	if errors.Is(err, billing.ErrCustomerNotFound) || errors.Is(err, billing.ErrInvoiceNotFound) {
		httpresponse.Error(w, http.StatusNotFound, "NOT_FOUND", err.Error())
		return
	}
	if errors.Is(err, billing.ErrLedgerDrift) {
		httpresponse.Error(w, http.StatusConflict, "LEDGER_DRIFT", err.Error())
		return
	}
	WriteBillingError(w, err)
}
