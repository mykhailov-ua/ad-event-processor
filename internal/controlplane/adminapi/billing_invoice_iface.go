package adminapi

import (
	"context"
	"time"

	"espx/internal/billing"

	"github.com/google/uuid"
)

type InProcessInvoiceService interface {
	PreviewInvoice(ctx context.Context, customerID uuid.UUID, billingMonth time.Time) (*billing.InvoicePreview, error)
	VoidInvoice(ctx context.Context, invoiceID uuid.UUID) error
}

type InvoiceRetryer interface {
	RetryInvoiceDelivery(ctx context.Context, invoice *billing.Invoice, idempotencyKey string) error
}

type VoidAuditor interface {
	AuditInvoiceVoid(ctx context.Context, invoiceID, customerID string) error
}
