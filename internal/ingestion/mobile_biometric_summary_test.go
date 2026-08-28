package ingestion

import (
	"testing"

	"ad-event-processor/internal/domain"

	"github.com/stretchr/testify/assert"
)

func TestSummarizeMobileBiometrics_holdoutFlatGyro(t *testing.T) {
	events := []domain.BehaviorTelemetryEvent{
		{T: "deviceorientation", TS: 1, X: 10, Y: 20},
		{T: "deviceorientation", TS: 2, X: 11, Y: 21},
		{T: "deviceorientation", TS: 3, X: 10, Y: 20},
	}
	sum := summarizeMobileBiometrics(events)
	assert.Equal(t, uint8(3), sum.gyroSamples)
	assert.Equal(t, uint8(1), sum.gyroFlat)
}

func TestSummarizeMobileBiometrics_holdoutTouchOnly(t *testing.T) {
	events := []domain.BehaviorTelemetryEvent{
		{T: "touchstart", TS: 1, X: 5, Y: 6},
		{T: "touchmove", TS: 2, X: 7, Y: 8},
	}
	sum := summarizeMobileBiometrics(events)
	assert.Equal(t, uint8(2), sum.touchCount)
	assert.Equal(t, uint8(0), sum.gyroSamples)
	assert.Equal(t, uint8(0), sum.gyroFlat)
}

func TestSummarizeMobileBiometrics_holdoutVariableGyroNotFlat(t *testing.T) {
	events := []domain.BehaviorTelemetryEvent{
		{T: "deviceorientation", TS: 1, X: 0, Y: 0},
		{T: "deviceorientation", TS: 2, X: 30, Y: 40},
		{T: "deviceorientation", TS: 3, X: -20, Y: 10},
	}
	sum := summarizeMobileBiometrics(events)
	assert.Equal(t, uint8(3), sum.gyroSamples)
	assert.Equal(t, uint8(0), sum.gyroFlat)
	assert.Greater(t, sum.gyroVariance, uint16(0))
}

func TestApplyMobileBiometricSummary_holdoutMobileUA(t *testing.T) {
	evt := &domain.Event{
		UA:           "Mozilla/5.0 (iPhone; CPU iPhone OS 17_0 like Mac OS X)",
		TelemetrySet: 1,
		TelemetryEvents: []domain.BehaviorTelemetryEvent{
			{T: "touchstart", TS: 1, X: 1, Y: 2},
		},
	}
	applyMobileBiometricSummary(evt)
	assert.Equal(t, uint8(1), evt.MobileBiometricSet)
	assert.Equal(t, uint8(1), evt.MobileBiometricMobile)
	assert.Equal(t, uint8(1), evt.MobileTouchCount)
}
