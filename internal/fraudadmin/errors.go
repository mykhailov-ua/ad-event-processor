package fraudadmin

import "errors"

var (
	ErrValidation            = errors.New("validation")
	ErrFraudDecisionNotFound = errors.New("fraud decision not found")
)

type validationError string

func (e validationError) Error() string {
	return string(e)
}

func (e validationError) Is(target error) bool {
	return target == ErrValidation
}

func ValidationError(msg string) error {
	return validationError(msg)
}
