package billingadmin

import (
	"context"

	db "ad-event-processor/internal/domain/db"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

type LedgerHost interface {
	Pool() *pgxpool.Pool
	RequirePgFencing(ctx context.Context) error
	AuditLog(ctx context.Context, q db.Querier, adminID uuid.UUID, action, targetType string, targetID *uuid.UUID, changes, metadata any)
	ErrCustomerNotFound() error
	ErrPaymentTopupNotFound() error
	ErrRefundExceedsTopup() error
	ErrChargebackExceedsTopup() error
	ErrChargebackReversalExceedsWithdrawn() error
	ErrValidation(msg string) error
	IsPgUniqueViolation(err error) bool
}

type LedgerStore struct {
	host LedgerHost
}

func NewLedgerStore(host LedgerHost) *LedgerStore {
	return &LedgerStore{host: host}
}
