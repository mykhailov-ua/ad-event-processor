package adminapi

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"time"

	"github.com/bidshard/ad-event-processor/internal/config"
	"github.com/bidshard/ad-event-processor/internal/database"
	"github.com/bidshard/ad-event-processor/internal/domain"
	"github.com/bidshard/ad-event-processor/internal/ledger"
	billingdb "github.com/bidshard/ad-event-processor/internal/ledger/db"
	"github.com/bidshard/ad-event-processor/pkg/coldpath"
	"github.com/bidshard/ad-event-processor/pkg/httpresponse"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type BillingHTTPHandlers struct {
	Billing                 domain.BillingAPI
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

type InvoiceListResponse struct {
	Items  []domain.Invoice `json:"items"`
	Total  int64            `json:"total"`
	Limit  int32            `json:"limit"`
	Offset int32            `json:"offset"`
}

type SelfServeInvoiceListResponse struct {
	Invoices []domain.Invoice `json:"invoices"`
	Total    int64            `json:"total"`
}

type CustomerBalanceReader interface {
	GetCustomerBalance(ctx context.Context, customerID uuid.UUID) (CustomerBalanceDTO, error)
	ListCustomerLedger(ctx context.Context, customerID uuid.UUID, limit, offset int32) ([]BalanceLedgerDTO, int64, error)
	ExportCustomerLedgerCSV(ctx context.Context, customerID uuid.UUID, cursor int64, w io.Writer) (LedgerExportResult, error)
}

type LedgerListResponse struct {
	Items  []BalanceLedgerDTO `json:"items"`
	Total  int64              `json:"total"`
	Limit  int32              `json:"limit"`
	Offset int32              `json:"offset"`
}

type DisputeRowDTO struct {
	IntentID                 string  `json:"intent_id"`
	CustomerID               string  `json:"customer_id"`
	AmountMicro              int64   `json:"amount_micro"`
	Currency                 string  `json:"currency"`
	ProviderDisputeID        string  `json:"provider_dispute_id"`
	UpdatedAt                string  `json:"updated_at,omitempty"`
	ChargebackLedgerEntryIDs []int64 `json:"chargeback_ledger_entry_ids"`
}

type DisputeListResult struct {
	Disputes []DisputeRowDTO `json:"disputes"`
	Total    int64           `json:"total"`
}

type DisputeLister interface {
	ListDisputes(ctx context.Context, customerFilter string, limit, offset int32) (DisputeListResult, error)
}

type InProcessInvoiceService interface {
	PreviewInvoice(ctx context.Context, customerID uuid.UUID, billingMonth time.Time) (*ledger.InvoicePreview, error)
	VoidInvoice(ctx context.Context, invoiceID uuid.UUID) error
}

type InvoiceRetryer interface {
	RetryInvoiceDelivery(ctx context.Context, invoice *domain.Invoice, idempotencyKey string) error
}

type VoidAuditor interface {
	AuditInvoiceVoid(ctx context.Context, invoiceID, customerID string) error
}

type AdminInvoiceFilters struct {
	CustomerID *uuid.UUID
	Month      *time.Time
	Status     string
	MinTotal   int64
}

type AdminInvoiceListResult struct {
	Items  []domain.InvoiceSummary `json:"items"`
	Total  int64                   `json:"total"`
	Limit  int32                   `json:"limit"`
	Offset int32                   `json:"offset"`
}

const billingForecastCHTimeout = 1500 * time.Millisecond

type ForecastDTO struct {
	CustomerID               string          `json:"customer_id"`
	Month                    string          `json:"month"`
	LedgerMTDMicro           int64           `json:"ledger_mtd_micro"`
	LedgerRunRateMicroPerDay int64           `json:"ledger_run_rate_micro_per_day"`
	CHHourlyImpressions      []CHHourlyPoint `json:"ch_hourly_impressions,omitempty"`
	ProjectedMonthEndMicro   int64           `json:"projected_month_end_micro"`
	DaysRemaining            int             `json:"days_remaining"`
	LowConfidence            bool            `json:"low_confidence"`
	CHUnavailable            bool            `json:"ch_unavailable,omitempty"`
}

type CHHourlyPoint struct {
	Hour        string `json:"hour"`
	Impressions int64  `json:"impressions"`
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
	httpresponse.JSON(w, http.StatusOK, InvoiceListResponse{
		Items:  resp.Invoices,
		Total:  resp.Total,
		Limit:  limit,
		Offset: offset,
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
	invoice.PDFURL = invoicePDFPath(invoiceID)
	httpresponse.JSON(w, http.StatusOK, invoice)
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
	pdf := ledger.RenderInvoicePDF(invoice)
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
	httpresponse.JSON(w, http.StatusOK, LedgerLinesListResponse{
		Items:      lines,
		Total:      total,
		NextCursor: nextCursor,
		Limit:      limit,
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
	httpresponse.JSON(w, http.StatusOK, DeliveryListResponse{Items: rows})
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
	if errors.Is(err, ledger.ErrCustomerNotFound) || errors.Is(err, ledger.ErrInvoiceNotFound) {
		httpresponse.Error(w, http.StatusNotFound, "NOT_FOUND", err.Error())
		return
	}
	if errors.Is(err, ledger.ErrLedgerDrift) {
		httpresponse.Error(w, http.StatusConflict, "LEDGER_DRIFT", err.Error())
		return
	}
	WriteBillingError(w, err)
}

func (s *CompositeReadService) ListInvoicesAdmin(ctx context.Context, filters AdminInvoiceFilters, limit, offset int32) (AdminInvoiceListResult, error) {
	if s == nil || s.queries == nil {
		return AdminInvoiceListResult{}, fmt.Errorf("composite read service not configured")
	}
	if limit <= 0 {
		limit = 50
	}
	if limit > 1000 {
		limit = 1000
	}

	var customer pgtype.UUID
	if filters.CustomerID != nil {
		customer = pgtype.UUID{Bytes: *filters.CustomerID, Valid: true}
	}
	var month pgtype.Date
	if filters.Month != nil {
		m := filters.Month.UTC()
		month = pgtype.Date{Time: time.Date(m.Year(), m.Month(), m.Day(), 0, 0, 0, 0, time.UTC), Valid: true}
	}

	params := billingdb.ListInvoicesAdminParams{
		Column1: customer,
		Column2: month,
		Column3: filters.Status,
		Column4: filters.MinTotal,
		Limit:   limit,
		Offset:  offset,
	}
	rows, err := s.queries.ListInvoicesAdmin(ctx, params)
	if err != nil {
		return AdminInvoiceListResult{}, err
	}
	total, err := s.queries.CountInvoicesAdmin(ctx, billingdb.CountInvoicesAdminParams{
		Column1: customer,
		Column2: month,
		Column3: filters.Status,
		Column4: filters.MinTotal,
	})
	if err != nil {
		return AdminInvoiceListResult{}, err
	}

	items := make([]domain.InvoiceSummary, 0, len(rows))
	for _, inv := range rows {
		monthStr := ""
		if inv.BillingMonth.Valid {
			monthStr = inv.BillingMonth.Time.UTC().Format("2006-01")
		}
		items = append(items, domain.InvoiceSummary{
			ID:            uuidString(inv.ID),
			CustomerID:    uuidString(inv.CustomerID),
			BillingMonth:  monthStr,
			SubtotalMicro: inv.SubtotalMicro,
			TaxMicro:      inv.TaxMicro,
			TotalMicro:    inv.TotalMicro,
			Status:        string(inv.Status),
			Currency:      inv.Currency,
		})
	}
	return AdminInvoiceListResult{
		Items:  items,
		Total:  total,
		Limit:  limit,
		Offset: offset,
	}, nil
}

func (s *CompositeReadService) WithCHQuery(q *database.CHQuery) *CompositeReadService {
	if s == nil {
		return nil
	}
	s.chQuery = q
	return s
}

func (s *CompositeReadService) BuildForecast(ctx context.Context, customerID uuid.UUID) (ForecastDTO, error) {
	if s == nil || s.pool == nil {
		return ForecastDTO{}, fmt.Errorf("composite read service not configured")
	}

	now := time.Now().UTC()
	monthStart := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, time.UTC)
	monthEnd := monthStart.AddDate(0, 1, 0)
	daysRemaining := int(monthEnd.Sub(now).Hours()/24) + 1
	if daysRemaining < 0 {
		daysRemaining = 0
	}

	pgCustomer := pgtype.UUID{Bytes: customerID, Valid: true}
	mtd, err := s.queries.SumCustomerSpendInWindow(ctx, billingdb.SumCustomerSpendInWindowParams{
		CustomerID:  pgCustomer,
		CreatedAt:   pgTimestamp(monthStart),
		CreatedAt_2: pgTimestamp(now),
	})
	if err != nil {
		return ForecastDTO{}, err
	}

	spend7d, err := s.queries.SumCustomerSpendLast7Days(ctx, pgCustomer)
	if err != nil {
		return ForecastDTO{}, err
	}
	runRate := spend7d / 7
	if runRate < 0 {
		runRate = 0
	}

	out := ForecastDTO{
		CustomerID:               customerID.String(),
		Month:                    monthStart.Format("2006-01"),
		LedgerMTDMicro:           mtd,
		LedgerRunRateMicroPerDay: runRate,
		DaysRemaining:            daysRemaining,
		ProjectedMonthEndMicro:   mtd + runRate*int64(daysRemaining),
	}

	if s.chQuery == nil {
		out.LowConfidence = true
		out.CHUnavailable = true
		return out, nil
	}

	campaignIDs, err := s.customerCampaignIDs(ctx, customerID)
	if err != nil {
		return ForecastDTO{}, err
	}
	if len(campaignIDs) == 0 {
		out.LowConfidence = true
		return out, nil
	}

	chCtx, cancel := context.WithTimeout(ctx, billingForecastCHTimeout)
	defer cancel()

	lookback := now.Add(-7 * 24 * time.Hour)
	points, chErr := s.queryCHHourlyImpressions(chCtx, lookback, now, campaignIDs)
	if chErr != nil {
		out.LowConfidence = true
		out.CHUnavailable = true
		return out, nil
	}
	out.CHHourlyImpressions = points
	if len(points) == 0 {
		out.LowConfidence = true
	}
	return out, nil
}

func (s *CompositeReadService) customerCampaignIDs(ctx context.Context, customerID uuid.UUID) ([]uuid.UUID, error) {
	rows, err := s.pool.Query(ctx, `
		SELECT id FROM campaigns
		WHERE customer_id = $1 AND deleted_at IS NULL`, customerID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var ids []uuid.UUID
	for rows.Next() {
		var id pgtype.UUID
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}
		ids = append(ids, uuid.UUID(id.Bytes))
	}
	return ids, rows.Err()
}

func (s *CompositeReadService) queryCHHourlyImpressions(ctx context.Context, from, to time.Time, campaignIDs []uuid.UUID) ([]CHHourlyPoint, error) {
	if s.chQuery == nil {
		return nil, fmt.Errorf("clickhouse not configured")
	}
	query := `
SELECT toStartOfHour(hour) AS hr, sum(impression_count) AS impressions
FROM mv_campaign_hourly_impressions
WHERE hour >= ? AND hour < ? AND campaign_id IN (?)
GROUP BY hr
ORDER BY hr`
	rows, err := s.chQuery.Query(ctx, query, from, to, campaignIDs)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	out := make([]CHHourlyPoint, 0, 168)
	for rows.Next() {
		var hr time.Time
		var impressions uint64
		if err := rows.Scan(&hr, &impressions); err != nil {
			return nil, err
		}
		out = append(out, CHHourlyPoint{
			Hour:        hr.UTC().Format(time.RFC3339),
			Impressions: int64(impressions),
		})
	}
	return out, rows.Err()
}

func (h *BillingHTTPHandlers) registerBalanceRoutes(mux *http.ServeMux) {
	if h.CustomerBalance == nil {
		return
	}
	limit := h.ApplyRateLimit
	perm := h.RequirePermission
	exportLimit := h.LimitExportByCustomer
	if exportLimit == nil {
		exportLimit = limit
	}
	mux.HandleFunc("GET /api/v1/customers/{id}/balance", limit(perm("customers:read", h.getCustomerBalance)))
	mux.HandleFunc("GET /api/v1/customers/{id}/ledger", limit(perm("customers:read", h.getCustomerLedger)))
	mux.HandleFunc("GET /api/v1/customers/{id}/balance/export", limit(exportLimit(perm("customers:read", h.exportCustomerBalance))))
}

func (h *BillingHTTPHandlers) getCustomerBalance(w http.ResponseWriter, r *http.Request) {
	idStr := r.PathValue("id")
	customerID, err := uuid.Parse(idStr)
	if err != nil {
		httpresponse.Error(w, http.StatusBadRequest, "BAD_REQUEST", "invalid customer id")
		return
	}
	if err := h.authorizeCustomer(r, idStr); err != nil {
		h.writeServiceError(w, err)
		return
	}

	report, err := h.CustomerBalance.GetCustomerBalance(r.Context(), customerID)
	if err != nil {
		h.writeServiceError(w, err)
		return
	}
	httpresponse.JSON(w, http.StatusOK, report)
}

func (h *BillingHTTPHandlers) getCustomerLedger(w http.ResponseWriter, r *http.Request) {
	idStr := r.PathValue("id")
	customerID, err := uuid.Parse(idStr)
	if err != nil {
		httpresponse.Error(w, http.StatusBadRequest, "BAD_REQUEST", "invalid customer id")
		return
	}
	if err := h.authorizeCustomer(r, idStr); err != nil {
		h.writeServiceError(w, err)
		return
	}
	if h.CustomerBalance == nil {
		httpresponse.Error(w, http.StatusServiceUnavailable, "UNAVAILABLE", "ledger reader not configured")
		return
	}

	limit, offset := parsePagination(r)

	items, total, err := h.CustomerBalance.ListCustomerLedger(r.Context(), customerID, limit, offset)
	if err != nil {
		h.writeServiceError(w, err)
		return
	}
	httpresponse.JSON(w, http.StatusOK, LedgerListResponse{
		Items:  items,
		Total:  total,
		Limit:  limit,
		Offset: offset,
	})
}

func (h *BillingHTTPHandlers) exportCustomerBalance(w http.ResponseWriter, r *http.Request) {
	if r.URL.Query().Get("format") != "csv" {
		httpresponse.Error(w, http.StatusBadRequest, "BAD_REQUEST", "format must be csv")
		return
	}

	idStr := r.PathValue("id")
	customerID, err := uuid.Parse(idStr)
	if err != nil {
		httpresponse.Error(w, http.StatusBadRequest, "BAD_REQUEST", "invalid customer id")
		return
	}
	if err := h.authorizeCustomer(r, idStr); err != nil {
		h.writeServiceError(w, err)
		return
	}

	cursor, err := parseExportCursor(r)
	if err != nil {
		h.writeServiceError(w, err)
		return
	}

	w.Header().Set("Content-Type", "text/csv; charset=utf-8")
	w.Header().Set("Cache-Control", "no-store")

	var buf bytes.Buffer
	result, err := h.CustomerBalance.ExportCustomerLedgerCSV(r.Context(), customerID, cursor, &buf)
	if err != nil {
		h.writeServiceError(w, err)
		return
	}

	if result.Truncated {
		w.Header().Set("X-Export-Truncated", "true")
		w.Header().Set("X-Next-Cursor", strconv.FormatInt(result.NextCursor, 10))
	}
	w.Header().Set("X-Export-Bytes", strconv.Itoa(result.Bytes))
	if _, err := w.Write(buf.Bytes()); err != nil {
		return
	}
}

func (h *BillingHTTPHandlers) authorizeCustomer(r *http.Request, customerID string) error {
	if h.AuthorizeCustomerAccess == nil {
		return nil
	}
	return h.AuthorizeCustomerAccess(r, customerID)
}

type invalidExportCursorError string

func errInvalidExportCursor(msg string) error {
	return invalidExportCursorError(msg)
}

func (e invalidExportCursorError) Error() string { return string(e) }

func parseExportCursor(r *http.Request) (int64, error) {
	cursorStr := r.URL.Query().Get("cursor")
	if cursorStr == "" {
		return 0, nil
	}
	cursor, err := strconv.ParseInt(cursorStr, 10, 64)
	if err != nil || cursor < 0 {
		return 0, errInvalidExportCursor("invalid cursor")
	}
	return cursor, nil
}

func (h *BillingHTTPHandlers) registerDisputeRoutes(mux *http.ServeMux) {
	if h.Disputes == nil {
		return
	}
	limit := h.ApplyRateLimit
	perm := h.RequirePermission
	mux.HandleFunc("GET /api/v1/disputes", limit(perm("customers:read", h.listDisputes)))
}

func (h *BillingHTTPHandlers) listDisputes(w http.ResponseWriter, r *http.Request) {
	customerFilter := r.URL.Query().Get("customer_id")
	if h.ResolveDisputeCustomerFilter != nil {
		filter, err := h.ResolveDisputeCustomerFilter(r)
		if err != nil {
			h.writeServiceError(w, err)
			return
		}
		customerFilter = filter
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

	result, err := h.Disputes.ListDisputes(r.Context(), customerFilter, limit, offset)
	if err != nil {
		if st, ok := status.FromError(err); ok && st.Code() == codes.InvalidArgument {
			httpresponse.Error(w, http.StatusBadRequest, "BAD_REQUEST", st.Message())
			return
		}
		httpresponse.Error(w, http.StatusInternalServerError, "INTERNAL", "failed to list disputes")
		return
	}
	httpresponse.JSON(w, http.StatusOK, result)
}

const ledgerInvariantToleranceMicro = int64(1)

type CompositeReadService struct {
	pool                *pgxpool.Pool
	cfg                 *config.Config
	paymentProviderName string
	queries             *billingdb.Queries
	chQuery             *database.CHQuery
}

func NewCompositeReadService(pool *pgxpool.Pool, cfg *config.Config) *CompositeReadService {
	if pool == nil {
		return nil
	}
	providerName := "placeholder"
	if cfg != nil && cfg.Billing.PaymentProvider != "" {
		providerName = cfg.Billing.PaymentProvider
	}
	return &CompositeReadService{
		pool:                pool,
		cfg:                 cfg,
		paymentProviderName: providerName,
		queries:             billingdb.New(pool),
	}
}

func (c *CompositeReadService) SetCHQuery(q *database.CHQuery) {
	if c != nil {
		c.chQuery = q
	}
}

type PeriodBounds struct {
	From time.Time `json:"from"`
	To   time.Time `json:"to"`
}

type StatementDTO struct {
	CustomerID          string                  `json:"customer_id"`
	Period              PeriodBounds            `json:"period"`
	OpeningBalanceMicro int64                   `json:"opening_balance_micro"`
	ClosingBalanceMicro int64                   `json:"closing_balance_micro"`
	Lines               []ledger.InvoiceLineDTO `json:"lines"`
	Invoices            []domain.InvoiceSummary `json:"invoices"`
	Payments            []domain.PaymentSummary `json:"payments"`
	TaxBreakdown        TaxBreakdownDTO         `json:"tax_breakdown"`
	Reconciliation      ReconciliationDTO       `json:"reconciliation"`
	Currency            string                  `json:"currency"`
}

type TaxBreakdownDTO struct {
	Scheme   string `json:"scheme"`
	RateBps  int32  `json:"rate_bps"`
	TaxMicro int64  `json:"tax_micro"`
}

type ReconciliationDTO struct {
	InvoiceTotalMicro int64 `json:"invoice_total_micro"`
	LedgerSumMicro    int64 `json:"ledger_sum_micro"`
	DeltaMicro        int64 `json:"delta_micro"`
}

type WalletDTO struct {
	CustomerID                string `json:"customer_id"`
	BalanceMicro              int64  `json:"balance_micro"`
	Currency                  string `json:"currency"`
	AllowedOverdraftMicro     int64  `json:"allowed_overdraft_micro"`
	LowBalanceThresholdMicro  int64  `json:"low_balance_threshold_micro"`
	BurnDaysEstimate          *int   `json:"burn_days_estimate,omitempty"`
	LastInvoiceAt             string `json:"last_invoice_at,omitempty"`
	PaymentProvider           string `json:"payment_provider"`
	PaymentProviderConfigured bool   `json:"payment_provider_configured"`
}

// fleetInvariantScanLimit caps operator fleet-wide invariant scans (no customer_id).
const fleetInvariantScanLimit = 500

type InvariantDTO struct {
	OK             bool   `json:"ok"`
	CustomerID     string `json:"customer_id,omitempty"`
	BalanceMicro   int64  `json:"balance_micro,omitempty"`
	LedgerSumMicro int64  `json:"ledger_sum_micro,omitempty"`
	DiffMicro      int64  `json:"diff_micro,omitempty"`
	// FleetScanLimit is set when customer_id is omitted (admin fleet scan).
	FleetScanLimit int `json:"fleet_scan_limit,omitempty"`
}

type SummaryDTO struct {
	InvoicedMTDMicro                int64 `json:"invoiced_mtd_micro"`
	InvoiceCountMTD                 int64 `json:"invoice_count_mtd"`
	UndeliveredInvoiceNotifications int64 `json:"undelivered_invoice_notifications"`
	CustomersWithSpendInMonth       int64 `json:"customers_with_spend_in_month"`
}

type DeliveryDTO struct {
	ID           string `json:"id"`
	Status       string `json:"status"`
	Provider     string `json:"provider"`
	Recipient    string `json:"recipient"`
	TemplateID   string `json:"template_id"`
	ErrorMessage string `json:"error_message,omitempty"`
	RetryCount   int32  `json:"retry_count"`
	CreatedAt    string `json:"created_at"`
	UpdatedAt    string `json:"updated_at"`
}

type LedgerLineDTO struct {
	ID          int64  `json:"id"`
	AmountMicro int64  `json:"amount_micro"`
	LedgerType  string `json:"ledger_type"`
	CreatedAt   string `json:"created_at"`
}

type TaxProfileDTO struct {
	CustomerID  string `json:"customer_id"`
	CountryCode string `json:"country_code"`
	TaxRegion   string `json:"tax_region,omitempty"`
	TaxScheme   string `json:"tax_scheme"`
	TaxRateBps  int32  `json:"tax_rate_bps"`
}

func (s *CompositeReadService) BuildStatement(ctx context.Context, customerID uuid.UUID, from, to time.Time) (StatementDTO, error) {
	if s == nil || s.pool == nil {
		return StatementDTO{}, fmt.Errorf("composite read service not configured")
	}
	from = from.UTC()
	to = to.UTC()
	if !to.After(from) {
		return StatementDTO{}, fmt.Errorf("invalid period: to must be after from")
	}

	pgCustomer := pgtype.UUID{Bytes: customerID, Valid: true}
	opening, err := s.queries.SumCustomerLedgerBefore(ctx, billingdb.SumCustomerLedgerBeforeParams{
		CustomerID: pgCustomer,
		CreatedAt:  pgTimestamp(from),
	})
	if err != nil {
		return StatementDTO{}, err
	}

	lines, err := s.queries.SumCustomerLedgerByTypeInWindow(ctx, billingdb.SumCustomerLedgerByTypeInWindowParams{
		CustomerID:  pgCustomer,
		CreatedAt:   pgTimestamp(from),
		CreatedAt_2: pgTimestamp(to),
	})
	if err != nil {
		return StatementDTO{}, err
	}

	var periodSum int64
	outLines := make([]ledger.InvoiceLineDTO, 0, len(lines))
	for _, line := range lines {
		periodSum += line.AmountMicro
		outLines = append(outLines, ledger.InvoiceLineDTO{
			LedgerType:  line.LedgerType,
			AmountMicro: line.AmountMicro,
			EntryCount:  line.EntryCount,
		})
	}
	closing := opening + periodSum

	invoices, err := s.queries.ListCustomerInvoicesInWindow(ctx, billingdb.ListCustomerInvoicesInWindowParams{
		CustomerID: pgCustomer,
		Column2:    pgDate(from),
		Column3:    pgDate(to),
	})
	if err != nil {
		return StatementDTO{}, err
	}

	invoiceDTOs := make([]domain.InvoiceSummary, 0, len(invoices))
	var invoiceTotal int64
	for _, inv := range invoices {
		month := ""
		if inv.BillingMonth.Valid {
			month = inv.BillingMonth.Time.UTC().Format("2006-01")
		}
		invoiceDTOs = append(invoiceDTOs, domain.InvoiceSummary{
			ID:            uuidString(inv.ID),
			BillingMonth:  month,
			SubtotalMicro: inv.SubtotalMicro,
			TaxMicro:      inv.TaxMicro,
			TotalMicro:    inv.TotalMicro,
			Status:        string(inv.Status),
			Currency:      inv.Currency,
		})
		if inv.Status == billingdb.BillingInvoiceStatusFINALIZED {
			invoiceTotal += inv.TotalMicro
		}
	}

	payments, err := s.queries.ListCustomerPaymentTopupsInWindow(ctx, billingdb.ListCustomerPaymentTopupsInWindowParams{
		CustomerID:  pgCustomer,
		CreatedAt:   pgTimestamp(from),
		CreatedAt_2: pgTimestamp(to),
		Limit:       100,
	})
	if err != nil {
		return StatementDTO{}, err
	}
	paymentDTOs := make([]domain.PaymentSummary, 0, len(payments))
	for _, p := range payments {
		intentID := ""
		if p.PaymentIntentID.Valid {
			intentID = uuid.UUID(p.PaymentIntentID.Bytes).String()
		}
		createdAt := ""
		if p.CreatedAt.Valid {
			createdAt = p.CreatedAt.Time.UTC().Format(time.RFC3339)
		}
		paymentDTOs = append(paymentDTOs, domain.PaymentSummary{
			LedgerID:        p.ID,
			AmountMicro:     p.Amount,
			PaymentIntentID: intentID,
			CreatedAt:       createdAt,
		})
	}

	cust, err := s.queries.GetCustomerBalance(ctx, pgCustomer)
	if err != nil {
		return StatementDTO{}, err
	}
	profile := ledger.ProfileFromDB(billingdb.BillingCustomerTaxProfile{})
	if row, perr := s.queries.GetCustomerTaxProfile(ctx, pgCustomer); perr == nil {
		profile = ledger.ProfileFromDB(row)
	} else if !errors.Is(perr, pgx.ErrNoRows) {
		return StatementDTO{}, perr
	}
	spendMicro, err := s.queries.SumCustomerSpendInWindow(ctx, billingdb.SumCustomerSpendInWindowParams{
		CustomerID:  pgCustomer,
		CreatedAt:   pgTimestamp(from),
		CreatedAt_2: pgTimestamp(to),
	})
	if err != nil {
		return StatementDTO{}, err
	}
	taxMicro, rateBPS := ledger.NewTaxCalculator().Compute(spendMicro, profile)

	return StatementDTO{
		CustomerID:          customerID.String(),
		Period:              PeriodBounds{From: from, To: to},
		OpeningBalanceMicro: opening,
		ClosingBalanceMicro: closing,
		Lines:               outLines,
		Invoices:            invoiceDTOs,
		Payments:            paymentDTOs,
		TaxBreakdown: TaxBreakdownDTO{
			Scheme:   string(profile.Scheme),
			RateBps:  rateBPS,
			TaxMicro: taxMicro,
		},
		Reconciliation: ReconciliationDTO{
			InvoiceTotalMicro: invoiceTotal,
			LedgerSumMicro:    periodSum,
			DeltaMicro:        invoiceTotal - periodSum,
		},
		Currency: cust.Currency,
	}, nil
}

func (s *CompositeReadService) GetWallet(ctx context.Context, customerID uuid.UUID) (WalletDTO, error) {
	if s == nil || s.pool == nil {
		return WalletDTO{}, fmt.Errorf("composite read service not configured")
	}
	pgCustomer := pgtype.UUID{Bytes: customerID, Valid: true}
	row, err := s.queries.GetCustomerWalletRow(ctx, pgCustomer)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return WalletDTO{}, ledger.ErrCustomerNotFound
		}
		return WalletDTO{}, err
	}

	wallet := WalletDTO{
		CustomerID:                customerID.String(),
		BalanceMicro:              row.Balance,
		Currency:                  row.Currency,
		AllowedOverdraftMicro:     row.AllowedOverdraft,
		PaymentProvider:           s.paymentProviderName,
		PaymentProviderConfigured: false,
	}
	if s.cfg != nil {
		wallet.LowBalanceThresholdMicro = s.cfg.Management.LowBalanceThresholdMicro
	}

	lastAt, err := s.queries.GetCustomerLastInvoiceAt(ctx, pgCustomer)
	if err == nil && lastAt.Valid && lastAt.Time.Year() > 1970 {
		wallet.LastInvoiceAt = lastAt.Time.UTC().Format(time.RFC3339)
	}

	spend7d, err := s.queries.SumCustomerSpendLast7Days(ctx, pgCustomer)
	if err == nil && spend7d > 0 && row.Balance > 0 {
		daily := spend7d / 7
		if daily > 0 {
			days := int(row.Balance / daily)
			wallet.BurnDaysEstimate = &days
		}
	}
	return wallet, nil
}

func (s *CompositeReadService) ListLedgerLines(ctx context.Context, customerID uuid.UUID, month time.Time, cursorID int64, limit int32) ([]LedgerLineDTO, string, int64, error) {
	if s == nil || s.pool == nil {
		return nil, "", 0, fmt.Errorf("composite read service not configured")
	}
	if limit <= 0 {
		limit = 50
	}
	if limit > 1000 {
		limit = 1000
	}
	monthStart := time.Date(month.Year(), month.Month(), 1, 0, 0, 0, 0, time.UTC)
	monthEnd := monthStart.AddDate(0, 1, 0)

	pgCustomer := pgtype.UUID{Bytes: customerID, Valid: true}
	total, err := s.queries.CountCustomerLedgerInWindow(ctx, billingdb.CountCustomerLedgerInWindowParams{
		CustomerID:  pgCustomer,
		CreatedAt:   pgTimestamp(monthStart),
		CreatedAt_2: pgTimestamp(monthEnd),
	})
	if err != nil {
		return nil, "", 0, err
	}

	rows, err := s.queries.ListCustomerLedgerInWindow(ctx, billingdb.ListCustomerLedgerInWindowParams{
		CustomerID:  pgCustomer,
		CreatedAt:   pgTimestamp(monthStart),
		CreatedAt_2: pgTimestamp(monthEnd),
		Column4:     cursorID,
		Limit:       limit,
	})
	if err != nil {
		return nil, "", 0, err
	}

	out := make([]LedgerLineDTO, 0, len(rows))
	var lastID int64
	for _, row := range rows {
		lastID = row.ID
		createdAt := ""
		if row.CreatedAt.Valid {
			createdAt = row.CreatedAt.Time.UTC().Format(time.RFC3339)
		}
		out = append(out, LedgerLineDTO{
			ID:          row.ID,
			AmountMicro: row.Amount,
			LedgerType:  string(row.Type),
			CreatedAt:   createdAt,
		})
	}
	nextCursor := ""
	if int32(len(out)) == limit && lastID > 0 {
		nextCursor = fmt.Sprintf("%d", lastID)
	}
	return out, nextCursor, total, nil
}

func (s *CompositeReadService) ListLedgerLinesInWindow(ctx context.Context, customerID uuid.UUID, from, to time.Time, cursorID int64, limit int32) ([]LedgerLineDTO, string, error) {
	if s == nil || s.pool == nil {
		return nil, "", fmt.Errorf("composite read service not configured")
	}
	if limit <= 0 {
		limit = 1000
	}
	pgCustomer := pgtype.UUID{Bytes: customerID, Valid: true}
	rows, err := s.queries.ListCustomerLedgerInWindow(ctx, billingdb.ListCustomerLedgerInWindowParams{
		CustomerID:  pgCustomer,
		CreatedAt:   pgTimestamp(from),
		CreatedAt_2: pgTimestamp(to),
		Column4:     cursorID,
		Limit:       limit,
	})
	if err != nil {
		return nil, "", err
	}
	out := make([]LedgerLineDTO, 0, len(rows))
	var lastID int64
	for _, row := range rows {
		lastID = row.ID
		createdAt := ""
		if row.CreatedAt.Valid {
			createdAt = row.CreatedAt.Time.UTC().Format(time.RFC3339)
		}
		out = append(out, LedgerLineDTO{
			ID:          row.ID,
			AmountMicro: row.Amount,
			LedgerType:  string(row.Type),
			CreatedAt:   createdAt,
		})
	}
	next := ""
	if int32(len(out)) == limit && lastID > 0 {
		next = fmt.Sprintf("%d", lastID)
	}
	return out, next, nil
}

func (s *CompositeReadService) GetInvariant(ctx context.Context, customerID *uuid.UUID) (InvariantDTO, error) {
	if s == nil || s.pool == nil {
		return InvariantDTO{}, fmt.Errorf("composite read service not configured")
	}
	if customerID != nil {
		snap, err := ledger.ReadLedgerInvariant(ctx, s.pool, *customerID)
		if err != nil {
			return InvariantDTO{}, err
		}
		diff := snap.BalanceMicro - snap.LedgerSumMicro
		ok := diff >= -ledgerInvariantToleranceMicro && diff <= ledgerInvariantToleranceMicro
		return InvariantDTO{
			OK:             ok,
			CustomerID:     customerID.String(),
			BalanceMicro:   snap.BalanceMicro,
			LedgerSumMicro: snap.LedgerSumMicro,
			DiffMicro:      diff,
		}, nil
	}

	rows, err := s.pool.Query(ctx, `SELECT id FROM customers ORDER BY id LIMIT $1`, fleetInvariantScanLimit)
	if err != nil {
		return InvariantDTO{}, err
	}
	defer rows.Close()
	for rows.Next() {
		var id pgtype.UUID
		if err := rows.Scan(&id); err != nil {
			return InvariantDTO{}, err
		}
		cid := uuid.UUID(id.Bytes)
		snap, err := ledger.ReadLedgerInvariant(ctx, s.pool, cid)
		if err != nil {
			return InvariantDTO{}, err
		}
		diff := snap.BalanceMicro - snap.LedgerSumMicro
		if diff < -ledgerInvariantToleranceMicro || diff > ledgerInvariantToleranceMicro {
			return InvariantDTO{
				OK:             false,
				CustomerID:     cid.String(),
				BalanceMicro:   snap.BalanceMicro,
				LedgerSumMicro: snap.LedgerSumMicro,
				DiffMicro:      diff,
				FleetScanLimit: fleetInvariantScanLimit,
			}, nil
		}
	}
	return InvariantDTO{OK: true, FleetScanLimit: fleetInvariantScanLimit}, rows.Err()
}

func (s *CompositeReadService) GetSummary(ctx context.Context) (SummaryDTO, error) {
	if s == nil || s.pool == nil {
		return SummaryDTO{}, fmt.Errorf("composite read service not configured")
	}
	mtd, err := s.queries.SumInvoicesMTD(ctx)
	if err != nil {
		return SummaryDTO{}, err
	}

	monthStart := time.Date(time.Now().UTC().Year(), time.Now().UTC().Month(), 1, 0, 0, 0, 0, time.UTC)
	monthEnd := monthStart.AddDate(0, 1, 0)
	customersSpend, err := s.queries.CountCustomersWithFeeSpendInWindow(ctx, billingdb.CountCustomersWithFeeSpendInWindowParams{
		CreatedAt:   pgTimestamp(monthStart),
		CreatedAt_2: pgTimestamp(monthEnd),
	})
	if err != nil {
		return SummaryDTO{}, err
	}

	var undelivered int64
	err = s.pool.QueryRow(ctx, `
		SELECT COUNT(*)::bigint
		FROM notify.notifications
		WHERE template_id = 'invoice_monthly' AND status NOT IN ('SENT')`).Scan(&undelivered)
	if err != nil {
		return SummaryDTO{}, err
	}

	return SummaryDTO{
		InvoicedMTDMicro:                mtd.Column1,
		InvoiceCountMTD:                 mtd.Column2,
		UndeliveredInvoiceNotifications: undelivered,
		CustomersWithSpendInMonth:       customersSpend,
	}, nil
}

func (s *CompositeReadService) ListDeliveries(ctx context.Context, invoiceID string) ([]DeliveryDTO, error) {
	if s == nil || s.pool == nil {
		return nil, fmt.Errorf("composite read service not configured")
	}
	dedupKey := "invoice:" + invoiceID
	rows, err := s.pool.Query(ctx, `
		SELECT id, status::text, provider::text, recipient, template_id, error_message, retry_count, created_at, updated_at
		FROM notify.notifications
		WHERE dedup_key = $1
		ORDER BY created_at DESC`, dedupKey)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	out := make([]DeliveryDTO, 0, 4)
	for rows.Next() {
		var (
			id, status, provider, recipient, templateID string
			errorMessage                                pgtype.Text
			retryCount                                  int32
			createdAt, updatedAt                        time.Time
		)
		if err := rows.Scan(&id, &status, &provider, &recipient, &templateID, &errorMessage, &retryCount, &createdAt, &updatedAt); err != nil {
			return nil, err
		}
		dto := DeliveryDTO{
			ID:         id,
			Status:     status,
			Provider:   provider,
			Recipient:  recipient,
			TemplateID: templateID,
			RetryCount: retryCount,
			CreatedAt:  createdAt.UTC().Format(time.RFC3339),
			UpdatedAt:  updatedAt.UTC().Format(time.RFC3339),
		}
		if errorMessage.Valid {
			dto.ErrorMessage = errorMessage.String
		}
		out = append(out, dto)
	}
	return out, rows.Err()
}

func (s *CompositeReadService) GetTaxProfile(ctx context.Context, customerID uuid.UUID) (TaxProfileDTO, error) {
	if s == nil || s.queries == nil {
		return TaxProfileDTO{}, fmt.Errorf("composite read service not configured")
	}
	row, err := s.queries.GetCustomerTaxProfile(ctx, pgtype.UUID{Bytes: customerID, Valid: true})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			calc := ledger.NewTaxCalculator()
			def := calc.DefaultProfile("", "")
			return TaxProfileDTO{
				CustomerID:  customerID.String(),
				CountryCode: def.CountryCode,
				TaxScheme:   string(def.Scheme),
				TaxRateBps:  def.RateBPS,
			}, nil
		}
		return TaxProfileDTO{}, err
	}
	dto := TaxProfileDTO{
		CustomerID:  customerID.String(),
		CountryCode: row.CountryCode,
		TaxScheme:   string(row.TaxScheme),
		TaxRateBps:  row.TaxRateBps,
	}
	if row.TaxRegion.Valid {
		dto.TaxRegion = row.TaxRegion.String
	}
	return dto, nil
}

func (s *CompositeReadService) UpsertTaxProfile(ctx context.Context, customerID uuid.UUID, dto TaxProfileDTO) (TaxProfileDTO, error) {
	if s == nil || s.queries == nil {
		return TaxProfileDTO{}, fmt.Errorf("composite read service not configured")
	}
	row, err := s.queries.UpsertCustomerTaxProfile(ctx, billingdb.UpsertCustomerTaxProfileParams{
		CustomerID:  pgtype.UUID{Bytes: customerID, Valid: true},
		CountryCode: dto.CountryCode,
		TaxRegion:   pgtype.Text{String: dto.TaxRegion, Valid: dto.TaxRegion != ""},
		TaxScheme:   billingdb.BillingTaxScheme(dto.TaxScheme),
		TaxRateBps:  dto.TaxRateBps,
	})
	if err != nil {
		return TaxProfileDTO{}, err
	}
	out := TaxProfileDTO{
		CustomerID:  customerID.String(),
		CountryCode: row.CountryCode,
		TaxScheme:   string(row.TaxScheme),
		TaxRateBps:  row.TaxRateBps,
	}
	if row.TaxRegion.Valid {
		out.TaxRegion = row.TaxRegion.String
	}
	return out, nil
}

func ParseStatementPeriod(fromRaw, toRaw, monthRaw string) (time.Time, time.Time, error) {
	if monthRaw != "" {
		month, err := time.Parse("2006-01", monthRaw)
		if err != nil {
			return time.Time{}, time.Time{}, fmt.Errorf("invalid month")
		}
		start := time.Date(month.Year(), month.Month(), 1, 0, 0, 0, 0, time.UTC)
		return start, start.AddDate(0, 1, 0), nil
	}
	if fromRaw == "" || toRaw == "" {
		now := time.Now().UTC()
		start := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, time.UTC)
		return start, start.AddDate(0, 1, 0), nil
	}
	from, err := time.Parse(time.RFC3339, fromRaw)
	if err != nil {
		return time.Time{}, time.Time{}, fmt.Errorf("invalid from")
	}
	to, err := time.Parse(time.RFC3339, toRaw)
	if err != nil {
		return time.Time{}, time.Time{}, fmt.Errorf("invalid to")
	}
	return from.UTC(), to.UTC(), nil
}

func pgTimestamp(t time.Time) pgtype.Timestamp {
	return pgtype.Timestamp{Time: t.UTC(), Valid: true}
}

func pgDate(t time.Time) pgtype.Date {
	return pgtype.Date{Time: time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, time.UTC), Valid: true}
}

func uuidString(id pgtype.UUID) string {
	if !id.Valid {
		return ""
	}
	return uuid.UUID(id.Bytes).String()
}
