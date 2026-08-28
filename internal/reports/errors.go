package reports

import "errors"

var ErrForbidden = errors.New("forbidden")

type invalidQueryError string

func (e invalidQueryError) Error() string { return string(e) }

func errInvalidQuery(msg string) error { return invalidQueryError(msg) }

type validationError string

func (e validationError) Error() string { return string(e) }

func errValidation(msg string) error { return validationError(msg) }
