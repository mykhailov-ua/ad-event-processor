package log

import (
	"fmt"
	"time"
)

type DurabilityMode int

const (
	DurabilityAsync DurabilityMode = iota
	DurabilityGroupCommit
	DurabilitySync
)

type DurabilityConfig struct {
	Mode               DurabilityMode
	FlushInterval      time.Duration
	GroupCommitRecords int64
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
