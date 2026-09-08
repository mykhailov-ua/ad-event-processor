package crowdprobe

import (
	"testing"

	"ad-event-processor/internal/domain"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func sopCorpusEvents() []domain.BehaviorTelemetryEvent {
	out := make([]domain.BehaviorTelemetryEvent, 20)
	for i := range out {
		out[i] = domain.BehaviorTelemetryEvent{
			T:       sopEventType(i),
			TS:      int64(i * 80),
			X:       i * 12,
			Y:       i * 12,
			FX:      float32(i * 12),
			FY:      float32(i * 12),
			Trusted: 1,
		}
	}
	return out
}

func sopEventType(i int) string {
	switch i % 4 {
	case 0:
		return "pointerdown"
	case 1:
		return "scroll"
	case 2:
		return "click"
	default:
		return "visibilitychange"
	}
}

func organicCorpusEvents() []domain.BehaviorTelemetryEvent {
	out := make([]domain.BehaviorTelemetryEvent, 24)
	for i := range out {
		out[i] = domain.BehaviorTelemetryEvent{
			T:       organicEventType(i),
			TS:      int64(i*140 + (i%5)*17),
			X:       100 + i*7 + (i%3)*11,
			Y:       200 + i*5 + (i%4)*7,
			FX:      float32(100+i*7+(i%3)*11) + 0.37,
			FY:      float32(200+i*5+(i%4)*7) + 0.52,
			Trusted: 1,
		}
	}
	return out
}

func organicEventType(i int) string {
	types := []string{"mousemove", "pointerdown", "scroll", "keydown", "click", "mousemove", "visibilitychange"}
	return types[i%len(types)]
}

func sopAntifraudSnap() domain.AntifraudSnapshot {
	return domain.AntifraudSnapshot{
		DwellMs:           1800,
		FooterReachMs:     1200,
		ScrollCVMilli:     18,
		PointerCVMilli:    22,
		PointerDtCVMilli:  15,
		TrustedRatioMilli: 950,
	}
}

func organicAntifraudSnap() domain.AntifraudSnapshot {
	return domain.AntifraudSnapshot{
		DwellMs:           8200,
		FooterReachMs:     6400,
		ScrollCVMilli:     210,
		PointerCVMilli:    180,
		PointerDtCVMilli:  160,
		TrustedRatioMilli: 920,
	}
}

func TestCorpus_holdoutSOPScoresAboveThreshold(t *testing.T) {
	feat := ComputeSessionFeatures(sopCorpusEvents(), sopAntifraudSnap())
	result := Score(feat, 3, 500)
	require.GreaterOrEqual(t, int(result.BehaviorScore), BehaviorSignalThreshold)
	assert.True(t, result.BehaviorHit)
}

func TestCorpus_holdoutOrganicStaysClean(t *testing.T) {
	feat := ComputeSessionFeatures(organicCorpusEvents(), organicAntifraudSnap())
	result := Score(feat, 0, 0)
	assert.Less(t, int(result.BehaviorScore), BehaviorSignalThreshold)
	assert.False(t, result.BehaviorHit)
	assert.False(t, result.TimingHit)
}

func TestCorpus_holdoutSOPTimingSignal(t *testing.T) {
	feat := ComputeSessionFeatures(sopCorpusEvents(), sopAntifraudSnap())
	result := Score(feat, 0, 0)
	assert.True(t, result.TimingHit)
}
