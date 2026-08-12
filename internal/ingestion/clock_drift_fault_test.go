package ingestion

import (
	"github.com/bidshard/ad-event-processor/pkg/faultproof"

	"context"
	"net/http"
	"strconv"
	"testing"
	"time"

	"github.com/bidshard/ad-event-processor/internal/domain"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const (
	clockDriftSeconds       = 3600
	clockDriftWallSleep     = 5 * time.Second
	clockDriftFilterTimeout = 100 * time.Millisecond
	clockDriftTTCMin        = 3 * time.Second
)

func TestFault_ClockDriftMonotonicTTC(t *testing.T) {
	if testing.Short() {
		t.Skip("fault integration test")
	}

	infra, cleanup := setupAdsFaultInfra(t)
	defer cleanup()

	stack := startAdsIngestStackWithFilterTimeout(t, infra, "ads-fault-clock-drift", 100)
	defer stack.Close(t)
	stack.UnifiedFilter.SetTTCMin(clockDriftTTCMin)

	ctx := context.Background()
	const userID = "clock-drift-user"
	streamLenBefore, err := infra.Redis.XLen(ctx, stack.Stream).Result()
	require.NoError(t, err)

	require.Equal(t, http.StatusAccepted, postFaultImpression(t, stack.Handler, stack.CampaignID, userID))

	impTSKey := "imp_ts:" + userID + ":" + stack.CampaignID.String()
	impTS, err := infra.Redis.Get(ctx, impTSKey).Int64()
	require.NoError(t, err)

	time.Sleep(clockDriftWallSleep)

	restore, driftMethod, err := applyFaultClockDrift(time.Duration(clockDriftSeconds) * time.Second)
	require.NoError(t, err)
	defer restore()

	deadlineCtx := attachFilterDeadline(context.Background(), clockDriftFilterTimeout)
	time.Sleep(50 * time.Millisecond)
	require.False(t, filterDeadlineExceeded(deadlineCtx),
		"monotonic filter deadline must not expire after wall-clock drift")

	eventsBefore := countFaultCampaignEvents(t, infra.Pool, stack.CampaignID)
	status := postFaultTrack(t, stack.Handler, stack.CampaignID, "click", userID, uuid.NewString())
	require.Equal(t, http.StatusAccepted, status)

	streamLenAfter, err := infra.Redis.XLen(ctx, stack.Stream).Result()
	require.NoError(t, err)
	require.Equal(t, streamLenBefore+2, streamLenAfter,
		"impression and click must reach Redis stream (low_ttc aborts before XADD)")

	nowMs := cachedUnixMilli.Load()
	elapsedMs := nowMs - impTS
	require.GreaterOrEqual(t, elapsedMs, clockDriftTTCMin.Milliseconds(),
		"TTC window must exceed configured minimum after drift")

	require.Eventually(t, func() bool {
		return countFaultCampaignEvents(t, infra.Pool, stack.CampaignID) > eventsBefore
	}, 10*time.Second, 100*time.Millisecond, "click must settle to Postgres after drift")

	faultproof.Log(t, "clock_drift_monotonic_safety", map[string]string{
		"drift_seconds": strconv.Itoa(clockDriftSeconds),
		"ttc_passed":    "true",
		"drift_method":  driftMethod,
		"filter_ms":     strconv.Itoa(100),
	})
}

func TestFault_ClockDrift_UDPTimePacket(t *testing.T) {
	if testing.Short() {
		t.Skip("fault integration test")
	}

	infra, cleanup := setupAdsFaultInfra(t)
	defer cleanup()

	stack := startAdsIngestStackWithFilterTimeout(t, infra, "ads-fault-udp-clock", 100)
	defer stack.Close(t)
	stack.UnifiedFilter.SetTTCMin(clockDriftTTCMin)

	const userID = "udp-clock-user"
	require.Equal(t, http.StatusAccepted, postFaultImpression(t, stack.Handler, stack.CampaignID, userID))

	time.Sleep(clockDriftWallSleep)

	restore, driftMethod, err := applyFaultClockDrift(time.Duration(clockDriftSeconds) * time.Second)
	require.NoError(t, err)
	defer restore()

	shiftedMs := cachedUnixMilli.Load()
	hdr := &UDPHeader{
		CoarseTimeNs: shiftedMs * int64(time.Millisecond),
		EpochID:      2,
	}
	var limits UDPControlLimits
	limits.NumShards = 1
	limits.Limits[0] = 50_000
	applyUDPCoarseTime(hdr.CoarseTimeNs)

	deadlineCtx := attachFilterDeadline(context.Background(), clockDriftFilterTimeout)
	require.False(t, filterDeadlineExceeded(deadlineCtx))

	status := postFaultTrack(t, stack.Handler, stack.CampaignID, "click", userID, uuid.NewString())
	require.Equal(t, http.StatusAccepted, status)

	faultproof.Log(t, "clock_drift_udp_time", map[string]string{
		"drift_seconds": strconv.Itoa(clockDriftSeconds),
		"drift_method":  driftMethod,
		"udp_clamp_ms":  "50",
		"ttc_passed":    "true",
	})
}

func applyFaultClockDrift(d time.Duration) (restore func(), method string, err error) {
	if restore, err := shiftSystemClock(d); err == nil {
		return restore, "system_clock", nil
	}
	restore, err = shiftCachedWallClock(d)
	if err != nil {
		return nil, "", err
	}
	return restore, "cached_wall_clock", nil
}

func shiftCachedWallClock(d time.Duration) (restore func(), err error) {
	SetClockRefreshPaused(true)
	before := cachedUnixMilli.Load()
	shifted := before + d.Milliseconds()
	cachedUnixMilli.Store(shifted)
	cachedUnixMilliAny.Store(shifted)
	t := time.UnixMilli(shifted).UTC()
	cachedNowUTC.Store(&t)

	return func() {
		cachedUnixMilli.Store(before)
		cachedUnixMilliAny.Store(before)
		storeCachedNowUTC()
		SetClockRefreshPaused(false)
	}, nil
}

func TestClockDrift_filterDeadlineSurvivesWallShift(t *testing.T) {
	ctx := attachFilterDeadline(context.Background(), clockDriftFilterTimeout)
	restore, err := shiftCachedWallClock(time.Duration(clockDriftSeconds) * time.Second)
	require.NoError(t, err)
	defer restore()

	time.Sleep(30 * time.Millisecond)
	assert.False(t, filterDeadlineExceeded(ctx))

	evt := &domain.Event{}
	engine := NewFilterEngine(clockDriftFilterTimeout, &countingFilter{})
	err = engine.Check(ctx, evt)
	assert.NoError(t, err)
}
