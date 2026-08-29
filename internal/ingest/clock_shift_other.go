//go:build !linux

package ingest

import (
	"fmt"
	"runtime"
	"time"

	"ad-event-processor/internal/filter"
)

func shiftSystemClock(d time.Duration) (restore func(), err error) {
	return nil, fmt.Errorf("shiftSystemClock unsupported on %s", runtime.GOOS)
}

func refreshCachedWallClockNow() {
	filter.RefreshCachedWallClockNow()
}
