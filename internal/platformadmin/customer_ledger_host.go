package platformadmin

import (
	"context"
	"io"

	"ad-event-processor/internal/domain"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

type CustomerLedgerHost interface {
	Pool() *pgxpool.Pool
	ExportChunkMaxBytes() int
	MapCustomerNotFound(err error) error
}

type DisputesHost interface {
	Pool() *pgxpool.Pool
	ListPaymentDisputes(ctx context.Context, customerFilter string, limit, offset int32) (domain.ListDisputesResult, error)
}

type CustomerLedger struct {
	host CustomerLedgerHost
}

func NewCustomerLedger(host CustomerLedgerHost) *CustomerLedger {
	return &CustomerLedger{host: host}
}

type Disputes struct {
	host DisputesHost
}

func NewDisputes(host DisputesHost) *Disputes {
	return &Disputes{host: host}
}

func (c *CustomerLedger) ListCustomerLedger(ctx context.Context, customerID uuid.UUID, limit, offset int32) ([]BalanceLedgerDTO, int64, error) {
	if c == nil || c.host == nil || c.host.Pool() == nil {
		return nil, 0, errPlatformServiceUnavailable()
	}
	return listCustomerLedger(ctx, c.host, customerID, limit, offset)
}

func (c *CustomerLedger) GetCustomerBalance(ctx context.Context, customerID uuid.UUID) (CustomerBalanceDTO, error) {
	if c == nil || c.host == nil || c.host.Pool() == nil {
		return CustomerBalanceDTO{}, errPlatformServiceUnavailable()
	}
	return getCustomerBalance(ctx, c.host, customerID)
}

func (c *CustomerLedger) ExportCustomerLedgerCSV(ctx context.Context, customerID uuid.UUID, cursor int64, w io.Writer) (LedgerExportResult, error) {
	if c == nil || c.host == nil || c.host.Pool() == nil {
		return LedgerExportResult{}, errPlatformServiceUnavailable()
	}
	return exportCustomerLedgerCSV(ctx, c.host, customerID, cursor, w)
}

func (d *Disputes) ListDisputes(ctx context.Context, customerFilter string, limit, offset int32) (DisputeListResult, error) {
	if d == nil || d.host == nil {
		return DisputeListResult{}, errPlatformServiceUnavailable()
	}
	return listDisputes(ctx, d.host, customerFilter, limit, offset)
}
