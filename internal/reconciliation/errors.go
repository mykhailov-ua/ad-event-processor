package reconciliation

import "errors"

var (
	// ErrPostgresGateRejected: worker tick skipped when WithPostgresLow rejects (hot path priority).
	ErrPostgresGateRejected = errors.New("postgres gate rejected")
	// ErrInvalidServiceFilter: ListRuns service param must be all, management, or payment.
	ErrInvalidServiceFilter = errors.New("invalid service filter")
)
