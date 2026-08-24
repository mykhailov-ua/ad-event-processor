package ledger

import (
	"context"
	"errors"
	"fmt"
	"time"

	"ad-event-processor/internal/domain"
	"ad-event-processor/internal/ledger/db"
	"ad-event-processor/internal/notify"
	"ad-event-processor/pkg/coldpath"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Service struct {
	pool            *pgxpool.Pool
	queries         *db.Queries
	tax             *TaxCalculator
	notifier        notify.NotifierAPI
	notifyProvider  string
	notifyRecipient string
	invoiceBaseURL  string
}

func NewService(pool *pgxpool.Pool) *Service {
	return &Service{
		pool:    pool,
		queries: db.New(pool),
		tax:     NewTaxCalculator(),
	}
}

func (service *Service) ListCustomerIDs(ctx context.Context, limit, offset int32) ([]uuid.UUID, error) {
	if limit <= 0 {
		limit = 200
	}
	rows, err := service.queries.ListCustomerIDs(ctx, db.ListCustomerIDsParams{Limit: limit, Offset: offset})
	if err != nil {
		return nil, err
	}
	out := make([]uuid.UUID, 0, len(rows))
	for _, row := range rows {
		out = append(out, uuid.UUID(row.Bytes))
	}
	return out, nil
}

func (service *Service) GenerateInvoice(ctx context.Context, customerID uuid.UUID, billingMonth time.Time) (domain.Invoice, error) {
	if err := validateBillingMonth(billingMonth); err != nil {
		return domain.Invoice{}, err
	}
	if err := CheckLedgerBalanceInvariant(ctx, service.pool, customerID); err != nil {
		LedgerDriftTotal.Inc()
		LedgerInvariantFailuresTotal.Inc()
		InvoiceErrorsTotal.WithLabelValues("ledger_drift").Inc()
		if service.notifier != nil {
			service.alertLedgerDrift(ctx, customerID.String(), err)
		}
		return domain.Invoice{}, err
	}

	monthStart := truncateMonthUTC(billingMonth)
	monthEnd := monthStart.AddDate(0, 1, 0)

	existing, err := service.queries.GetInvoiceByCustomerMonth(ctx, db.GetInvoiceByCustomerMonthParams{
		CustomerID:   pgtype.UUID{Bytes: customerID, Valid: true},
		BillingMonth: pgtype.Date{Time: monthStart, Valid: true},
	})
	if err == nil {
		return service.invoiceFromDB(ctx, existing)
	}
	if !errors.Is(err, pgx.ErrNoRows) {
		return domain.Invoice{}, fmt.Errorf("lookup invoice: %w", err)
	}

	tx, err := service.pool.Begin(ctx)
	if err != nil {
		return domain.Invoice{}, fmt.Errorf("begin transaction: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	qtx := service.queries.WithTx(tx)

	cust, err := qtx.GetCustomerBalance(ctx, pgtype.UUID{Bytes: customerID, Valid: true})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return domain.Invoice{}, ErrCustomerNotFound
		}
		return domain.Invoice{}, fmt.Errorf("load customer: %w", err)
	}

	ledgerSum, err := qtx.SumCustomerLedgerTotal(ctx, pgtype.UUID{Bytes: customerID, Valid: true})
	if err != nil {
		return domain.Invoice{}, fmt.Errorf("sum ledger: %w", err)
	}
	if diff := cust.Balance - ledgerSum; diff < -ledgerInvariantToleranceMicro || diff > ledgerInvariantToleranceMicro {
		return domain.Invoice{}, fmt.Errorf("%w: balance=%d ledger_sum=%d diff=%d", ErrLedgerDrift, cust.Balance, ledgerSum, diff)
	}

	spendMicro, err := qtx.SumCustomerSpendInWindow(ctx, db.SumCustomerSpendInWindowParams{
		CustomerID:  pgtype.UUID{Bytes: customerID, Valid: true},
		CreatedAt:   pgTimestamp(monthStart),
		CreatedAt_2: pgTimestamp(monthEnd),
	})
	if err != nil {
		return domain.Invoice{}, fmt.Errorf("sum spend window: %w", err)
	}

	lines, err := qtx.SumCustomerLedgerByTypeInWindow(ctx, db.SumCustomerLedgerByTypeInWindowParams{
		CustomerID:  pgtype.UUID{Bytes: customerID, Valid: true},
		CreatedAt:   pgTimestamp(monthStart),
		CreatedAt_2: pgTimestamp(monthEnd),
	})
	if err != nil {
		return domain.Invoice{}, fmt.Errorf("aggregate ledger lines: %w", err)
	}

	profile := service.resolveTaxProfile(ctx, qtx, customerID, cust.Currency)
	taxMicro, rateBPS := service.tax.Compute(spendMicro, profile)
	totalMicro := spendMicro + taxMicro

	if spendMicro == 0 {
		_ = tx.Rollback(ctx)
		return domain.Invoice{}, ErrNoSpend
	}

	invoiceID, err := uuid.NewV7()
	if err != nil {
		return domain.Invoice{}, fmt.Errorf("generate invoice id: %w", err)
	}

	invoice, err := qtx.CreateInvoice(ctx, db.CreateInvoiceParams{
		ID:             pgtype.UUID{Bytes: invoiceID, Valid: true},
		CustomerID:     pgtype.UUID{Bytes: customerID, Valid: true},
		BillingMonth:   pgtype.Date{Time: monthStart, Valid: true},
		SubtotalMicro:  spendMicro,
		TaxMicro:       taxMicro,
		TotalMicro:     totalMicro,
		Currency:       cust.Currency,
		TaxScheme:      MapSchemeToDB(profile.Scheme),
		TaxRateBps:     rateBPS,
		LedgerSumMicro: ledgerSum,
	})
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" {
			_ = tx.Rollback(ctx)
			existing, lookupErr := service.queries.GetInvoiceByCustomerMonth(ctx, db.GetInvoiceByCustomerMonthParams{
				CustomerID:   pgtype.UUID{Bytes: customerID, Valid: true},
				BillingMonth: pgtype.Date{Time: monthStart, Valid: true},
			})
			if lookupErr == nil {
				return service.invoiceFromDB(ctx, existing)
			}
		}
		return domain.Invoice{}, fmt.Errorf("insert invoice: %w", err)
	}

	for _, line := range lines {
		if _, err := qtx.CreateInvoiceLine(ctx, db.CreateInvoiceLineParams{
			InvoiceID:   invoice.ID,
			LedgerType:  line.LedgerType,
			AmountMicro: line.AmountMicro,
			EntryCount:  line.EntryCount,
		}); err != nil {
			return domain.Invoice{}, fmt.Errorf("insert invoice line: %w", err)
		}
	}

	if err := tx.Commit(ctx); err != nil {
		return domain.Invoice{}, fmt.Errorf("commit invoice: %w", err)
	}

	InvoicesGeneratedTotal.Inc()
	return service.invoiceFromDB(ctx, invoice)
}

