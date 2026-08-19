package trialregistry

import "time"

type AnchorType string

const (
	AnchorTelegram     AnchorType = "telegram"
	AnchorDeploymentID AnchorType = "deployment_id"
	AnchorHWID         AnchorType = "hwid"
	AnchorUSDTTx       AnchorType = "usdt_tx"
)

type Status string

const (
	StatusActive    Status = "active"
	StatusExpired   Status = "expired"
	StatusConverted Status = "converted"
	StatusRevoked   Status = "revoked"
)

type AnchorRecord struct {
	AnchorType   AnchorType `json:"anchor_type"`
	AnchorValue  string     `json:"anchor_value"`
	DeploymentID string     `json:"deployment_id"`
	LicenseKey   string     `json:"license_key,omitempty"`
	IssuedAt     time.Time  `json:"issued_at"`
	ValidUntil   time.Time  `json:"valid_until"`
	Status       Status     `json:"status"`
	Notes        string     `json:"notes,omitempty"`
}

type OverrideRecord struct {
	DeploymentID string    `json:"deployment_id"`
	Reason       string    `json:"reason"`
	Operator     string    `json:"operator,omitempty"`
	At           time.Time `json:"at"`
}

type CheckInput struct {
	TelegramID   string
	HWID         string
	USDTTx       string
	DeploymentID string
}

type RecordInput struct {
	TelegramID   string
	HWID         string
	USDTTx       string
	DeploymentID string
	LicenseKey   string
	ValidUntil   time.Time
	Force        bool
	ForceReason  string
	Operator     string
}

type fileSnapshot struct {
	Version   int              `json:"version"`
	Anchors   []AnchorRecord   `json:"anchors"`
	Overrides []OverrideRecord `json:"overrides"`
	Pending   []PendingRequest `json:"pending,omitempty"`
}

type PendingStatus string

const (
	PendingStatusOpen     PendingStatus = "pending"
	PendingStatusApproved PendingStatus = "approved"
	PendingStatusRejected PendingStatus = "rejected"
)

type PendingRequest struct {
	ID               string        `json:"id"`
	TelegramID       string        `json:"telegram_id"`
	TelegramUsername string        `json:"telegram_username,omitempty"`
	DeploymentID     string        `json:"deployment_id,omitempty"`
	RequestedAt      time.Time     `json:"requested_at"`
	Status           PendingStatus `json:"status"`
	Notes            string        `json:"notes,omitempty"`
}

type EnqueuePendingInput struct {
	TelegramID       string
	TelegramUsername string
	Notes            string
}
