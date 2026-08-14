package ledger

import (
	"context"
	"errors"
	"log/slog"
	"time"

	"github.com/google/uuid"
)

const advisoryUnlockTimeout = 5 * time.Second

type InvoiceWorker struct {
	service *Service
}

func NewInvoiceWorker(service *Service) *InvoiceWorker {
	return &InvoiceWorker{service: service}
}

func (w *InvoiceWorker) Start(ctx context.Context) {
	if w == nil || w.service == nil {
		return
	}

	for {
		wait := durationUntilNextInvoiceRun(time.Now().UTC())
		timer := time.NewTimer(wait)
		select {
		case <-ctx.Done():
			timer.Stop()
			return
		case <-timer.C:
			w.runMonth(ctx, previousBillingMonth(time.Now().UTC()))
		}
	}
}

func durationUntilNextInvoiceRun(now time.Time) time.Duration {
	next := time.Date(now.Year(), now.Month(), 1, 0, 15, 0, 0, time.UTC)
	if !now.Before(next) {
		next = next.AddDate(0, 1, 0)
	}
	return next.Sub(now)
}

func previousBillingMonth(now time.Time) time.Time {
	u := now.UTC()
	return time.Date(u.Year(), u.Month(), 1, 0, 0, 0, 0, time.UTC).AddDate(0, -1, 0)
}

func (w *InvoiceWorker) runMonth(ctx context.Context, month time.Time) {
	opCtx, cancel := context.WithTimeout(ctx, 30*time.Minute)
	defer cancel()

	acquired, err := w.service.tryInvoiceCronLock(opCtx)
	if err != nil {
		return
	}
	if !acquired {
		return
	}
	defer w.service.releaseInvoiceCronLock(opCtx)

	const pageSize int32 = 200
	var offset int32
	for {
		ids, err := w.service.ListCustomerIDs(opCtx, pageSize, offset)
		if err != nil {
			return
		}
		if len(ids) == 0 {
			break
		}
		for _, customerID := range ids {
			inv, genErr := w.service.GenerateInvoice(opCtx, customerID, month)
			if genErr != nil {
				if errors.Is(genErr, ErrNoSpend) {
					continue
				}
				continue
			}
			_ = w.service.DeliverInvoice(opCtx, inv)
		}
		if len(ids) < int(pageSize) {
			break
		}
		offset += pageSize
	}
}

func (w *InvoiceWorker) RunInvoiceMonthForTest(ctx context.Context, month time.Time) {
	if w != nil {
		w.runMonth(ctx, month)
	}
}

const invoiceCronLockKey = int64(0x657370785f696e76)

func (service *Service) tryInvoiceCronLock(ctx context.Context) (bool, error) {
	var ok bool
	err := service.pool.QueryRow(ctx, `SELECT pg_try_advisory_lock($1)`, invoiceCronLockKey).Scan(&ok)
	return ok, err
}

func (service *Service) releaseInvoiceCronLock(ctx context.Context) {
	if service == nil || service.pool == nil {
		return
	}
	ctx, cancel := context.WithTimeout(ctx, advisoryUnlockTimeout)
	defer cancel()
	_, err := service.pool.Exec(ctx, `SELECT pg_advisory_unlock($1)`, invoiceCronLockKey)
	if err == nil {
		return
	}
	if errors.Is(err, context.DeadlineExceeded) || ctx.Err() == context.DeadlineExceeded {
		slog.Error("invoice cron advisory unlock timed out", "error", err)
		return
	}
	slog.Warn("invoice cron advisory unlock failed", "error", err)
}

func (service *Service) GenerateInvoiceForCustomers(ctx context.Context, customerIDs []uuid.UUID, month time.Time) {
	for _, id := range customerIDs {
		inv, err := service.GenerateInvoice(ctx, id, month)
		if err == nil {
			_ = service.DeliverInvoice(ctx, inv)
		}
	}
}
