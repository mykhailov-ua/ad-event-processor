package outbox

import (
	"time"

	"github.com/bidshard/ad-event-processor/internal/metrics"
)

// PollActiveInterval is the outbox worker repoll delay after work is found.
const PollActiveInterval = 20 * time.Millisecond

// PollIdleMax caps outbox worker idle backoff.
const PollIdleMax = 250 * time.Millisecond

// WorkerTimeout bounds outbox DB reads and lag metric queries.
const WorkerTimeout = 30 * time.Second

// PollBackoff implements active-then-idle polling for outbox workers.
type PollBackoff struct {
	idle time.Duration
}

// NewPollBackoff returns a backoff tracker starting at PollActiveInterval.
func NewPollBackoff() *PollBackoff {
	return &PollBackoff{idle: PollActiveInterval}
}

// Next returns the sleep duration before the next poll.
// Zero means immediate repoll when processed > 0.
func (backoff *PollBackoff) Next(processed int) time.Duration {
	if processed > 0 {
		backoff.idle = PollActiveInterval
		metrics.OutboxPollIntervalMs.Observe(float64(PollActiveInterval) / float64(time.Millisecond))
		return 0
	}
	if backoff.idle < PollActiveInterval {
		backoff.idle = PollActiveInterval
	}
	next := backoff.idle * 2
	if next > PollIdleMax {
		next = PollIdleMax
	}
	backoff.idle = next
	metrics.OutboxPollIntervalMs.Observe(float64(next) / float64(time.Millisecond))
	return next
}
