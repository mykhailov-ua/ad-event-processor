//go:build linux

package licensing

import (
	"time"

	"golang.org/x/sys/unix"
)

func monotonicRaw() time.Duration {
	var ts unix.Timespec
	if err := unix.ClockGettime(unix.CLOCK_MONOTONIC_RAW, &ts); err != nil {
		return 0
	}
	return time.Duration(ts.Sec)*time.Second + time.Duration(ts.Nsec)
}
