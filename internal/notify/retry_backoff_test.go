package notify

import (
	"testing"
)

func TestBackoffDuration_firstRetry(t *testing.T) {
	got := backoffDuration(1)
	if got != retryBackoffBase {
		t.Fatalf("expected %v, got %v", retryBackoffBase, got)
	}
}

func TestBackoffDuration_exponential(t *testing.T) {
	got := backoffDuration(3)
	want := retryBackoffBase * 4
	if got != want {
		t.Fatalf("expected %v, got %v", want, got)
	}
}

func TestBackoffDuration_zeroRetries(t *testing.T) {
	if got := backoffDuration(0); got != 0 {
		t.Fatalf("expected 0, got %v", got)
	}
}
