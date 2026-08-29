package worker

import (
	"errors"
	"fmt"
)

var errValidation = errors.New("validation error")

func serviceUnavailable() error {
	return fmt.Errorf("%w: %s", errValidation, "service unavailable")
}
