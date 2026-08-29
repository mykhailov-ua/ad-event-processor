package fraud

import (
	"errors"

	adminhooks "ad-event-processor/internal/fraud/admin_hooks"
)

var (
	ErrScorerNotRegistered   = errors.New("scorer not registered")
	ErrOutboxBackpressure    = errors.New("management outbox backpressure: pending queue full")
	ErrManagementUnavailable = adminhooks.ErrManagementUnavailable
	ErrInvalidIP             = adminhooks.ErrInvalidIP
)
