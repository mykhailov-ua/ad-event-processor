package provider

import (
	"time"

	"github.com/google/uuid"
)

type LineType string

const (
	LineTypeSpend   LineType = "spend"
	LineTypeRevenue LineType = "revenue"
)

type CostLine struct {
	CustomerID   uuid.UUID
	CampaignID   uuid.UUID
	Date         time.Time
	Network      string
	PlacementID  string
	AdsetID      string
	AdID         string
	LineType     LineType
	AmountMicro  int64
	Currency     string
	SnapshotHour time.Time
}

type Credential struct {
	CustomerID          uuid.UUID
	Network             string
	AccountID           string
	AccessToken         string
	RefreshToken        string
	APIKey              string
	ExtraConfig         map[string]string
	ExpiresAt           time.Time
	SyncIntervalMinutes int
	TokenMapping        TokenMapping
}
