//go:build !linux

package entitlements

import "time"

var skewMonoStart = time.Now()

func monotonicRaw() time.Duration {
	return time.Since(skewMonoStart)
}
