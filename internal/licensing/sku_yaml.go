package licensing

import (
	"fmt"
	"os"
	"strings"
	"time"

	"gopkg.in/yaml.v3"
)

const DefaultSKUCode = "license"

type SKUFile struct {
	SKUs []SKUDefinition `yaml:"skus"`
}

type SKUDefinition struct {
	Code                 string     `yaml:"code"`
	DisplayName          string     `yaml:"display_name"`
	PriceUSDMonthly      float64    `yaml:"price_usd_monthly"`
	ValidDays            int        `yaml:"valid_days"`
	GraceDays            int        `yaml:"grace_days"`
	OfflineGraceDays     int        `yaml:"offline_grace_days"`
	HeartbeatIntervalHrs int        `yaml:"heartbeat_interval_hours"`
	PreRenewalWarnDays   int        `yaml:"pre_renewal_warn_days"`
	VolumeBand           VolumeBand `yaml:"volume_band"`
	Features             FeatureSet `yaml:"features"`
	Limits               Limits     `yaml:"limits"`
	Bind                 struct {
		Mode string `yaml:"mode"`
	} `yaml:"bind"`
	SupportTier string `yaml:"support_tier"`
}

func LoadSKUFile(path string) (*SKUFile, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var doc SKUFile
	if err := yaml.Unmarshal(data, &doc); err != nil {
		return nil, err
	}
	if len(doc.SKUs) == 0 {
		return nil, fmt.Errorf("sku file %s: no skus defined", path)
	}
	seen := make(map[string]struct{}, len(doc.SKUs))
	for i := range doc.SKUs {
		code := strings.TrimSpace(doc.SKUs[i].Code)
		if code == "" {
			return nil, fmt.Errorf("sku[%d]: code is required", i)
		}
		if _, ok := seen[code]; ok {
			return nil, fmt.Errorf("duplicate sku code %q", code)
		}
		seen[code] = struct{}{}
		doc.SKUs[i].Code = code
		if doc.SKUs[i].ValidDays <= 0 {
			doc.SKUs[i].ValidDays = 30
		}
		if doc.SKUs[i].GraceDays <= 0 {
			doc.SKUs[i].GraceDays = 7
		}
	}
	return &doc, nil
}

func (f *SKUFile) GetSKU(code string) (*SKUDefinition, error) {
	code = strings.TrimSpace(code)
	if code == "" {
		code = DefaultSKUCode
	}
	for i := range f.SKUs {
		if f.SKUs[i].Code == code {
			return &f.SKUs[i], nil
		}
	}
	return nil, fmt.Errorf("sku %q not found", code)
}

func (f *SKUFile) DefaultSKU() (*SKUDefinition, error) {
	return f.GetSKU(DefaultSKUCode)
}

type IssueLicenseInput struct {
	SKUCode      string
	CustomerName string
	DeploymentID string
	LicenseID    string
	Fingerprint  string
	HWIDHash     string
	ValidFrom    time.Time
}

func (s SKUDefinition) BuildClaims(in IssueLicenseInput) LicenseClaims {
	validFrom := in.ValidFrom
	if validFrom.IsZero() {
		validFrom = time.Now().UTC()
	}
	claims := LicenseClaims{
		Issuer:       "ad-event-processor-license",
		Subject:      in.LicenseID,
		DeploymentID: in.DeploymentID,
		CustomerName: in.CustomerName,
		Plan:         s.Code,
		SKU:          s.Code,
		VolumeBand:   s.VolumeBand,
		ValidFrom:    validFrom,
		ValidUntil:   validFrom.Add(time.Duration(s.ValidDays) * 24 * time.Hour),
		GraceDays:    s.GraceDays,
		Limits:       s.Limits,
		Features:     SanitizeFeaturesForSKU(s.Code, s.Features).Normalized(),
		SupportTier:  s.SupportTier,
	}
	claims.Bind.Mode = s.Bind.Mode
	if claims.Bind.Mode == "" {
		claims.Bind.Mode = "soft"
	}
	claims.Bind.Fingerprint = in.Fingerprint
	claims.HWIDHash = strings.TrimSpace(in.HWIDHash)
	return claims
}
