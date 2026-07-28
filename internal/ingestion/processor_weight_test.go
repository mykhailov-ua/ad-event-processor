package ingestion

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestProcessorWeight_EffectiveReadCount(t *testing.T) {
	ctrl := NewProcessorWeightController(ProcessorWeightConfig{
		NodeID:        "processor-a",
		InstanceLabel: "processor-a",
		Floor:         0.05,
		Ceil:          0.95,
		EpochInterval: time.Second,
	}, nil, nil)

	ctrl.SetWeightForTest(1.0)
	assert.Equal(t, int64(950), ctrl.EffectiveReadCount(1000))

	ctrl.SetWeightForTest(0.25)
	assert.Equal(t, int64(250), ctrl.EffectiveReadCount(1000))

	ctrl.SetWeightForTest(0.01)
	assert.Equal(t, int64(50), ctrl.EffectiveReadCount(1000))
}

func TestProcessorWeight_HTTPPoll(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		require.Equal(t, "/ops/processor-weights", r.URL.Path)
		_ = json.NewEncoder(w).Encode(map[string]any{
			"epoch":     42,
			"epoch_lag": 0,
			"node_weights": []map[string]any{
				{"node_id": "processor-b", "weight": 0.35},
			},
		})
	}))
	defer srv.Close()

	ctrl := NewProcessorWeightController(ProcessorWeightConfig{
		NodeID:        "processor-b",
		InstanceLabel: "processor-b",
		Floor:         0.05,
		Ceil:          0.95,
		EpochInterval: time.Hour,
		WeightsURL:    srv.URL + "/ops/processor-weights",
	}, nil, nil)
	ctrl.refresh()
	assert.InDelta(t, 0.35, ctrl.LocalWeight(), 0.001)
}

func TestProcessorWeight_PgGateDrain(t *testing.T) {
	gate := NewProcessorPgGate(1, 2)
	ctrl := NewProcessorWeightController(ProcessorWeightConfig{
		NodeID:        "processor",
		InstanceLabel: "processor",
		Floor:         0.05,
		Ceil:          0.95,
		EpochInterval: time.Hour,
		DrainPgWait:   10 * time.Millisecond,
	}, gate, nil)
	ctrl.SetWeightForTest(0.8)

	gate.recordWait(25 * time.Millisecond)
	ctrl.refresh()
	assert.InDelta(t, 0.05, ctrl.LocalWeight(), 0.001)
}

func TestProcessorWeight_ThrottleRespectsCancel(t *testing.T) {
	ctrl := NewProcessorWeightController(ProcessorWeightConfig{
		NodeID:        "processor",
		InstanceLabel: "processor",
		Floor:         0.05,
		Ceil:          0.95,
		EpochInterval: time.Second,
	}, nil, nil)
	ctrl.SetWeightForTest(0.1)

	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	start := time.Now()
	ctrl.ThrottleBeforeRead(ctx)
	assert.Less(t, time.Since(start), 50*time.Millisecond)
}

func TestProcessorWeight_TwoReplicaReadSkew(t *testing.T) {
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
	for epoch := 0; epoch < 3; epoch++ {
		for i := 0; i < 100; i++ {
			fastReads += int(fast.EffectiveReadCount(1000))
			slowReads += int(slow.EffectiveReadCount(1000))
		}
	}
	require.Greater(t, slowReads, 0)
	ratio := float64(fastReads) / float64(slowReads)
	assert.Greater(t, ratio, 3.0, "fast replica should read at least 3x more per epoch window")
}
