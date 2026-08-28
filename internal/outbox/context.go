package outbox

import (
	"context"
	"time"
)

func workerContext(parent context.Context, timeout time.Duration) (context.Context, context.CancelFunc) {
	if deadline, ok := parent.Deadline(); ok {
		remaining := time.Until(deadline)
		if remaining <= 0 {
			return context.WithCancel(parent)
		}
		if remaining < timeout {
			timeout = remaining
		}
	}
	return context.WithTimeout(parent, timeout)
}
