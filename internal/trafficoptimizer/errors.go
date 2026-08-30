package trafficoptimizer

import "errors"

var (
	ErrNotImplemented = errors.New("traffic optimizer dry-run preview not implemented")
	ErrUnavailable    = errors.New("traffic optimizer service unavailable")
)

const NotImplementedMessage = "traffic optimizer dry-run preview not implemented"
