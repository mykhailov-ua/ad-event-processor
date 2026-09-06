package pool

import "ad-event-processor/internal/ingest/domainhosts"

type Table = domainhosts.Table

type Snapshot = domainhosts.Snapshot

type SyncRow = domainhosts.SyncRow

type Sync = domainhosts.Sync

var (
	NewTable                    = domainhosts.NewTable
	NewSync                     = domainhosts.NewSync
	BuildSnapshotFromRows       = domainhosts.BuildSnapshotFromRows
	BuildTrackingDomainRotation = domainhosts.BuildTrackingDomainRotation
	SchemeFromHost              = domainhosts.SchemeFromHost
)
