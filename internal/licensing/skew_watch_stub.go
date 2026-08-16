//go:build !linux

package licensing

import "time"

var skewMonoStart = time.Now()

func monotonicRaw() time.Duration {
	return time.Since(skewMonoStart)
}
