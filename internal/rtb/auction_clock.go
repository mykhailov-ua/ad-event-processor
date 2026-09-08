package rtb

import (
	"sync/atomic"
	"time"
)

var rtbWallClockUnix atomic.Int64

func auctionNowUnix() int64 {
	v := rtbWallClockUnix.Load()
	if v > 0 {
		return v
	}
	return time.Now().UTC().Unix()
}

// SnapshotWallClockUnix mirrors ingest cached unix for zero-alloc RTB shadow paths.
func SnapshotWallClockUnix(unix int64) {
	if unix > 0 {
		rtbWallClockUnix.Store(unix)
	}
}
