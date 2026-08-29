//go:build linux

package ingest

import (
	"fmt"
	"syscall"
	"time"

	"ad-event-processor/internal/filter"
)

func shiftSystemClock(d time.Duration) (restore func(), err error) {
	var tv syscall.Timeval
	if err := syscall.Gettimeofday(&tv); err != nil {
		return nil, fmt.Errorf("gettimeofday: %w", err)
	}
	orig := syscall.Timeval{Sec: tv.Sec, Usec: tv.Usec}

	delta := d / time.Second
	deltaUsec := int64((d % time.Second) / time.Microsecond)
	tv.Sec += int64(delta)
	tv.Usec += deltaUsec
	if tv.Usec >= 1_000_000 {
		tv.Sec += tv.Usec / 1_000_000
		tv.Usec %= 1_000_000
	}

	if err := syscall.Settimeofday(&tv); err != nil {
		return nil, fmt.Errorf("settimeofday: %w", err)
	}
	filter.RefreshCachedWallClockNow()

	return func() {
		_ = syscall.Settimeofday(&orig)
		filter.RefreshCachedWallClockNow()
	}, nil
}
