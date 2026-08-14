package ingestion

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestFault_ProcessorWeightDrain(t *testing.T) {
	if testing.Short() {
		t.Skip("fault integration test")
	}

	fast := NewProcessorWeightController(ProcessorWeightConfig{
		NodeID: "processor", InstanceLabel: "processor",
		Floor: 0.05, Ceil: 0.95, EpochInterval: time.Second,
	}, nil, nil)
	slow := NewProcessorWeightController(ProcessorWeightConfig{
		NodeID: "processor-1", InstanceLabel: "processor-1",
		Floor: 0.05, Ceil: 0.95, EpochInterval: time.Second,
	}, nil, nil)
	fast.SetWeightForTest(0.9)
	slow.SetWeightForTest(0.1)

	var fastReads, slowReads int
	for range 3 {
		for range 200 {
			fastReads += int(fast.EffectiveReadCount(500))
			slowReads += int(slow.EffectiveReadCount(500))
		}
	}
	require.Greater(t, slowReads, 0)
	lagRatio := float64(fastReads) / float64(slowReads)
	assert.Greater(t, lagRatio, 3.0)

	slowGate := NewProcessorPgGate(1, 2)
	slowDrain := NewProcessorWeightController(ProcessorWeightConfig{
		NodeID: "processor-1", InstanceLabel: "processor-1",
		Floor: 0.05, Ceil: 0.95, EpochInterval: time.Second,
		DrainPgWait: 5 * time.Millisecond,
	}, slowGate, nil)
	slowDrain.SetWeightForTest(0.75)
	slowGate.recordWait(20 * time.Millisecond)
	slowDrain.refresh()
	assert.InDelta(t, 0.05, slowDrain.LocalWeight(), 0.001)

	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Millisecond)
	defer cancel()
	slowDrain.ThrottleBeforeRead(ctx)

	t.Logf("fault_proof fault=processor_weight_drain lag_ratio=%.2f fast_reads=%d slow_reads=%d drain_weight=%.2f baseline_ok=true",
		lagRatio, fastReads, slowReads, slowDrain.LocalWeight())
}
