package log

import (
	"fmt"
	"time"
)

type DurabilityMode int

// DurabilityAsync: mmap append only; background flush loop calls Sync on FlushInterval.
// DurabilityGroupCommit: fsync after GroupCommitRecords or gate interval; DurabilitySync: fsync every leader append.
const (
	DurabilityAsync DurabilityMode = iota
	DurabilityGroupCommit
	DurabilitySync
)

type DurabilityConfig struct {
	Mode               DurabilityMode // selects fsync cadence in PartitionLog.append/flush loop
	FlushInterval      time.Duration  // async/group ticker when gate absent or interval due
	GroupCommitRecords int64          // batch size before syncLocked when DiskWriteGate nil
}

func DefaultDurabilityConfig() DurabilityConfig {
	return DurabilityConfig{
		Mode:               DurabilityAsync,
		FlushInterval:      100 * time.Millisecond,
		GroupCommitRecords: 64,
	}
}

func ParseDurabilityMode(s string) (DurabilityMode, error) {
	switch s {
	case "", "async":
		return DurabilityAsync, nil
	case "group", "group_commit":
		return DurabilityGroupCommit, nil
	case "sync":
		return DurabilitySync, nil
	default:
		return DurabilityAsync, fmt.Errorf("unknown durability mode %q (want async|group|sync)", s)
	}
}
