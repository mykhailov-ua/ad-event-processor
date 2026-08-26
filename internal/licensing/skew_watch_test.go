package licensing_test

import (
	"testing"
	"time"

	"ad-event-processor/internal/licensing"

	"github.com/stretchr/testify/require"
)

func TestSkewWatch_ClockRewind(t *testing.T) {
	licensing.ResetSkewWatchForTest()
	t.Cleanup(licensing.ResetSkewWatchForTest)

	wall := time.Date(2026, 6, 1, 12, 0, 0, 0, time.UTC)
	mono := time.Duration(0)
	licensing.SetClockSampleHookForTest(func() (time.Time, time.Duration) {
		return wall, mono
	})

	sw := licensing.NewSkewWatch(time.Minute)
	require.False(t, sw.Check(wall, mono))

	mono = time.Hour
	wall = wall.Add(-30 * 24 * time.Hour)
	require.True(t, sw.Check(wall, mono))
	require.True(t, sw.Violated())
}

func TestSkewWatch_StepBackwardBetweenSamples(t *testing.T) {
	licensing.ResetSkewWatchForTest()
	t.Cleanup(licensing.ResetSkewWatchForTest)

	wall := time.Date(2026, 6, 1, 12, 0, 0, 0, time.UTC)
	mono := time.Duration(0)
	licensing.SetClockSampleHookForTest(func() (time.Time, time.Duration) {
		return wall, mono
	})

	sw := licensing.NewSkewWatch(time.Minute)
	require.False(t, sw.Check(wall, mono))

	mono += time.Hour
	wall = wall.Add(-2 * time.Minute)
	require.True(t, sw.Check(wall, mono))
}

func TestSkewWatch_AllowsNTPSkewWithinThreshold(t *testing.T) {
	licensing.ResetSkewWatchForTest()
	t.Cleanup(licensing.ResetSkewWatchForTest)

	wall := time.Date(2026, 6, 1, 12, 0, 0, 0, time.UTC)
	mono := time.Duration(0)
	licensing.SetClockSampleHookForTest(func() (time.Time, time.Duration) {
		return wall, mono
	})

	sw := licensing.NewSkewWatch(5 * time.Minute)
	require.False(t, sw.Check(wall, mono))

	mono = time.Hour
	wall = wall.Add(59 * time.Minute)
	require.False(t, sw.Check(wall, mono))
	require.False(t, sw.Violated())
}

func TestEvaluateClockSkew_globalWatch(t *testing.T) {
	licensing.ResetSkewWatchForTest()
	t.Cleanup(licensing.ResetSkewWatchForTest)

	wall := time.Date(2026, 6, 1, 12, 0, 0, 0, time.UTC)
	mono := time.Duration(0)
	licensing.SetClockSampleHookForTest(func() (time.Time, time.Duration) {
		return wall, mono
	})
	licensing.ConfigureSkewWatch(licensing.SkewWatchOptions{
		Enabled:   true,
		Interval:  time.Hour,
		Threshold: time.Minute,
	})
	require.False(t, licensing.EvaluateClockSkew())

	mono = 2 * time.Hour
	wall = wall.Add(-30 * 24 * time.Hour)
	require.True(t, licensing.EvaluateClockSkew())
	require.True(t, licensing.ClockSkewViolated())
}
