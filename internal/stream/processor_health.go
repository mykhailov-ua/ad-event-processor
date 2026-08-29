package stream

import "sync/atomic"

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