func (service *Service) resolveTaxProfile(ctx context.Context, q *db.Queries, customerID uuid.UUID, currency string) TaxProfile {
	row, err := q.GetCustomerTaxProfile(ctx, pgtype.UUID{Bytes: customerID, Valid: true})
	if err == nil {
		return ProfileFromDB(row)
	}
	if !errors.Is(err, pgx.ErrNoRows) {
		return service.tax.DefaultProfile("US", currency)
	}
	return service.tax.DefaultProfile("US", currency)
}

func (service *Service) GetInvoice(ctx context.Context, invoiceID uuid.UUID) (domain.Invoice, error) {
	invoice, err := service.queries.GetInvoice(ctx, pgtype.UUID{Bytes: invoiceID, Valid: true})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return domain.Invoice{}, ErrInvoiceNotFound
		}
		return domain.Invoice{}, fmt.Errorf("get invoice: %w", err)
	}
	return service.invoiceFromDB(ctx, invoice)
}

func (service *Service) ListInvoices(ctx context.Context, customerID uuid.UUID, limit, offset int32) ([]domain.Invoice, int64, error) {
	if limit <= 0 {
		limit = 20
	}
	if offset < 0 {
		offset = 0
	}

	custUUID := pgtype.UUID{Bytes: customerID, Valid: true}
	listParams := db.ListCustomerInvoicesParams{
		CustomerID: custUUID,
		Limit:      limit,
		Offset:     offset,
	}
	rows, total, err := coldpath.PaginatedQuery(
		func() (int64, error) { return service.queries.CountCustomerInvoices(ctx, custUUID) },
		func() ([]db.BillingInvoice, error) { return service.queries.ListCustomerInvoices(ctx, listParams) },
	)
	if err != nil {
		return nil, 0, fmt.Errorf("list invoices: %w", err)
	}

	invoices, err := service.invoicesFromDB(ctx, rows)
	if err != nil {
		return nil, 0, err
	}
	return invoices, total, nil
}

func validateBillingMonth(month time.Time) error {
	m := month.UTC()
	if m.Day() != 1 || m.Hour() != 0 || m.Minute() != 0 || m.Second() != 0 || m.Nanosecond() != 0 {
		return ErrInvalidBillingMonth
	}
	return nil
}

func truncateMonthUTC(t time.Time) time.Time {
	u := t.UTC()
	return time.Date(u.Year(), u.Month(), 1, 0, 0, 0, 0, time.UTC)
}

func pgTimestamp(t time.Time) pgtype.Timestamp {
	return pgtype.Timestamp{Time: t.UTC(), Valid: true}
}

func ParseBillingMonth(raw string) (time.Time, error) {
	t, err := time.Parse("2006-01", raw)
	if err != nil {
		return time.Time{}, ErrInvalidBillingMonth
	}
	return truncateMonthUTC(t), nil
}
