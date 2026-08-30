package log

import (
	"sync/atomic"
	"time"
)

// testSyncDelay injects sleep before activeSeg.Sync in syncLocked (fault/gate tests only).
var testSyncDelayNanos atomic.Int64

func SetSyncDelayForTest(d time.Duration) {
	testSyncDelayNanos.Store(int64(d))
}

func testSyncDelay() time.Duration {
	return time.Duration(testSyncDelayNanos.Load())
}
