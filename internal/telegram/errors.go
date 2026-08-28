package telegram

type validationError string

func (e validationError) Error() string { return string(e) }

func errValidation(msg string) error { return validationError(msg) }
