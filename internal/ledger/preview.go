package ledger

import (
	"context"
	"errors"
	"fmt"
	"time"

	"ad-event-processor/internal/ledger/db"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
)

type InvoicePreview struct {
	CustomerID     string           `json:"customer_id"`
	BillingMonth   string           `json:"billing_month"`
	Currency       string           `json:"currency"`
	SubtotalMicro  int64            `json:"subtotal_micro"`
	TaxMicro       int64            `json:"tax_micro"`
	TotalMicro     int64            `json:"total_micro"`
	TaxScheme      string           `json:"tax_scheme"`
	TaxRateBps     int32            `json:"tax_rate_bps"`
	Lines          []InvoiceLineDTO `json:"lines"`
	WouldSkip      bool             `json:"would_skip"`
	LedgerSumMicro int64            `json:"ledger_sum_micro"`
}

type InvoiceLineDTO struct {
	LedgerType  string `json:"ledger_type"`
	AmountMicro int64  `json:"amount_micro"`
	EntryCount  int32  `json:"entry_count"`
}

func (s *Service) PreviewInvoice(ctx context.Context, customerID uuid.UUID, billingMonth time.Time) (*InvoicePreview, error) {
	if err := validateBillingMonth(billingMonth); err != nil {
		return nil, err
	}
	if err := CheckLedgerBalanceInvariant(ctx, s.pool, customerID); err != nil {
		return nil, err
	}

	monthStart := truncateMonthUTC(billingMonth)
	monthEnd := monthStart.AddDate(0, 1, 0)

	cust, err := s.queries.GetCustomerBalance(ctx, pgtype.UUID{Bytes: customerID, Valid: true})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrCustomerNotFound
		}
		return nil, err
	}

	ledgerSum, err := s.queries.SumCustomerLedgerTotal(ctx, pgtype.UUID{Bytes: customerID, Valid: true})
	if err != nil {
		return nil, err
	}

	spendMicro, err := s.queries.SumCustomerSpendInWindow(ctx, db.SumCustomerSpendInWindowParams{
		CustomerID:  pgtype.UUID{Bytes: customerID, Valid: true},
		CreatedAt:   pgTimestamp(monthStart),
		CreatedAt_2: pgTimestamp(monthEnd),
	})
	if err != nil {
		return nil, err
	}

	lines, err := s.queries.SumCustomerLedgerByTypeInWindow(ctx, db.SumCustomerLedgerByTypeInWindowParams{
		CustomerID:  pgtype.UUID{Bytes: customerID, Valid: true},
		CreatedAt:   pgTimestamp(monthStart),
		CreatedAt_2: pgTimestamp(monthEnd),
	})
	if err != nil {
		return nil, err
	}

	profile := s.resolveTaxProfile(ctx, s.queries, customerID, cust.Currency)
	taxMicro, rateBPS := s.tax.Compute(spendMicro, profile)
	totalMicro := spendMicro + taxMicro

	out := &InvoicePreview{
		CustomerID:     customerID.String(),
		BillingMonth:   monthStart.Format("2006-01"),
		Currency:       cust.Currency,
		SubtotalMicro:  spendMicro,
		TaxMicro:       taxMicro,
		TotalMicro:     totalMicro,
		TaxScheme:      string(profile.Scheme),
		TaxRateBps:     rateBPS,
		LedgerSumMicro: ledgerSum,
		WouldSkip:      spendMicro == 0,
	}
	for _, line := range lines {
		out.Lines = append(out.Lines, InvoiceLineDTO{
			LedgerType:  line.LedgerType,
			AmountMicro: line.AmountMicro,
			EntryCount:  line.EntryCount,
		})
	}
	return out, nil
}

func (s *Service) VoidInvoice(ctx context.Context, invoiceID uuid.UUID) error {
	tag, err := s.queries.VoidInvoice(ctx, pgtype.UUID{Bytes: invoiceID, Valid: true})
	if err != nil {
		return err
	}
	if tag == 0 {
		inv, lookupErr := s.queries.GetInvoice(ctx, pgtype.UUID{Bytes: invoiceID, Valid: true})
		if lookupErr != nil {
			if errors.Is(lookupErr, pgx.ErrNoRows) {
				return ErrInvoiceNotFound
			}
			return lookupErr
		}
		if inv.Status == db.BillingInvoiceStatusVOID {
			return nil
		}
		return fmt.Errorf("invoice cannot be voided in status %s", inv.Status)
	}
	return nil
}
