package reconciliation

// auditOutboxEventMeta links adjust audit rows to outbox idempotency (recon_adjust_outbox_{id}).
type auditOutboxEventMeta struct {
	OutboxEventID int64 `json:"outbox_event_id"`
}

// pauseCampaignPayload enqueued when HYG30-C ledger invariant sample fails (fail-closed before drift grows).
type pauseCampaignPayload struct {
	CampaignID string `json:"campaign_id"`
}
