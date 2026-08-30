package commandpalette

import "errors"

var (
	errInvalidCustomerID  = errors.New("invalid customer_id")
	errCustomerIDRequired = errors.New("customer_id is required")
	errForbidden          = errors.New("forbidden")
)

func isForbiddenCustomerError(err error) bool {
	return errors.Is(err, errForbidden) || (err != nil && err.Error() == "forbidden")
}
