package billing

import (
	"context"
	"time"

	"espx/internal/billing/pb"

	"github.com/google/uuid"
	"google.golang.org/grpc/metadata"
)

type InvoiceLine struct {
	LedgerType  string
	AmountMicro int64
	EntryCount  int32
}

type Invoice struct {
	ID            string
	CustomerID    string
	BillingMonth  time.Time
	SubtotalMicro int64
	TaxMicro      int64
	TotalMicro    int64
	Currency      string
	TaxScheme     string
	TaxRateBps    int32
	Lines         []InvoiceLine
	CreatedAt     time.Time
}

type ListInvoicesResult struct {
	Invoices []Invoice
	Total    int64
}

type BillingAPI interface {
	ListInvoices(ctx context.Context, customerID string, limit, offset int32) (ListInvoicesResult, error)
	GetInvoice(ctx context.Context, invoiceID string) (*Invoice, error)
	GenerateInvoice(ctx context.Context, customerID string, billingMonth time.Time) (*Invoice, error)
}

type billingAPI struct {
	h     *Handler
	token string
}

func (m *Module) API(token string) BillingAPI {
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

func (a *billingAPI) ListInvoices(ctx context.Context, customerID string, limit, offset int32) (ListInvoicesResult, error) {
	customerUUID, err := uuid.Parse(customerID)
	if err != nil || customerUUID == uuid.Nil {
		return ListInvoicesResult{}, ErrInvalidCustomerID
	}
	invoices, total, err := a.h.listInvoices(a.incoming(ctx), customerUUID, limit, offset)
	if err != nil {
		return ListInvoicesResult{}, err
	}
	out := ListInvoicesResult{Total: total}
	if len(invoices) == 0 {
		return out, nil
	}
	out.Invoices = make([]Invoice, 0, len(invoices))
	for _, inv := range invoices {
		if parsed := InvoiceFromPB(inv); parsed != nil {
			out.Invoices = append(out.Invoices, *parsed)
		}
	}
	return out, nil
}

func (a *billingAPI) GetInvoice(ctx context.Context, invoiceID string) (*Invoice, error) {
	invoiceUUID, err := uuid.Parse(invoiceID)
	if err != nil || invoiceUUID == uuid.Nil {
		return nil, ErrInvalidInvoiceID
	}
	inv, err := a.h.getInvoice(a.incoming(ctx), invoiceUUID)
	if err != nil {
		return nil, err
	}
	return InvoiceFromPB(inv), nil
}

func (a *billingAPI) GenerateInvoice(ctx context.Context, customerID string, billingMonth time.Time) (*Invoice, error) {
	customerUUID, err := uuid.Parse(customerID)
	if err != nil || customerUUID == uuid.Nil {
		return nil, ErrInvalidCustomerID
	}
	month := time.Date(billingMonth.Year(), billingMonth.Month(), 1, 0, 0, 0, 0, time.UTC)
	inv, err := a.h.generateInvoice(a.incoming(ctx), customerUUID, month)
	if err != nil {
		return nil, err
	}
	return InvoiceFromPB(inv), nil
}

func InvoiceFromPB(inv *pb.Invoice) *Invoice {
	if inv == nil {
		return nil
	}
	out := &Invoice{
		ID:            inv.Id,
		CustomerID:    inv.CustomerId,
		SubtotalMicro: inv.SubtotalMicro,
		TaxMicro:      inv.TaxMicro,
		TotalMicro:    inv.TotalMicro,
		Currency:      inv.Currency,
		TaxScheme:     inv.TaxScheme,
		TaxRateBps:    inv.TaxRateBps,
	}
	if inv.BillingMonth != nil {
		out.BillingMonth = inv.BillingMonth.AsTime().UTC()
	}
	if inv.CreatedAt != nil {
		out.CreatedAt = inv.CreatedAt.AsTime().UTC()
	}
	if len(inv.Lines) > 0 {
		out.Lines = make([]InvoiceLine, 0, len(inv.Lines))
		for _, line := range inv.Lines {
			if line == nil {
				continue
			}
			out.Lines = append(out.Lines, InvoiceLine{
				LedgerType:  line.LedgerType,
				AmountMicro: line.AmountMicro,
				EntryCount:  line.EntryCount,
			})
		}
	}
	return out
}

func ListInvoicesResultFromPB(resp *pb.ListInvoicesResponse) ListInvoicesResult {
	if resp == nil {
		return ListInvoicesResult{}
	}
	out := ListInvoicesResult{Total: resp.Total}
	if len(resp.Invoices) == 0 {
		return out
	}
	out.Invoices = make([]Invoice, 0, len(resp.Invoices))
	for _, inv := range resp.Invoices {
		if parsed := InvoiceFromPB(inv); parsed != nil {
			out.Invoices = append(out.Invoices, *parsed)
		}
	}
	return out
}
