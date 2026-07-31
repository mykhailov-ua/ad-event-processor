package domain

import "errors"

var (
	ErrMigrationFenced        = errors.New("campaign debit fenced")
	ErrFreqLimitExceeded      = errors.New("frequency limit exceeded")
	ErrEmergencyBreakerActive = errors.New("service temporarily unavailable (emergency breaker active)")
)
