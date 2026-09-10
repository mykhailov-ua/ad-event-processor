package postback

import (
	"testing"
	"time"
)

func TestHostCircuitBreaker_opensAfterThreshold(t *testing.T) {
	cb := newHostCircuitBreaker()
	url := "https://partner.example/postback"
	for i := 0; i < circuitFailureThreshold; i++ {
		cb.recordFailure(url)
	}
	if cb.allow(url) {
		t.Fatal("holdout: circuit must open after consecutive failures")
	}
}

func TestHostCircuitBreaker_resetsAfterSuccess(t *testing.T) {
	cb := newHostCircuitBreaker()
	url := "https://partner.example/reset"
	for i := 0; i < circuitFailureThreshold-1; i++ {
		cb.recordFailure(url)
	}
	cb.recordSuccess(url)
	if !cb.allow(url) {
		t.Fatal("success should clear failure count")
	}
	_ = time.Now()
}
