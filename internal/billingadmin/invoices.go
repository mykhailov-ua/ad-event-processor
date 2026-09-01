package billingadmin

import (
	"ad-event-processor/internal/domain"
	billingdb "ad-event-processor/internal/ledger/db"
	"ad-event-processor/pkg/coldpath"
)

func invoiceSummaryFromRow(inv billingdb.BillingInvoice) domain.InvoiceSummary {
	monthStr := ""
	if inv.BillingMonth.Valid {
		monthStr = inv.BillingMonth.Time.UTC().Format("2006-01")
	}
	summary := domain.InvoiceSummary{
		ID:            uuidString(inv.ID),
		CustomerID:    uuidString(inv.CustomerID),
		BillingMonth:  monthStr,
		SubtotalMicro: inv.SubtotalMicro,
		TaxMicro:      inv.TaxMicro,
		TotalMicro:    inv.TotalMicro,
		Status:        string(inv.Status),
		Currency:      inv.Currency,
	}
	attachInvoiceSummaryDisplay(&summary)
	return summary
}

func attachInvoiceSummaryDisplay(summary *domain.InvoiceSummary) {
	if summary == nil {
		return
	}
	summary.SubtotalMicroDisplay = coldpath.FormatMicroDisplay(summary.SubtotalMicro)
	summary.TaxMicroDisplay = coldpath.FormatMicroDisplay(summary.TaxMicro)
	summary.TotalMicroDisplay = coldpath.FormatMicroDisplay(summary.TotalMicro)
}
