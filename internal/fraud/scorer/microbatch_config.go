package scorer

import "time"

const (
	defaultMicrobatchFlushInterval = 50 * time.Millisecond
	defaultMicrobatchMaxLagSec     = 30.0
)

type MicroBatcherConfig struct {
	FlushInterval   time.Duration
	MaxStreamLagSec float64
}

func DefaultMicroBatcherConfig() MicroBatcherConfig {
	return MicroBatcherConfig{
		FlushInterval:   defaultMicrobatchFlushInterval,
		MaxStreamLagSec: defaultMicrobatchMaxLagSec,
	}
}

func (c MicroBatcherConfig) normalized() MicroBatcherConfig {
	out := c
	if out.FlushInterval <= 0 {
		out.FlushInterval = defaultMicrobatchFlushInterval
	}
	if out.MaxStreamLagSec <= 0 {
		out.MaxStreamLagSec = defaultMicrobatchMaxLagSec
	}
	return out
}
