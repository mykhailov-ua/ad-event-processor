package opsadmin

import (
	"errors"
	"fmt"
)

var ErrInvalidQuery = errors.New("invalid query")

func errInvalidQuery(msg string) error {
	return fmt.Errorf("%w: %s", ErrInvalidQuery, msg)
}

func errValidation(msg string) error {
	return fmt.Errorf("%w: %s", ErrInvalidQuery, msg)
}

var ErrDLQEntryNotFound = errors.New("dlq entry not found")
