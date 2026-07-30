package licensing

import "time"

type LicenseClaims struct {
	Issuer       string     `json:"iss"`
	Subject      string     `json:"sub"`
	KeyID        string     `json:"kid"`
	DeploymentID string     `json:"deployment_id"`
	CustomerName string     `json:"customer_name"`
	Plan         string     `json:"plan"`
	SKU          string     `json:"sku,omitempty"`
	VolumeBand   VolumeBand `json:"volume_band"`
	ValidFrom    time.Time  `json:"valid_from"`
	ValidUntil   time.Time  `json:"valid_until"`
	GraceDays    int        `json:"grace_days"`
	Limits       Limits     `json:"limits"`
	Features     FeatureSet `json:"features"`
	Bind         struct {
		Mode        string `json:"mode"`
		Fingerprint string `json:"fingerprint"`
	} `json:"bind"`
	SupportTier string `json:"support_tier"`
}

type LicenseState string

const (
	StateActive  LicenseState = "ACTIVE"
	StateGrace   LicenseState = "GRACE"
	StateExpired LicenseState = "EXPIRED"
	StateRevoked LicenseState = "REVOKED"
)
