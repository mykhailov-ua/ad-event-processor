package automation

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"
)

const (
	GroupByPlacement = "placement_id"
	GroupByCampaign  = "campaign"

	ActionNotify             = "notify"
	ActionPauseCampaign      = "pause_campaign"
	ActionBlacklistPlacement = "blacklist_placement"
	ActionPlatformPause      = "platform_pause"
)

type Action struct {
	Type       string `json:"type"`
	WebhookURL string `json:"webhook_url,omitempty"`
	Network    string `json:"network,omitempty"`
}

type Rule struct {
	ID              uuid.UUID
	CustomerID      uuid.UUID
	CampaignID      uuid.UUID
	HasCampaign     bool
	Name            string
	Metric          string
	Operator        string
	Threshold       float64
	WindowMinutes   int
	GroupBy         string
	Actions         []Action
	CooldownMinutes int
	Enabled         bool
	LastFiredAt     time.Time
	HasLastFired    bool
}

func ParseActions(raw []byte) ([]Action, error) {
	if len(raw) == 0 {
		return nil, nil
	}
	var actions []Action
	if err := json.Unmarshal(raw, &actions); err != nil {
		return nil, err
	}
	return actions, nil
}

func MarshalActions(actions []Action) ([]byte, error) {
	if len(actions) == 0 {
		return []byte("[]"), nil
	}
	return json.Marshal(actions)
}

type Match struct {
	RuleID        uuid.UUID
	CustomerID    uuid.UUID
	CampaignID    uuid.UUID
	PlacementID   string
	Metric        string
	Operator      string
	Threshold     float64
	ObservedValue float64
	WindowStart   time.Time
	WindowEnd     time.Time
	Actions       []Action
}

type WouldFire struct {
	RuleID        string   `json:"rule_id"`
	CampaignID    string   `json:"campaign_id"`
	PlacementID   string   `json:"placement_id,omitempty"`
	Metric        string   `json:"metric"`
	Operator      string   `json:"operator"`
	Threshold     float64  `json:"threshold"`
	ObservedValue float64  `json:"observed_value"`
	Actions       []Action `json:"actions"`
}
