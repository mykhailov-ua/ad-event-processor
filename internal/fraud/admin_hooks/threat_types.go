package admin_hooks

type FraudThreatEnqueueItem struct {
	Action     string  `json:"action"`
	IP         string  `json:"ip"`
	CampaignID string  `json:"campaign_id"`
	Score      float64 `json:"score"`
	Boost      int32   `json:"boost"`
	TTLSeconds int64   `json:"ttl_seconds"`
}

type fraudThreatBatchRequest struct {
	Items []FraudThreatEnqueueItem `json:"items"`
}
