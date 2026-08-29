package admin_hooks

import "errors"

var (
	ErrManagementUnavailable = errors.New("management blacklist API unavailable")
	ErrInvalidIP             = errors.New("invalid IP address")
)
