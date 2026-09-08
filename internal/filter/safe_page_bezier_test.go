package filter

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func linearMouseVerifyEvents(n int) []SafePageVerifyEvent {
	out := make([]SafePageVerifyEvent, n)
	for i := range n {
		out[i] = SafePageVerifyEvent{
			T:  "mousemove",
			TS: int64(i * 10),
			X:  i * 10,
			Y:  i * 10,
		}
	}
	return out
}

func humanMouseVerifyEvents(n int) []SafePageVerifyEvent {
	out := make([]SafePageVerifyEvent, n)
	for i := range n {
		out[i] = SafePageVerifyEvent{
			T:  "mousemove",
			TS: int64(i * 12),
			X:  i*7 + (i % 3),
			Y:  i*5 + (i % 2),
		}
	}
	return out
}

func TestCheckBezierBot_holdoutLinearSynthetic(t *testing.T) {
	code := CheckBezierBot(linearMouseVerifyEvents(12))
	require.Equal(t, SafePageAttestBezierBot, code)
}

func TestCheckBezierBot_holdoutHumanCurvePasses(t *testing.T) {
	code := CheckBezierBot(humanMouseVerifyEvents(18))
	assert.Equal(t, "", code)
}

func TestCheckBezierBot_holdoutTooFewPoints(t *testing.T) {
	code := CheckBezierBot(linearMouseVerifyEvents(4))
	assert.Equal(t, "", code)
}
