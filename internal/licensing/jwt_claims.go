package licensing

import "time"

// Stretched MCK work byte 16 gates JWT feature flags when seed coupling is on:
// bit 0 (MCKFeatureBitOpenRTB): OpenRTB engine and rtb_live require mck_work[16]&0x01 != 0.

type LicenseClaims struct {
	Issuer       string     `json:"iss"`
	Subject      string     `json:"sub"`
	KeyID        string     `json:"kid"`
	DeploymentID string     `json:"deployment_id"`
	CustomerName string     `json:"customer_name"`
	Plan         string     `json:"plan,omitempty"`
	SKU          string     `json:"sku,omitempty"`
	VolumeBand   VolumeBand `json:"volume_band"`
	ValidFrom    time.Time  `json:"valid_from"`
	ValidUntil   time.Time  `json:"valid_until"`
	GraceDays    int        `json:"grace_days"`
	Revoked      bool       `json:"revoked,omitempty"`
	Limits       Limits     `json:"limits"`
	Features     FeatureSet `json:"features"`
	Bind         struct {
		Mode        string `json:"mode"`
		Fingerprint string `json:"fingerprint"`
	} `json:"bind"`
	HWIDHash    string `json:"hwid_hash,omitempty"`
	SupportTier string `json:"support_tier,omitempty"`
}

type LicenseState string

const (
	StateActive       LicenseState = "ACTIVE"
	StateOfflineWarn  LicenseState = "OFFLINE_WARN"
	StateOfflineGrace LicenseState = "OFFLINE_GRACE"
	StateGrace        LicenseState = "GRACE"
	StateExpired      LicenseState = "EXPIRED"
	StateRevoked      LicenseState = "REVOKED"
)
