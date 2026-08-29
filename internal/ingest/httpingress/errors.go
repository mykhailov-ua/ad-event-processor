package httpingress

import "errors"

var (
	ErrIncomplete      = errors.New("incomplete HTTP request")
	ErrInvalid         = errors.New("invalid HTTP request")
	ErrPayloadTooLarge = errors.New("payload too large")
)
