package campaign

type auditReasonChange struct {
	Reason string `json:"reason"`
}

type AuditIdempotencyMeta struct {
	IdempotencyKey string `json:"idempotency_key"`
}

type auditIdempotencyMeta = AuditIdempotencyMeta

const CampaignExportVersion = 1
