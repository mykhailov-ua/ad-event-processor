package track

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func humanizedBehaviorEvents(n int) []SafePageVerifyEvent {
	out := make([]SafePageVerifyEvent, n)
	for i := range n {
		fx := 100.25 + float64(i)*3.17
		fy := 200.75 + float64(i)*2.33
		out[i] = SafePageVerifyEvent{
			T:       "mousemove",
			TS:      int64(i * 137),
			X:       int(fx),
			Y:       int(fy),
			FX:      fx,
			FY:      fy,
			Trusted: 1,
		}
	}
	out[0].T = "pointerdown"
	out[n/2].T = "click"
	out[n-1].T = "keydown"
	return out
}

func TestScoreSafePageBehavior_holdoutCDPLinearPath(t *testing.T) {
	baseline := ScoreSafePageBehavior(humanizedBehaviorEvents(safePageVerifyMinEvents))
	require.Greater(t, baseline, safePageVerifyMinEvents+3)

	linear := make([]SafePageVerifyEvent, safePageVerifyMinEvents)
	for i := range linear {
		linear[i] = SafePageVerifyEvent{
			T:       "mousemove",
			TS:      int64(i * 50),
			X:       i * 4,
			Y:       i * 4,
			FX:      float64(i * 4),
			FY:      float64(i * 4),
			Trusted: 1,
		}
	}
	assert.Less(t, ScoreSafePageBehavior(linear), baseline)
}

func TestScoreSafePageBehavior_holdoutHumanizedBeatsBaseline(t *testing.T) {
	baseline := ScoreSafePageBehavior(humanizedBehaviorEvents(safePageVerifyMinEvents))
	require.Greater(t, baseline, safePageVerifyMinEvents+3)

	synthetic := make([]SafePageVerifyEvent, safePageVerifyMinEvents)
	for i := range synthetic {
		synthetic[i] = SafePageVerifyEvent{
			T:  "mousemove",
			TS: int64(i * 100),
			X:  i,
			Y:  i,
		}
	}
	assert.Less(t, ScoreSafePageBehavior(synthetic), baseline)
}

func TestScoreSafePageBehavior_holdoutTrustedSubpixelBonus(t *testing.T) {
	events := humanizedBehaviorEvents(safePageVerifyMinEvents)
	score := ScoreSafePageBehavior(events)
	for i := range events {
		events[i].Trusted = 0
		events[i].FX = float64(events[i].X)
		events[i].FY = float64(events[i].Y)
		events[i].TS = int64(i * 100)
	}
	assert.Greater(t, score, ScoreSafePageBehavior(events))
}
