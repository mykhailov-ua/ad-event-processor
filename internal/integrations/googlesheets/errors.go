package googlesheets

import "errors"

var (
	ErrNotConfigured = errors.New("google sheets integration is not configured")
	ErrNotConnected  = errors.New("google sheets is not connected for this operator")
	ErrInvalidState  = errors.New("invalid oauth state")
	ErrStateExpired  = errors.New("oauth state expired")
)
