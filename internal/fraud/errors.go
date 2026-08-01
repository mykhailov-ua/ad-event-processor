package fraud

import "errors"

var (
	ErrScorerNotRegistered   = errors.New("scorer not registered")
	ErrOutboxBackpressure    = errors.New("management outbox backpressure: pending queue full")
	ErrManagementUnavailable = errors.New("management blacklist API unavailable")
	ErrInvalidIP             = errors.New("invalid IP address")
)
