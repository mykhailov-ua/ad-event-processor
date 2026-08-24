package outbox

import (
	"time"

	"ad-event-processor/internal/metrics"
)


const PollActiveInterval = 20 * time.Millisecond


const PollIdleMax = 250 * time.Millisecond


const WorkerTimeout = 30 * time.Second


type PollBackoff struct {
	idle time.Duration
}


func NewPollBackoff() *PollBackoff {
	return &PollBackoff{idle: PollActiveInterval}
}


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
