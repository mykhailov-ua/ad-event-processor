package controlplane

import (
	"net/http"

	"ad-event-processor/internal/billingadmin"
)

type (
	BillingHTTPHandlers          = billingadmin.HTTPHandlers
	CryptoBillingWebhookHandlers = billingadmin.CryptoWebhookHandlers
	CompositeReadService         = billingadmin.CompositeReadService
	UsageExportSpec              = billingadmin.UsageExportSpec
	UsageExportCursor            = billingadmin.UsageExportCursor
	AdminInvoiceFilters          = billingadmin.AdminInvoiceFilters
	ForecastDTO                  = billingadmin.ForecastDTO
	StatementDTO                 = billingadmin.StatementDTO
	WalletDTO                    = billingadmin.WalletDTO
	LedgerLineDTO                = billingadmin.LedgerLineDTO
	InvariantDTO                 = billingadmin.InvariantDTO
	SummaryDTO                   = billingadmin.SummaryDTO
	DeliveryDTO                  = billingadmin.DeliveryDTO
	TaxProfileDTO                = billingadmin.TaxProfileDTO
	DisputeRowDTO                = billingadmin.DisputeRowDTO
	DisputeListResult            = billingadmin.DisputeListResult
	InvoiceRetryer               = billingadmin.InvoiceRetryer
	InProcessInvoiceService      = billingadmin.InProcessInvoiceService
	VoidAuditor                  = billingadmin.VoidAuditor
	CustomerBalanceReader        = billingadmin.CustomerBalanceReader
	UsageDailyExporter           = billingadmin.UsageDailyExporter
	DisputeLister                = billingadmin.DisputeLister
	PatchCustomerCostCenterRequest = billingadmin.PatchCustomerCostCenterRequest
	SelfServeInvoiceListResponse = billingadmin.SelfServeInvoiceListResponse
	LedgerListResponse           = billingadmin.LedgerListResponse
)

var (
	NewCompositeReadService = billingadmin.NewCompositeReadService
	ParseUsageExportCursor  = billingadmin.ParseUsageExportCursor
	ParseStatementPeriod    = billingadmin.ParseStatementPeriod
)

func WriteBillingError(w http.ResponseWriter, err error) {
	billingadmin.WriteBillingError(w, err)
}
