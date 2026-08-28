package shardadmin

import "errors"

var (
	ErrSlotMigrationNotReady       = errors.New("slot migration copy not complete for all MIGRATING slots")
	ErrSlotMigrationNoDraft        = errors.New("no draft slot map version with MIGRATING slots")
	ErrSlotMigrationKeysMissing    = errors.New("slot migration target shard missing required keys")
	ErrSlotMigrationLagNotCaughtUp = errors.New("slot migration dual-write lag above epsilon")
)
