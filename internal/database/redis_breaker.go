package database

import (
	"context"
	"errors"
	"math"
	"net"
	"strings"
	"sync/atomic"
	"time"

	redis "github.com/redis/go-redis/v9"
)

var ErrRedisCircuitOpen = errors.New("redis circuit breaker is open")

type CircuitState int32

const (
	CircuitClosed   CircuitState = 0
	CircuitOpen     CircuitState = 1
	CircuitHalfOpen CircuitState = 2
)

func (s CircuitState) String() string {
	switch s {
	case CircuitClosed:
		return "closed"
	case CircuitOpen:
		return "open"
	case CircuitHalfOpen:
		return "half-open"
	default:
		return "unknown"
	}
}

// RedisBreaker: process-wide open/half-open gate on redis hook; distinct from stream processor CB.
type RedisBreaker struct {
	state            int32
	failures         int64
	successes        int64
	lastOpenedUnix   int64
	failThreshold    int64
	successThreshold int64
	openTimeout      time.Duration

	failRateRatio   float64
	totalReqs       uint64
	failedReqs      uint64
	windowStartUnix int64
	ewmaRPSBits     uint64
}

func NewRedisBreaker(failThreshold, successThreshold int64, openTimeout time.Duration) *RedisBreaker {
	return NewAdaptiveRedisBreaker(failThreshold, successThreshold, openTimeout, 0.20)
}

func NewAdaptiveRedisBreaker(failThreshold, successThreshold int64, openTimeout time.Duration, failRateRatio float64) *RedisBreaker {
	if failThreshold <= 0 {
		failThreshold = 150
	}
	if failRateRatio <= 0 {
		failRateRatio = 0.20
	}
	return &RedisBreaker{
		state:            int32(CircuitClosed),
		failThreshold:    failThreshold,
		successThreshold: successThreshold,
		openTimeout:      openTimeout,
		failRateRatio:    failRateRatio,
		windowStartUnix:  time.Now().Unix(),
	}
}

func (b *RedisBreaker) State() CircuitState {
	return CircuitState(atomic.LoadInt32(&b.state))
}

func (b *RedisBreaker) EWMARPS() float64 {
	_ = b.updateEWMAAndGetThreshold()
	return math.Float64frombits(atomic.LoadUint64(&b.ewmaRPSBits))
}

func (b *RedisBreaker) updateEWMAAndGetThreshold() int64 {
	nowSec := time.Now().Unix()
	winSec := atomic.LoadInt64(&b.windowStartUnix)
	if nowSec > winSec {
		if atomic.CompareAndSwapInt64(&b.windowStartUnix, winSec, nowSec) {
			total := atomic.SwapUint64(&b.totalReqs, 0)
			_ = atomic.SwapUint64(&b.failedReqs, 0)

			oldEWMA := math.Float64frombits(atomic.LoadUint64(&b.ewmaRPSBits))
			var newEWMA float64
			if oldEWMA == 0 {
				newEWMA = float64(total)
			} else {
				newEWMA = 0.2*float64(total) + 0.8*oldEWMA
			}
			atomic.StoreUint64(&b.ewmaRPSBits, math.Float64bits(newEWMA))
		}
	}

	ewmaRPS := math.Float64frombits(atomic.LoadUint64(&b.ewmaRPSBits))
	dynamicThreshold := int64(ewmaRPS * b.failRateRatio)
	if dynamicThreshold < b.failThreshold {
		dynamicThreshold = b.failThreshold
	}
	return dynamicThreshold
}

func (b *RedisBreaker) Allow() bool {
	state := atomic.LoadInt32(&b.state)
	if state == int32(CircuitClosed) {
		return true
	}

	if state == int32(CircuitHalfOpen) {
		return true
	}

	if state == int32(CircuitOpen) {
		lastOpened := atomic.LoadInt64(&b.lastOpenedUnix)
		if time.Since(time.Unix(0, lastOpened)) >= b.openTimeout {
			if atomic.CompareAndSwapInt32(&b.state, int32(CircuitOpen), int32(CircuitHalfOpen)) {
				atomic.StoreInt64(&b.successes, 0)
				atomic.StoreInt64(&b.failures, 0)
				return true
			}
		}
		return false
	}

	return false
}

