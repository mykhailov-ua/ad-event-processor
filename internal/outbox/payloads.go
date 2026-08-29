package outbox

import (
	"time"

	"github.com/google/uuid"
)

type CampaignPayload struct {
	CampaignID  string `json:"campaign_id"`
	BudgetLimit int64  `json:"budget_limit,omitempty"`
}

type SettingsPayload struct {
	Settings map[string]string `json:"settings"`
}

type BlacklistPayload struct {
	Action string `json:"action"`
	IP     string `json:"ip"`
	Reason string `json:"reason"`
}

type FraudThreatPayload struct {
	Action     string  `json:"action"`
	IP         string  `json:"ip"`
	CampaignID string  `json:"campaign_id"`
	Score      float64 `json:"score"`
	Boost      int32   `json:"boost"`
	TTLSeconds int64   `json:"ttl_seconds"`
}

type FraudModelVersionPayload struct {
	ModelVersion string `json:"model_version"`
	Hash         string `json:"hash"`
	ShardID      int    `json:"shard_id"`
}

type CampaignIDPayload struct {
	CampaignID string `json:"campaign_id"`
}

type BrandIDPayload struct {
	BrandID string `json:"brand_id"`
}

type BrandFcapPayload struct {
	BrandID    string `json:"brand_id"`
	FreqLimit  int32  `json:"freq_limit"`
	FreqWindow int32  `json:"freq_window"`
}

type CampaignSchedulePayload struct {
	CampaignID   string     `json:"campaign_id"`
	StartAt      *time.Time `json:"start_at,omitempty"`
	EndAt        *time.Time `json:"end_at,omitempty"`
	DaypartHours []int16    `json:"daypart_hours,omitempty"`
}

type CampaignPacingPayload struct {
	CampaignID string `json:"campaign_id"`
	PacingMode string `json:"pacing_mode"`
}

type UserConsentPayload struct {
	UserIDHash string `json:"user_id_hash"`
	Purposes   int16  `json:"purposes"`
}

type PurgeUserDataPayload struct {
	ErasureID     string `json:"erasure_id"`
	UserIDHash    string `json:"user_id_hash"`
	SubjectUserID string `json:"subject_user_id"`
}

type PausePlacementPayload struct {
	CampaignID  string `json:"campaign_id"`
	PlacementID string `json:"placement_id"`
	Action      string `json:"action,omitempty"`
}

type CohortSnapshotPayload struct {
	Version int64 `json:"version"`
}

type CTVGtaxSettlementPayload struct {
	SettlementID string `json:"settlement_id"`
	CustomerID   string `json:"customer_id"`
	CampaignID   string `json:"campaign_id"`
	SpendMicro   int64  `json:"spend_micro"`
}

type TelegramEventPayload struct {
	CampaignID uuid.UUID `json:"campaign_id"`
	BotID      int64     `json:"bot_id"`
	Payload    []byte    `json:"payload"`
}

type LanderPublishedPayload struct {
	LanderID string `json:"lander_id"`
}

type RtbCatalogReloadPayload struct {
	Trigger string `json:"trigger"`
}
