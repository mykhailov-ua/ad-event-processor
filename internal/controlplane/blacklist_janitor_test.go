package controlplane

import (
	"testing"
	"time"
)

func TestBlacklistJanitor_DefaultInterval(t *testing.T) {
	j := NewBlacklistJanitor(nil, 0)
	if j == nil {
		t.Fatal("expected janitor")
	}
	if j.Interval() != time.Minute {
		t.Fatalf("interval: got %v want 1m", j.Interval())
	}
}
