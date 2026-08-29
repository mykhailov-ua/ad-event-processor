package outbox

import "context"

type Alerter interface {
	OutboxStuckThresholdSec() int
	AlertOutboxStuck(ctx context.Context, pending int64, oldestSeconds float64)
}
