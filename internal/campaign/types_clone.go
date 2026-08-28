package campaign

import "github.com/google/uuid"

type CloneCampaignSpec struct {
	SourceID       uuid.UUID
	NamePrefix     string
	NameSuffix     string
	IdempotencyKey string
	Options        CloneCampaignOptions
}

type CloneCampaignResult struct {
	ID       string `json:"id"`
	SourceID string `json:"source_id"`
	Name     string `json:"name"`
}
