package reconciliation

import "errors"

var (
	ErrPostgresGateRejected = errors.New("postgres gate rejected")
	ErrInvalidServiceFilter = errors.New("invalid service filter")
)
