package flow

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"
)

type LanderDTO struct {
	ID            uuid.UUID  `json:"id"`
	Name          string     `json:"name"`
	URL           string     `json:"url,omitempty"`
	HostedAssetID *uuid.UUID `json:"hosted_asset_id,omitempty"`
	HostedURL     string     `json:"hosted_url,omitempty"`
	CreatedAt     time.Time  `json:"created_at"`
}

type OfferDTO struct {
	ID        uuid.UUID `json:"id"`
	Name      string    `json:"name"`
	URL       string    `json:"url"`
	CreatedAt time.Time `json:"created_at"`
}

type CreateLanderRequest struct {
	Name string `json:"name"`
	URL  string `json:"url"`
}

type CreateOfferRequest struct {
	Name string `json:"name"`
	URL  string `json:"url"`
}

type CreateFlowRequest struct {
	Name  string    `json:"name"`
	Paths []PathDTO `json:"paths"`
}

type UpdateFlowRequest struct {
	Name  string    `json:"name"`
	Paths []PathDTO `json:"paths"`
}

type DTO struct {
	ID        uuid.UUID       `json:"id"`
	Name      string          `json:"name"`
	Paths     json.RawMessage `json:"paths"`
	CreatedAt time.Time       `json:"created_at"`
}

type PathLanderRef struct {
	LanderID uuid.UUID `json:"lander_id"`
	Weight   int32     `json:"weight"`
}

type PathOfferRef struct {
	OfferID  uuid.UUID `json:"offer_id"`
	Weight   int32     `json:"weight"`
	CapDaily *int32    `json:"cap_daily,omitempty"`
	CapTotal *int32    `json:"cap_total,omitempty"`
}

type PathFiltersDTO struct {
	Countries []string `json:"countries,omitempty"`
	Devices   []string `json:"devices,omitempty"`
	OS        []string `json:"os,omitempty"`
	Languages []string `json:"languages,omitempty"`
}

type PathDTO struct {
	Weight  int32           `json:"weight"`
	Landers []PathLanderRef `json:"landers"`
	Offers  []PathOfferRef  `json:"offers"`
	Filters *PathFiltersDTO `json:"filters,omitempty"`
}

type PathErrorDTO struct {
	PathIndex int    `json:"path_index"`
	Code      string `json:"code"`
	Message   string `json:"message"`
}

type ValidateResponseDTO struct {
	Valid              bool           `json:"valid"`
	PathErrors         []PathErrorDTO `json:"path_errors,omitempty"`
	SuggestedFixAction string         `json:"suggested_fix_action,omitempty"`
}

type HostedEditorFileDTO struct {
	Path     string `json:"path"`
	Size     int64  `json:"size"`
	Editable bool   `json:"editable"`
}

type HostedEditorStateDTO struct {
	LanderID            uuid.UUID             `json:"lander_id"`
	Name                string                `json:"name"`
	DraftVersion        int                   `json:"draft_version"`
	PublishedVersion    int                   `json:"published_version"`
	HasUnpublishedDraft bool                  `json:"has_unpublished_draft"`
	Files               []HostedEditorFileDTO `json:"files"`
	PreviewURL          string                `json:"preview_url,omitempty"`
}

type HostedEditorFileBodyDTO struct {
	Content string `json:"content"`
}

type HostedEditorSaveResultDTO struct {
	DraftVersion        int  `json:"draft_version"`
	HasUnpublishedDraft bool `json:"has_unpublished_draft"`
}

type HostedEditorPublishRequest struct {
	Version int `json:"version"`
}
