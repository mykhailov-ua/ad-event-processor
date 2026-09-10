package postback

import (
	"net/url"
	"sync"
	"time"
)

const (
	circuitFailureThreshold = 5
	circuitOpenDuration     = 10 * time.Minute
	circuitFailureWindow    = 5 * time.Minute
)

type circuitState struct {
	failures    int
	windowStart time.Time
	openUntil   time.Time
}

type hostCircuitBreaker struct {
	mu     sync.Mutex
	states map[string]*circuitState
}

func newHostCircuitBreaker() *hostCircuitBreaker {
	return &hostCircuitBreaker{states: make(map[string]*circuitState)}
}

func (cb *hostCircuitBreaker) hostKey(targetURL string) string {
	u, err := url.Parse(targetURL)
	if err != nil || u.Host == "" {
		return targetURL
	}
	return u.Host
}

func (cb *hostCircuitBreaker) allow(targetURL string) bool {
	if cb == nil {
		return true
	}
	key := cb.hostKey(targetURL)
	now := time.Now().UTC()
	cb.mu.Lock()
	defer cb.mu.Unlock()
	st := cb.states[key]
	if st == nil {
		return true
	}
	if st.openUntil.After(now) {
		return false
	}
	if st.openUntil.Before(now) && !st.openUntil.IsZero() {
		st.openUntil = time.Time{}
		st.failures = 0
	}
	return true
}

func (cb *hostCircuitBreaker) recordSuccess(targetURL string) {
	if cb == nil {
		return
	}
	key := cb.hostKey(targetURL)
	cb.mu.Lock()
	defer cb.mu.Unlock()
	delete(cb.states, key)
}

func (cb *hostCircuitBreaker) recordFailure(targetURL string) bool {
	if cb == nil {
		return false
	}
	key := cb.hostKey(targetURL)
	now := time.Now().UTC()
	cb.mu.Lock()
	defer cb.mu.Unlock()
	st := cb.states[key]
	if st == nil {
		st = &circuitState{windowStart: now}
		cb.states[key] = st
	}
	if now.Sub(st.windowStart) > circuitFailureWindow {
		st.windowStart = now
		st.failures = 0
	}
	st.failures++
	if st.failures >= circuitFailureThreshold {
		st.openUntil = now.Add(circuitOpenDuration)
		recordCircuitOpen()
		return true
	}
	return false
}
