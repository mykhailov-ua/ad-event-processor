package control

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

var controlRuntimePolicyActive atomic.Pointer[RuntimePolicy]

type RuntimePolicy struct {
	Version                int  `json:"version"`
	LicenseRecheckRequired bool `json:"license_recheck_required"`
	MaxLicenseApplyPerHour int  `json:"max_license_apply_per_hour"`
}

func InitRuntimePolicy() error {
	raw, err := resolveRuntimePolicyBytes()
	if err != nil {
		return err
	}
	var policy RuntimePolicy
	if err := json.Unmarshal(raw, &policy); err != nil {
		return fmt.Errorf("control runtime policy json: %w", err)
	}
	if policy.Version < 1 {
		return fmt.Errorf("control runtime policy version: %d", policy.Version)
	}
	controlRuntimePolicyActive.Store(&policy)
	return nil
}

func RuntimePolicyLoaded() (RuntimePolicy, bool) {
	p := controlRuntimePolicyActive.Load()
	if p == nil {
		return RuntimePolicy{}, false
	}
	return *p, true
}

func LicenseRecheckRequired() bool {
	policy, ok := RuntimePolicyLoaded()
	if !ok {
		return config.LicenseRequiredFromEnv()
	}
	return policy.LicenseRecheckRequired
}

func sealedRuntimeBlobPath() string {
	if v := os.Getenv("AD_EVENT_PROCESSOR_CONTROL_RUNTIME_SEALED_BLOB"); v != "" {
		return v
	}
	return filepath.Join("internal", "control", "control_runtime_sealed.bin")
}

func sealedRuntimeBlob() ([]byte, error) {
	path := sealedRuntimeBlobPath()
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	if len(data) == 0 {
		return nil, os.ErrNotExist
	}
	return data, nil
}

func resolveRuntimePolicyBytes() ([]byte, error) {
	if config.LicenseAssetsUnsealed() {
		return controlRuntimePolicyEmbed, nil
	}
	sealed, err := sealedRuntimeBlob()
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
