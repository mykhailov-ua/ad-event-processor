package log

import (
	"sync/atomic"
	"time"
)

var testSyncDelayNanos atomic.Int64

func SetSyncDelayForTest(d time.Duration) {
	testSyncDelayNanos.Store(int64(d))
}

func testSyncDelay() time.Duration {
	return time.Duration(testSyncDelayNanos.Load())
}
