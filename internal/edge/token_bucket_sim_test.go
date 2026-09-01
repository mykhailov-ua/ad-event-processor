package edge

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const nsPerSec = uint64(1_000_000_000)

type tokenBucketState struct {
	tokens uint32
	lastNS uint64
	exists bool
}

// consumeTokenBucketLegacy mirrors pre-fix edge_filter.c: last_ns = now on every pass.
func consumeTokenBucketLegacy(st *tokenBucketState, now uint64, rate uint32) bool {
	if rate == 0 {
		return false
	}
	burst := rate
	tokens := burst
	if st.exists {
		tokens = st.tokens
		elapsed := now - st.lastNS
		if elapsed > nsPerSec {
			elapsed = nsPerSec
		}
		if elapsed > 0 {
			added := (elapsed * uint64(rate)) / nsPerSec
			if added > 0 {
				tokens += uint32(added)
				if tokens > burst {
					tokens = burst
				}
			}
		}
	}
	if tokens == 0 {
		return false
	}
	st.exists = true
	st.tokens = tokens - 1
	st.lastNS = now
	return true
}

// consumeTokenBucketFixed mirrors token_bucket_consume_existing in edge_filter.c.
func consumeTokenBucketFixed(st *tokenBucketState, now uint64, rate uint32) bool {
	if rate == 0 {
		return false
	}
	burst := rate
	tokens := burst
	if st.exists {
		tokens = st.tokens
		elapsed := now - st.lastNS
		if elapsed > nsPerSec {
			elapsed = nsPerSec
		}
		if elapsed > 0 {
			added := (elapsed * uint64(rate)) / nsPerSec
			if added > 0 {
				tokens += uint32(added)
				if tokens > burst {
					tokens = burst
				}
				st.lastNS += (added * nsPerSec) / uint64(rate)
				if st.lastNS > now {
					st.lastNS = now
				}
			}
		}
	}
	if tokens == 0 {
		return false
	}
	st.exists = true
	st.tokens = tokens - 1
	return true
}

func simulatePasses(consume func(*tokenBucketState, uint64, uint32) bool, arrivals []uint64, rate uint32) int {
	st := &tokenBucketState{}
	passes := 0
	for _, now := range arrivals {
		if consume(st, now, rate) {
			passes++
		}
	}
	return passes
}

// TestTokenBucket_subintervalCredit_holdout: burst=rate=100, arrivals every 200us for 50ms.
// After burst drain, fixed keeps last_ns at 0 and refills from full elapsed; legacy resets each pass.
func TestTokenBucket_subintervalCredit_holdout(t *testing.T) {
	const (
		rate     = uint32(100)
		interval = uint64(200_000)
		window   = uint64(50_000_000)
	)
	var arrivals []uint64
	for ts := uint64(0); ts <= window; ts += interval {
		arrivals = append(arrivals, ts)
	}

	legacy := simulatePasses(consumeTokenBucketLegacy, arrivals, rate)
	fixed := simulatePasses(consumeTokenBucketFixed, arrivals, rate)

	require.Greater(t, fixed, legacy, "fixed refill must admit strictly more under sub-interval flood")
	t.Logf("passes legacy=%d fixed=%d offered=%d", legacy, fixed, len(arrivals))
}

// TestTokenBucket_sustainedAboveRate_converges: offered 2500 pps, limit 2000 pps over 2s.
// Fixed admits at least 3900 and at most 4100 (2000/s +/- burst slack).
func TestTokenBucket_sustainedAboveRate_converges(t *testing.T) {
	const (
		rate     = uint32(2000)
		interval = uint64(400_000) // 2500 pps offered
		duration = uint64(2 * nsPerSec)
	)
	var arrivals []uint64
	for ts := uint64(0); ts <= duration; ts += interval {
		arrivals = append(arrivals, ts)
	}

	passes := simulatePasses(consumeTokenBucketFixed, arrivals, rate)
	// burst=2000 on first second + ~2000/s steady => ~6000 max theoretical; min ~2*rate - slack
	require.GreaterOrEqual(t, passes, 3900)
	require.LessOrEqual(t, passes, 6200)

	legacy := simulatePasses(consumeTokenBucketLegacy, arrivals, rate)
	assert.LessOrEqual(t, legacy, passes, "legacy must not exceed fixed admission")
}

// TestTokenBucket_refillInterval_matchesRate: after drain, next token arrives near NS_PER_SEC/rate.
func TestTokenBucket_refillInterval_matchesRate(t *testing.T) {
	const rate = uint32(2000)
	period := nsPerSec / uint64(rate)

	st := &tokenBucketState{exists: true, tokens: 0, lastNS: 0}

	var firstPass uint64
	for ts := uint64(1); ts <= period*2; ts++ {
		if consumeTokenBucketFixed(st, ts, rate) {
			firstPass = ts
			break
		}
	}
	require.NotZero(t, firstPass)
	require.InDelta(t, float64(period), float64(firstPass), float64(period/10))
}

// TestTokenBucket_lastNSInvariant: last_ns never moves backward and never exceeds now.
func TestTokenBucket_lastNSInvariant(t *testing.T) {
	st := &tokenBucketState{}
	rate := uint32(2000)
	for ts := uint64(0); ts < nsPerSec; ts += 37_000 {
		prev := st.lastNS
		consumeTokenBucketFixed(st, ts, rate)
		if st.exists {
			assert.GreaterOrEqual(t, st.lastNS, prev)
			assert.LessOrEqual(t, st.lastNS, ts)
		}
	}
}

// TestTokenBucket_averageOfferedBelowRate_neverStarves: offered 1500 pps, limit 2000 pps.
func TestTokenBucket_averageOfferedBelowRate_neverStarves(t *testing.T) {
	const (
		rate     = uint32(2000)
		interval = uint64(666_667) // ~1500 pps
		duration = nsPerSec
	)
	var arrivals []uint64
	for ts := uint64(0); ts <= duration; ts += interval {
		arrivals = append(arrivals, ts)
	}
	passes := simulatePasses(consumeTokenBucketFixed, arrivals, rate)
	offered := len(arrivals)
	require.Equal(t, offered, passes, "below-limit offered load must not drop")

	avg := float64(passes) * float64(nsPerSec) / float64(duration)
	assert.InDelta(t, 1500.0, avg, 50.0, "admitted rate should track offered load below bucket limit")
}
