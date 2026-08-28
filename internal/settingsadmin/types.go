package settingsadmin

import (
	"ad-event-processor/internal/campaign"
)

type MutationPreview = campaign.MutationPreviewDTO
type BlacklistEntry = campaign.BlacklistDTO

type BlockIPPreviewChange struct {
	IP          string `json:"ip"`
	Reason      string `json:"reason"`
	OutboxEvent string `json:"outbox_event"`
	Action      string `json:"action"`
	ExpiresAt   string `json:"expires_at,omitempty"`
}

type FraudThreatItem struct {
	Action     string  `json:"action"`
	IP         string  `json:"ip"`
	CampaignID string  `json:"campaign_id"`
	Score      float64 `json:"score"`
	Boost      int32   `json:"boost"`
	TTLSeconds int64   `json:"ttl_seconds"`
}

type settingsOutboxPayload struct {
	Settings map[string]string `json:"settings"`
}

type blacklistOutboxPayload struct {
	Action string `json:"action"`
	IP     string `json:"ip"`
	Reason string `json:"reason"`
}

type fraudThreatOutboxPayload struct {
	Action     string  `json:"action"`
	IP         string  `json:"ip"`
	CampaignID string  `json:"campaign_id"`
	Score      float64 `json:"score"`
	Boost      int32   `json:"boost"`
	TTLSeconds int64   `json:"ttl_seconds"`
}
