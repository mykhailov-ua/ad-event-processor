package controlplane

import (
	"testing"
	"time"

	"ad-event-processor/internal/fraudadmin"
)

func TestBlacklistJanitor_DefaultInterval(t *testing.T) {
	j := fraudadmin.NewBlacklistJanitor(nil, 0)
	if j == nil {
		t.Fatal("expected janitor")
	}
	if j.Interval() != time.Minute {
		t.Fatalf("interval: got %v want 1m", j.Interval())
	}
}
