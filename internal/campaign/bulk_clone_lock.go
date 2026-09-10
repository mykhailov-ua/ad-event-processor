package campaign

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

func bulkCloneLockKey(customerID uuid.UUID) string {
	return fmt.Sprintf("bulk_clone:%s", customerID.String())
}

// WithBulkCloneCustomerLock serializes bulk-clone for one customer.
func WithBulkCloneCustomerLock(ctx context.Context, pool *pgxpool.Pool, customerID uuid.UUID, fn func() error) error {
	if pool == nil || customerID == uuid.Nil {
		return fn()
	}
	key := bulkCloneLockKey(customerID)
	if _, err := pool.Exec(ctx, `SELECT pg_advisory_lock(hashtext($1))`, key); err != nil {
		return err
	}
	defer func() { _, _ = pool.Exec(context.WithoutCancel(ctx), `SELECT pg_advisory_unlock(hashtext($1))`, key) }()
	return fn()
}
