package billingadmin

import (
	"bytes"
	"context"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"

	"ad-event-processor/pkg/httpresponse"

	"github.com/google/uuid"
)

type UsageDailyExporter interface {
	ExportUsageDailyCSV(ctx context.Context, spec UsageExportSpec, w io.Writer) (UsageExportResult, error)
}

type PatchCustomerCostCenterRequest struct {
	CostCenter string `json:"cost_center"`
}

func (h *HTTPHandlers) registerUsageExportRoutes(mux *http.ServeMux) {
	if h.UsageExport == nil {
		return
	}
	limit := h.ApplyRateLimit
	perm := h.RequirePermission
	exportLimit := h.LimitExportByCustomer
	if exportLimit == nil {
		exportLimit = limit
	}
	mux.HandleFunc("GET /api/v1/billing/usage/export", limit(exportLimit(perm("customers:read", h.exportUsageDaily))))
}

func (h *HTTPHandlers) exportUsageDaily(w http.ResponseWriter, r *http.Request) {
	if r.URL.Query().Get("format") != "csv" {
		httpresponse.Error(w, http.StatusBadRequest, "BAD_REQUEST", "format must be csv")
		return
	}

	spec, err := h.parseUsageExportSpec(r)
	if err != nil {
		h.writeServiceError(w, err)
		return
	}

	w.Header().Set("Content-Type", "text/csv; charset=utf-8")
	w.Header().Set("Cache-Control", "no-store")

	var buf bytes.Buffer
	result, err := h.UsageExport.ExportUsageDailyCSV(r.Context(), spec, &buf)
	if err != nil {
		h.writeServiceError(w, err)
		return
	}

	if result.Truncated {
		w.Header().Set("X-Export-Truncated", "true")
		if result.NextCursor != "" {
			w.Header().Set("X-Next-Cursor", result.NextCursor)
		}
	}
	w.Header().Set("X-Export-Bytes", strconv.Itoa(result.Bytes))
	if _, err := w.Write(buf.Bytes()); err != nil {
		return
	}
}

func (h *HTTPHandlers) parseUsageExportSpec(r *http.Request) (UsageExportSpec, error) {
	fromRaw := strings.TrimSpace(r.URL.Query().Get("from"))
	toRaw := strings.TrimSpace(r.URL.Query().Get("to"))
	if fromRaw == "" || toRaw == "" {
		return UsageExportSpec{}, errValidation("from and to are required (YYYY-MM-DD)")
	}
	fromDate, err := time.Parse("2006-01-02", fromRaw)
	if err != nil {
		return UsageExportSpec{}, errValidation("invalid from date")
	}
	toDate, err := time.Parse("2006-01-02", toRaw)
	if err != nil {
		return UsageExportSpec{}, errValidation("invalid to date")
	}

	cursor, err := ParseUsageExportCursor(r.URL.Query().Get("cursor"))
	if err != nil {
		return UsageExportSpec{}, err
	}

	customerRaw := strings.TrimSpace(r.URL.Query().Get("customer_id"))
	costCenter := strings.TrimSpace(r.URL.Query().Get("cost_center"))

	if h.ResolveUsageExportCustomerFilter != nil {
		filteredCustomer, filteredCostCenter, authErr := h.ResolveUsageExportCustomerFilter(r, customerRaw, costCenter)
		if authErr != nil {
			return UsageExportSpec{}, authErr
		}
		customerRaw = filteredCustomer
		costCenter = filteredCostCenter
	}

	var customerID *uuid.UUID
	if customerRaw != "" {
		parsed, parseErr := uuid.Parse(customerRaw)
		if parseErr != nil {
			return UsageExportSpec{}, errValidation("invalid customer_id")
		}
		customerID = &parsed
		if err := h.authorizeCustomerAccess(r, customerRaw); err != nil {
			return UsageExportSpec{}, err
		}
	}

	normalizedCostCenter, err := normalizeCostCenter(costCenter)
	if err != nil {
		return UsageExportSpec{}, err
	}

	return UsageExportSpec{
		CustomerID: customerID,
		CostCenter: normalizedCostCenter,
		FromDate:   fromDate,
		ToDate:     toDate,
		Cursor:     cursor,
	}, nil
}
