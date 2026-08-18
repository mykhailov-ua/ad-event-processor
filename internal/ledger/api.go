package ledger

import (
	"context"
	"time"

	"github.com/bidshard/ad-event-processor/internal/config"
	"github.com/bidshard/ad-event-processor/internal/domain"

	"github.com/google/uuid"
	"google.golang.org/grpc/metadata"
)

type Handler struct {
	service *Service
	cfg     *config.Config
}

func NewHandler(service *Service, cfg *config.Config) *Handler {
	return &Handler{
		service: service,
		cfg:     cfg,
	}
}

type billingAPI struct {
	h     *Handler
	token string
}

func (m *Module) API(token string) domain.BillingAPI {
	if m == nil || m.Handler == nil {
		return nil
	}
	return &billingAPI{h: m.Handler, token: token}
}

func (a *billingAPI) incoming(ctx context.Context) context.Context {
	if a.token == "" {
		return ctx
	}
	return metadata.NewIncomingContext(ctx, metadata.Pairs("x-internal-token", a.token))
}

func (a *billingAPI) ListInvoices(ctx context.Context, customerID string, limit, offset int32) (domain.ListInvoicesResult, error) {
	customerUUID, err := uuid.Parse(customerID)
	if err != nil || customerUUID == uuid.Nil {
		return domain.ListInvoicesResult{}, ErrInvalidCustomerID
	}
	invoices, total, err := a.h.listInvoices(a.incoming(ctx), customerUUID, limit, offset)
	if err != nil {
		return domain.ListInvoicesResult{}, err
	}
	return domain.ListInvoicesResult{Invoices: invoices, Total: total}, nil
}

func (a *billingAPI) GetInvoice(ctx context.Context, invoiceID string) (*domain.Invoice, error) {
	invoiceUUID, err := uuid.Parse(invoiceID)
	if err != nil || invoiceUUID == uuid.Nil {
		return nil, ErrInvalidInvoiceID
	}
	inv, err := a.h.getInvoice(a.incoming(ctx), invoiceUUID)
	if err != nil {
		return nil, err
	}
	out := inv
	return &out, nil
}

func (a *billingAPI) GenerateInvoice(ctx context.Context, customerID string, billingMonth time.Time) (*domain.Invoice, error) {
	customerUUID, err := uuid.Parse(customerID)
	if err != nil || customerUUID == uuid.Nil {
		return nil, ErrInvalidCustomerID
	}
	month := time.Date(billingMonth.Year(), billingMonth.Month(), 1, 0, 0, 0, 0, time.UTC)
	inv, err := a.h.generateInvoice(a.incoming(ctx), customerUUID, month)
	if err != nil {
		return nil, err
	}
	out := inv
	return &out, nil
}
