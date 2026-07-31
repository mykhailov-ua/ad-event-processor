package fraud

import "errors"

var (
	ErrScorerNotRegistered = errors.New("scorer not registered")
)
