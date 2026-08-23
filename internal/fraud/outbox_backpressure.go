package fraud

// OutboxEnforcementEventTypes are safety-critical outbox rows drained on the
// fast lane. They are excluded from ivt-detector backpressure pending counts
// so fraud enforcement is not starved by campaign pacing storms.
var OutboxEnforcementEventTypes = []string{
	"UPDATE_BLACKLIST",
	"ML_BLACKLIST_ADD",
	"ML_SCORE_BOOST",
	"ML_GHOST_IVT",
}

const outboxBackpressurePendingSQL = `
SELECT COUNT(*)::bigint FROM outbox_events
WHERE status = 'PENDING'
AND NOT (event_type = ANY($1::text[]))`
