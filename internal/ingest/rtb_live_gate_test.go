package ingest

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestEvaluateRtbLiveGate_insufficientShadow(t *testing.T) {
	ResetRtbShadowDiffBuckets()
	gate := EvaluateRtbLiveGate(time.Hour)
	assert.False(t, gate.Ready)
	assert.Contains(t, gate.Reasons, rtbLiveGateInsufficient)
}

func TestEvaluateRtbLiveGate_parityOk(t *testing.T) {
	ResetRtbShadowDiffBuckets()
	b := rtbShadowDiffBucketNow()
	for range 120 {
		b.RecordParityMatchForTest()
	}
	gate := EvaluateRtbLiveGate(time.Hour)
	assert.True(t, gate.Ready, gate.Reasons)
	assert.GreaterOrEqual(t, gate.Shadow.ParityRate, rtbLiveGateMinParityRate)
}
