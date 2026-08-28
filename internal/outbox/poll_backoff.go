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

func (pb *PollBackoff) Next(processed int) time.Duration {
	if processed > 0 {
		pb.idle = PollActiveInterval
		metrics.OutboxPollIntervalMs.Observe(float64(PollActiveInterval) / float64(time.Millisecond))
		return 0
	}
	if pb.idle < PollActiveInterval {
		pb.idle = PollActiveInterval
	}
	next := pb.idle * 2
	if next > PollIdleMax {
		next = PollIdleMax
	}
	pb.idle = next
	metrics.OutboxPollIntervalMs.Observe(float64(next) / float64(time.Millisecond))
	return next
}
