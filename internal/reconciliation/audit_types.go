package reconciliation

type auditOutboxEventMeta struct {
	OutboxEventID int64 `json:"outbox_event_id"`
}

type pauseCampaignPayload struct {
	CampaignID string `json:"campaign_id"`
}
