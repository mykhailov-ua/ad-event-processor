package fraud

var OutboxEnforcementEventTypes = []string{
	"UPDATE_BLACKLIST",
	"ML_BLACKLIST_ADD",
	"ML_SCORE_BOOST",
	"ML_SILENT_REJECT",
	"ML_GHOST_IVT",
}

const outboxBackpressurePendingSQL = `
SELECT COUNT(*)::bigint FROM outbox_events
WHERE status = 'PENDING'
AND NOT (event_type = ANY($1::text[]))`
