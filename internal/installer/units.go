package installer

const legacyStackSlug = "es" + "px"

const (
	TrackerSystemdUnitName      = "ad-event-processor-tracker.service"
	RollbackSystemdUnitPrefix   = "ad-event-processor-rollback@"
	LegacyTrackerSystemdUnit    = legacyStackSlug + "-tracker.service"
	LegacyRollbackSystemdPrefix = legacyStackSlug + "-rollback@"
)
