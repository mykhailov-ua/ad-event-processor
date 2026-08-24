package fraud

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestMicroBatcherConfig_normalized_defaults(t *testing.T) {
	cfg := MicroBatcherConfig{}.normalized()
	require.Equal(t, defaultMicrobatchFlushInterval, cfg.FlushInterval)
	require.Equal(t, defaultMicrobatchMaxLagSec, cfg.MaxStreamLagSec)
}

func TestMicroBatcherConfig_normalized_clampsInvalid(t *testing.T) {
	cfg := MicroBatcherConfig{
		FlushInterval:   -1,
		MaxStreamLagSec: 0,
	}.normalized()
	require.Equal(t, 50*time.Millisecond, cfg.FlushInterval)
	require.Equal(t, 30.0, cfg.MaxStreamLagSec)
}
