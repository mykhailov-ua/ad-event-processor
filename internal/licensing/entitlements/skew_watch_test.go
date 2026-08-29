package entitlements_test

import (
	"testing"
	"time"

	"ad-event-processor/internal/licensing/entitlements"

	"github.com/stretchr/testify/require"
)

func TestSkewWatch_ClockRewind(t *testing.T) {
	entitlements.ResetSkewWatchForTest()
	t.Cleanup(entitlements.ResetSkewWatchForTest)

	wall := time.Date(2026, 6, 1, 12, 0, 0, 0, time.UTC)
	mono := time.Duration(0)
	entitlements.SetClockSampleHookForTest(func() (time.Time, time.Duration) {
		return wall, mono
	})

	sw := entitlements.NewSkewWatch(time.Minute)
	require.False(t, sw.Check(wall, mono))

	mono = time.Hour
	wall = wall.Add(-30 * 24 * time.Hour)
	require.True(t, sw.Check(wall, mono))
	require.True(t, sw.Violated())
}

func TestSkewWatch_StepBackwardBetweenSamples(t *testing.T) {
	entitlements.ResetSkewWatchForTest()
	t.Cleanup(entitlements.ResetSkewWatchForTest)

	wall := time.Date(2026, 6, 1, 12, 0, 0, 0, time.UTC)
	mono := time.Duration(0)
	entitlements.SetClockSampleHookForTest(func() (time.Time, time.Duration) {
		return wall, mono
	})

	sw := entitlements.NewSkewWatch(time.Minute)
	require.False(t, sw.Check(wall, mono))

	mono += time.Hour
	wall = wall.Add(-2 * time.Minute)
	require.True(t, sw.Check(wall, mono))
}

func TestSkewWatch_AllowsNTPSkewWithinThreshold(t *testing.T) {
	entitlements.ResetSkewWatchForTest()
	t.Cleanup(entitlements.ResetSkewWatchForTest)

	wall := time.Date(2026, 6, 1, 12, 0, 0, 0, time.UTC)
	mono := time.Duration(0)
	entitlements.SetClockSampleHookForTest(func() (time.Time, time.Duration) {
		return wall, mono
	})

	sw := entitlements.NewSkewWatch(5 * time.Minute)
	require.False(t, sw.Check(wall, mono))

	mono = time.Hour
	wall = wall.Add(59 * time.Minute)
	require.False(t, sw.Check(wall, mono))
	require.False(t, sw.Violated())
}

func TestEvaluateClockSkew_globalWatch(t *testing.T) {
	entitlements.ResetSkewWatchForTest()
	t.Cleanup(entitlements.ResetSkewWatchForTest)

	wall := time.Date(2026, 6, 1, 12, 0, 0, 0, time.UTC)
	mono := time.Duration(0)
	entitlements.SetClockSampleHookForTest(func() (time.Time, time.Duration) {
		return wall, mono
	})
	entitlements.ConfigureSkewWatch(entitlements.SkewWatchOptions{
		Enabled:   true,
		Interval:  time.Hour,
		Threshold: time.Minute,
	})
	require.False(t, entitlements.EvaluateClockSkew())

	mono = 2 * time.Hour
	wall = wall.Add(-30 * 24 * time.Hour)
	require.True(t, entitlements.EvaluateClockSkew())
	require.True(t, entitlements.ClockSkewViolated())
}
