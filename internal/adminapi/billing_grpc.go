package adminapi

import (
	"context"
	"time"

	"espx/internal/billing"
	billingpb "espx/internal/billing/pb"

	"github.com/google/uuid"
)

type InvoiceGRPCClient interface {
	ListInvoices(ctx context.Context, customerID string, limit, offset int32) (*billingpb.ListInvoicesResponse, error)
	GetInvoice(ctx context.Context, invoiceID string) (*billingpb.Invoice, error)
}

type InProcessInvoiceService interface {
	PreviewInvoice(ctx context.Context, customerID uuid.UUID, billingMonth time.Time) (*billing.InvoicePreview, error)
	VoidInvoice(ctx context.Context, invoiceID uuid.UUID) error
}

type InvoiceRetryer interface {
	RetryInvoiceDelivery(ctx context.Context, invoice *billingpb.Invoice, idempotencyKey string) error
}

type VoidAuditor interface {
	AuditInvoiceVoid(ctx context.Context, invoiceID, customerID string) error
}
