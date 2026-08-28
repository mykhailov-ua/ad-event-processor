package controlplane

import (
	"encoding/json"
	"net/http"

	"ad-event-processor/internal/settingsadmin"
)

type MutationPreview = MutationPreviewDTO

type BlockIPWouldChange = settingsadmin.BlockIPPreviewChange

type PauseCampaignWouldChange struct {
	CampaignID  string `json:"campaign_id"`
	Status      string `json:"status,omitempty"`
	Noop        bool   `json:"noop,omitempty"`
	StatusFrom  string `json:"status_from,omitempty"`
	StatusTo    string `json:"status_to,omitempty"`
	OutboxEvent string `json:"outbox_event,omitempty"`
	Reason      string `json:"reason,omitempty"`
}

type ResumeCampaignWouldChange struct {
	CampaignID  string `json:"campaign_id"`
	StatusFrom  string `json:"status_from"`
	StatusTo    string `json:"status_to"`
	OutboxEvent string `json:"outbox_event"`
	Reason      string `json:"reason"`
}

func newMutationPreview(action string, change any) (MutationPreview, error) {
	raw, err := json.Marshal(change)
	if err != nil {
		return MutationPreview{}, err
	}
	return MutationPreview{DryRun: true, Action: action, WouldChange: raw}, nil
}

func ParseDryRun(r *http.Request) bool {
	if r == nil {
		return false
	}
	if r.Header.Get("X-Dry-Run") == "1" {
		return true
	}
	return r.URL.Query().Get("dry_run") == "1"
}
