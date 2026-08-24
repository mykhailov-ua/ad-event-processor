package installer

import "ad-event-processor/internal/edge"

const legacyStackSlug = "es" + "px"

const (
	TrackerSystemdUnitName      = "ad-event-processor-tracker.service"
	EdgeXDPSystemdUnitName      = "ad-event-processor-edge-xdp.service"
	EdgeBPFSyncSystemdUnitName  = "ad-event-processor-edge-bpf-sync.service"
	RollbackSystemdUnitPrefix   = "ad-event-processor-rollback@"
	LegacyTrackerSystemdUnit    = legacyStackSlug + "-tracker.service"
	LegacyRollbackSystemdPrefix = legacyStackSlug + "-rollback@"
)

const edgeBPFPinDir = edge.DefaultBPFPinDir
