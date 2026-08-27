package controlplane

import (
	_ "embed"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync/atomic"

	"ad-event-processor/internal/config"
	"ad-event-processor/internal/licensing"
	"ad-event-processor/internal/metrics"
)

//go:embed control_runtime_policy.json
var controlRuntimePolicyEmbed []byte

const sealedControlRuntimeAssetLabel = licensing.AssetLabelControlRuntime

var controlRuntimePolicyActive atomic.Pointer[ControlRuntimePolicy]

// ControlRuntimePolicy is the cold-path control plane policy opened from a sealed blob.
type ControlRuntimePolicy struct {
	Version                int  `json:"version"`
	LicenseRecheckRequired bool `json:"license_recheck_required"`
	MaxLicenseApplyPerHour int  `json:"max_license_apply_per_hour"`
}

// InitControlRuntimePolicy loads the control runtime policy (sealed or dev embed).
func InitControlRuntimePolicy() error {
	raw, err := resolveControlRuntimePolicyBytes()
	if err != nil {
		return err
	}
	var policy ControlRuntimePolicy
	if err := json.Unmarshal(raw, &policy); err != nil {
		return fmt.Errorf("control runtime policy json: %w", err)
	}
	if policy.Version < 1 {
		return fmt.Errorf("control runtime policy version: %d", policy.Version)
	}
	controlRuntimePolicyActive.Store(&policy)
	return nil
}

// ControlRuntimePolicyLoaded returns the active policy when InitControlRuntimePolicy succeeded.
func ControlRuntimePolicyLoaded() (ControlRuntimePolicy, bool) {
	p := controlRuntimePolicyActive.Load()
	if p == nil {
		return ControlRuntimePolicy{}, false
	}
	return *p, true
}

// ControlLicenseRecheckRequired reports whether sealed control policy requires license recheck.
func ControlLicenseRecheckRequired() bool {
	policy, ok := ControlRuntimePolicyLoaded()
	if !ok {
		return config.LicenseRequiredFromEnv()
	}
	return policy.LicenseRecheckRequired
}

func sealedControlRuntimeBlobPath() string {
	if v := os.Getenv("AD_EVENT_PROCESSOR_CONTROL_RUNTIME_SEALED_BLOB"); v != "" {
		return v
	}
	return filepath.Join("internal", "controlplane", "control_runtime_sealed.bin")
}

func sealedControlRuntimeBlob() ([]byte, error) {
	path := sealedControlRuntimeBlobPath()
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	if len(data) == 0 {
		return nil, os.ErrNotExist
	}
	return data, nil
}

func resolveControlRuntimePolicyBytes() ([]byte, error) {
	if config.LicenseAssetsUnsealed() {
		return controlRuntimePolicyEmbed, nil
	}
	sealed, err := sealedControlRuntimeBlob()
	if err != nil {
		if os.IsNotExist(err) {
			return controlRuntimePolicyEmbed, nil
		}
		return nil, err
	}
	mck, err := licensing.DeriveMCKFromLicenseFile(
		config.LicensePathFromEnv(),
		nil,
		licensing.HostFingerprint(),
	)
	if err != nil {
		metrics.ControlRuntimeSealFailTotal.Inc()
		return nil, fmt.Errorf("sealed control runtime mck: %w", err)
	}
	plain, err := licensing.OpenAsset(sealedControlRuntimeAssetLabel, sealed, mck)
	if err != nil {
		metrics.ControlRuntimeSealFailTotal.Inc()
		return nil, fmt.Errorf("sealed control runtime open: %w", err)
	}
	if len(plain) == 0 {
		metrics.ControlRuntimeSealFailTotal.Inc()
		return nil, licensing.ErrSealFormat
	}
	return plain, nil
}
