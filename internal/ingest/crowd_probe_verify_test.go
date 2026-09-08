package ingest

import (
	"testing"

	"ad-event-processor/internal/domain"
	"ad-event-processor/internal/track"
	"ad-event-processor/pkg/crowdprobe"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func verifySOPTelemetry() []track.SafePageVerifyEvent {
	out := make([]track.SafePageVerifyEvent, 20)
	for i := range out {
		t := "mousemove"
		switch i % 4 {
		case 0:
			t = "pointerdown"
		case 1:
			t = "scroll"
		case 2:
			t = "click"
		case 3:
			t = "visibilitychange"
		}
		out[i] = track.SafePageVerifyEvent{
			T: t, TS: int64(i * 80), X: i * 12, Y: i * 12,
			FX: float64(i * 12), FY: float64(i * 12), Trusted: 1,
		}
	}
	return out
}

func TestCrowdProbeVerifySession_holdoutComputesFromVerifyEvents(t *testing.T) {
	telemetry := track.BehaviorTelemetryFromVerify(verifySOPTelemetry())
	feat := crowdprobe.ComputeSessionFeatures(telemetry, domain.AntifraudSnapshot{
		FooterReachMs: 1100,
		ScrollCVMilli: 20,
		DwellMs:       1800,
	})
	result := crowdprobe.Score(feat, 3, 500)
	require.GreaterOrEqual(t, int(result.BehaviorScore), crowdprobe.BehaviorSignalThreshold)
	assert.True(t, result.BehaviorHit)
}

func TestCrowdProbeVerifySession_holdoutHumanVerifyStaysLower(t *testing.T) {
	events := humanMouseEvents(18)
	telemetry := track.BehaviorTelemetryFromVerify(events)
	feat := crowdprobe.ComputeSessionFeatures(telemetry, domain.AntifraudSnapshot{
		FooterReachMs: 8000,
		ScrollCVMilli: 180,
	})
	result := crowdprobe.Score(feat, 0, 0)
	assert.Less(t, int(result.BehaviorScore), crowdprobe.BehaviorSignalThreshold)
}
