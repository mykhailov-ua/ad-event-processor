package stream

import "sync/atomic"

// ProcessorHealthState exposes stream lag seconds for processor readiness probes
// (updated from flushBatch on first event CreatedAt).
var ProcessorHealthState struct {
	streamLagSec atomic.Int64
}

func SetProcessorStreamLagSec(sec int64) {
	if sec < 0 {
		sec = 0
	}
	ProcessorHealthState.streamLagSec.Store(sec)
}

func ProcessorStreamLagSec() int64 {
	return ProcessorHealthState.streamLagSec.Load()
}
