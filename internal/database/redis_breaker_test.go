package database

import (
	"context"
	"errors"
	"net"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	redis "github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRedisBreaker_StartsClosed(t *testing.T) {
	b := NewRedisBreaker(3, 2, 50*time.Millisecond)
	assert.Equal(t, CircuitClosed, b.State())
	assert.True(t, b.Allow())
}

func TestRedisBreaker_TripsAfterThreshold(t *testing.T) {
	b := NewRedisBreaker(3, 2, 50*time.Millisecond)

	b.RecordFailure()
	b.RecordFailure()
	assert.Equal(t, CircuitClosed, b.State())
	assert.True(t, b.Allow())

	b.RecordFailure()
	assert.Equal(t, CircuitOpen, b.State())
	assert.False(t, b.Allow())
}

func TestRedisBreaker_TransitionsToHalfOpen(t *testing.T) {
	b := NewRedisBreaker(1, 2, 20*time.Millisecond)

	b.RecordFailure()
	assert.Equal(t, CircuitOpen, b.State())
	assert.False(t, b.Allow())

	time.Sleep(25 * time.Millisecond)

	assert.True(t, b.Allow())
	assert.Equal(t, CircuitHalfOpen, b.State())

	assert.True(t, b.Allow())
}

func TestRedisBreaker_HalfOpenFailureReopens(t *testing.T) {
	b := NewRedisBreaker(1, 2, 20*time.Millisecond)

	b.RecordFailure()
	assert.Equal(t, CircuitOpen, b.State())

	time.Sleep(25 * time.Millisecond)
	require.True(t, b.Allow())

	b.RecordFailure()
	assert.Equal(t, CircuitOpen, b.State())
	assert.False(t, b.Allow())
}

func TestRedisBreaker_HalfOpenSuccessCloses(t *testing.T) {
	b := NewRedisBreaker(1, 2, 20*time.Millisecond)

	b.RecordFailure()
	assert.Equal(t, CircuitOpen, b.State())

	time.Sleep(25 * time.Millisecond)
	require.True(t, b.Allow())

	b.RecordSuccess()
	assert.Equal(t, CircuitHalfOpen, b.State())

	require.True(t, b.Allow())
	b.RecordSuccess()
	assert.Equal(t, CircuitClosed, b.State())
	assert.True(t, b.Allow())
}

func TestIsNetworkOrSystemError(t *testing.T) {
	assert.False(t, IsNetworkOrSystemError(nil))
	assert.False(t, IsNetworkOrSystemError(redis.Nil))
	assert.False(t, IsNetworkOrSystemError(errors.New("ERR syntax error")))

	assert.True(t, IsNetworkOrSystemError(context.DeadlineExceeded))
	assert.False(t, IsNetworkOrSystemError(context.Canceled))
	assert.True(t, IsNetworkOrSystemError(errors.New("connection refused")))
	assert.True(t, IsNetworkOrSystemError(errors.New("broken pipe")))
	assert.True(t, IsNetworkOrSystemError(errors.New("client is closed")))

	netErr := &net.DNSError{IsTimeout: true}
	assert.True(t, IsNetworkOrSystemError(netErr))
}

func TestRedisBreaker_CanceledDoesNotTrip(t *testing.T) {
	b := NewRedisBreaker(1, 2, 50*time.Millisecond)
	hook := NewRedisCircuitBreakerHook(b)

	next := func(ctx context.Context, cmd redis.Cmder) error {
		return context.Canceled
	}
	processHook := hook.ProcessHook(next)

	cmd := redis.NewStatusCmd(context.Background(), "PING")
	err := processHook(context.Background(), cmd)
	require.ErrorIs(t, err, context.Canceled)
	assert.Equal(t, CircuitClosed, b.State())
}

func TestRedisBreaker_ConcurrentStress(t *testing.T) {
	b := NewRedisBreaker(100, 2, 10*time.Millisecond)

	var wg sync.WaitGroup
	for i := 0; i < 200; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			b.Allow()
			if idx%2 == 0 {
				b.RecordSuccess()
			} else {
				b.RecordFailure()
			}
		}(i)
	}
	wg.Wait()

	state := b.State()
	assert.Contains(t, []CircuitState{CircuitClosed, CircuitOpen}, state)
}

func TestRedisBreaker_FastFailWhenOpen(t *testing.T) {
	const threshold = 5
	b := NewRedisBreaker(threshold, 2, time.Minute)
	hook := NewRedisCircuitBreakerHook(b)

	var redisCalls atomic.Int64
	next := func(ctx context.Context, cmd redis.Cmder) error {
		redisCalls.Add(1)
		return errors.New("connection refused")
	}
	processHook := hook.ProcessHook(next)

	for i := 0; i < threshold; i++ {
		cmd := redis.NewStatusCmd(context.Background(), "PING")
		err := processHook(context.Background(), cmd)
		require.Error(t, err)
	}
	require.Equal(t, CircuitOpen, b.State())
	callsAtOpen := redisCalls.Load()

	for i := 0; i < 100; i++ {
		cmd := redis.NewStatusCmd(context.Background(), "PING")
		err := processHook(context.Background(), cmd)
		require.ErrorIs(t, err, ErrRedisCircuitOpen)
		require.ErrorIs(t, cmd.Err(), ErrRedisCircuitOpen)
	}

	require.Equal(t, callsAtOpen, redisCalls.Load(), "open breaker must fast-fail without hitting redis")
}

func TestRedisBreaker_AdaptiveEWMAThreshold(t *testing.T) {
	// minFailThreshold = 100, failRateRatio = 0.20
	b := NewAdaptiveRedisBreaker(100, 2, 50*time.Millisecond, 0.20)

	// Simulate 1 second of 10,000 successful requests to establish EWMA RPS ~ 10,000
	for i := 0; i < 10000; i++ {
		b.RecordSuccess()
	}

	// Move to next second window
	time.Sleep(1100 * time.Millisecond)

	// Record 1 success to trigger EWMA window calculation
	b.RecordSuccess()
	ewma := b.EWMARPS()
	assert.Greater(t, ewma, 5000.0, "EWMA RPS should reflect past window volume")

	// Dynamic threshold should be ~20% of 10k = 2000 failures.
	// 200 failures under 10k EWMA RPS should NOT trip the breaker.
	for i := 0; i < 200; i++ {
		b.RecordFailure()
	}
	assert.Equal(t, CircuitClosed, b.State(), "200 failures under 10k EWMA RPS must not trip breaker")
}

func TestRedisBreaker_AdaptiveTripsOnSustainedOutage(t *testing.T) {
	b := NewAdaptiveRedisBreaker(50, 2, 50*time.Millisecond, 0.20)

	// Establish low EWMA RPS
	for i := 0; i < 100; i++ {
		b.RecordSuccess()
	}
	time.Sleep(1100 * time.Millisecond)

	// Exceed minimum failure threshold (50 failures)
	for i := 0; i < 60; i++ {
		b.RecordFailure()
	}
	assert.Equal(t, CircuitOpen, b.State(), "sustained failures exceeding dynamic threshold must trip breaker")
}

func BenchmarkRedisBreaker_AdaptiveHotPath(b *testing.B) {
	breaker := NewAdaptiveRedisBreaker(150, 2, time.Minute, 0.20)

	b.Run("Allow", func(b *testing.B) {
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			_ = breaker.Allow()
		}
	})

	b.Run("RecordSuccess", func(b *testing.B) {
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			breaker.RecordSuccess()
		}
	})

	b.Run("RecordFailure", func(b *testing.B) {
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			breaker.RecordFailure()
		}
	})
}