func (b *RedisBreaker) RecordSuccess() {
	atomic.AddUint64(&b.totalReqs, 1)
	_ = b.updateEWMAAndGetThreshold()
	state := atomic.LoadInt32(&b.state)
	if state == int32(CircuitHalfOpen) {
		successes := atomic.AddInt64(&b.successes, 1)
		if successes >= b.successThreshold {
			if atomic.CompareAndSwapInt32(&b.state, int32(CircuitHalfOpen), int32(CircuitClosed)) {
				atomic.StoreInt64(&b.failures, 0)
			}
		}
	} else if state == int32(CircuitClosed) {
		atomic.StoreInt64(&b.failures, 0)
	}
}

func (b *RedisBreaker) RecordFailure() {
	atomic.AddUint64(&b.totalReqs, 1)
	atomic.AddUint64(&b.failedReqs, 1)

	state := atomic.LoadInt32(&b.state)
	if state == int32(CircuitHalfOpen) {
		b.trip()
	} else if state == int32(CircuitClosed) {
		threshold := b.updateEWMAAndGetThreshold()
		failures := atomic.AddInt64(&b.failures, 1)
		if failures >= threshold {
			b.trip()
		}
	}
}

func (b *RedisBreaker) trip() {
	for {
		state := atomic.LoadInt32(&b.state)
		if state == int32(CircuitOpen) {
			return
		}
		if atomic.CompareAndSwapInt32(&b.state, state, int32(CircuitOpen)) {
			atomic.StoreInt64(&b.lastOpenedUnix, time.Now().UnixNano())
			return
		}
	}
}

func IsNetworkOrSystemError(err error) bool {
	if err == nil {
		return false
	}
	var opErr *net.OpError
	if errors.As(err, &opErr) {
		return true
	}
	var netErr net.Error
	if errors.As(err, &netErr) {
		return true
	}
	if errors.Is(err, context.DeadlineExceeded) {
		return true
	}
	if errors.Is(err, redis.Nil) {
		return false
	}
	if errors.Is(err, context.Canceled) {
		return false
	}
	errStr := err.Error()
	if strings.Contains(errStr, "connection refused") ||
		strings.Contains(errStr, "EOF") ||
		strings.Contains(errStr, "broken pipe") ||
		strings.Contains(errStr, "client is closed") ||
		strings.Contains(errStr, "use of closed network connection") {
		return true
	}
	return false
}

type RedisCircuitBreakerHook struct {
	breaker *RedisBreaker
}

func NewRedisCircuitBreakerHook(breaker *RedisBreaker) *RedisCircuitBreakerHook {
	return &RedisCircuitBreakerHook{breaker: breaker}
}

func (h *RedisCircuitBreakerHook) DialHook(next redis.DialHook) redis.DialHook {
	return next
}

func (h *RedisCircuitBreakerHook) ProcessHook(next redis.ProcessHook) redis.ProcessHook {
	return func(ctx context.Context, cmd redis.Cmder) error {
		if !h.breaker.Allow() {
			cmd.SetErr(ErrRedisCircuitOpen)
			return ErrRedisCircuitOpen
		}

		err := next(ctx, cmd)
		if err != nil {
			if IsNetworkOrSystemError(err) {
				h.breaker.RecordFailure()
			} else {
				h.breaker.RecordSuccess()
			}
		} else {
			h.breaker.RecordSuccess()
		}
		return err
	}
}

func (h *RedisCircuitBreakerHook) ProcessPipelineHook(next redis.ProcessPipelineHook) redis.ProcessPipelineHook {
	return func(ctx context.Context, cmds []redis.Cmder) error {
		if !h.breaker.Allow() {
			for _, cmd := range cmds {
				cmd.SetErr(ErrRedisCircuitOpen)
			}
			return ErrRedisCircuitOpen
		}

		err := next(ctx, cmds)
		if err != nil {
			if IsNetworkOrSystemError(err) {
				h.breaker.RecordFailure()
			} else {
				h.breaker.RecordSuccess()
			}
		} else {
			h.breaker.RecordSuccess()
		}
		return err
	}
}
