package ingest

import (
	"context"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestRtbCatalogReload_debounceCoalesces(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	trigger := make(chan struct{}, 1)
	var reloads atomic.Int32

	go runRtbCatalogReloadDebouncer(ctx, trigger, func() {
		reloads.Add(1)
	}, 50*time.Millisecond)

	for range 5 {
		select {
		case trigger <- struct{}{}:
		default:
		}
		time.Sleep(10 * time.Millisecond)
	}

	time.Sleep(120 * time.Millisecond)
	assert.LessOrEqual(t, int(reloads.Load()), 2, "burst pubsub triggers should coalesce to <=2 reloads in 500ms window")
}

func TestRtbCatalogReload_debounceCancelStops(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	trigger := make(chan struct{}, 1)
	var reloads atomic.Int32

	go runRtbCatalogReloadDebouncer(ctx, trigger, func() {
		reloads.Add(1)
	}, 20*time.Millisecond)

	trigger <- struct{}{}
	cancel()
	time.Sleep(50 * time.Millisecond)
	assert.Equal(t, int32(0), reloads.Load(), "cancel before debounce fires should not reload")
}
