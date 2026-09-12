package ledger

import "errors"

var errEnforcementUnavailable = errors.New("margin guard enforcement unavailable")

func ErrEnforcementUnavailable() error {
	return errEnforcementUnavailable
}
