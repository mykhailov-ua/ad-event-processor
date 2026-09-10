package licensing

import (
	"fmt"
	"os"
	"path/filepath"

	"ad-event-processor/internal/config"
)

// SealedAssetRole names a license-sealed blob checked at process startup before traffic.
type SealedAssetRole uint8

const (
	SealedAssetRoleTracker   SealedAssetRole = 1
	SealedAssetRoleProcessor SealedAssetRole = 2
	SealedAssetRoleControl   SealedAssetRole = 3
	SealedAssetRoleEdge      SealedAssetRole = 4
)

func sealedAssetRoleName(role SealedAssetRole) string {
	switch role {
	case SealedAssetRoleTracker:
		return "tracker"
	case SealedAssetRoleProcessor:
		return "processor"
	case SealedAssetRoleControl:
		return "control"
	case SealedAssetRoleEdge:
		return "edge"
	default:
		return fmt.Sprintf("unknown(%d)", role)
	}
}

// VerifySealedAssetsReady opens the role sealed blob when asset sealing is active.
// Dev/unsealed mode returns immediately. When license is neither required nor present
// (config.LicenseProbeEnabled false), verification is skipped to match doctor probes.
func VerifySealedAssetsReady(role SealedAssetRole) error {
	if config.LicenseAssetsUnsealed() {
		return nil
	}
	if !config.LicenseProbeEnabled() {
		return nil
	}
	if !config.LicenseFilePresent() {
		if config.LicenseRequiredFromEnv() {
			return fmt.Errorf("sealed assets %s: license file required: %w", sealedAssetRoleName(role), os.ErrNotExist)
		}
		return nil
	}

	switch role {
	case SealedAssetRoleTracker:
		return verifySealedAssetBlob(role, sealedUnifiedFilterBlobPath(), AssetLabelUnifiedFilter, true)
	case SealedAssetRoleProcessor:
		return verifySealedAssetBlob(role, sealedProcessorCHIngestBlobPath(), AssetLabelProcessorClickHouseIngest, false)
	case SealedAssetRoleControl:
		return verifySealedAssetBlob(role, sealedControlRuntimeBlobPath(), AssetLabelControlRuntime, false)
	case SealedAssetRoleEdge:
		return verifySealedAssetBlob(role, sealedEdgeBlobPath(), AssetLabelEdge, false)
	default:
		return fmt.Errorf("sealed assets: unknown role %d", role)
	}
}

func verifySealedAssetBlob(role SealedAssetRole, path, label string, required bool) error {
	sealed, err := readSealedBlobFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			if required {
				return fmt.Errorf("sealed assets %s: blob: %w", sealedAssetRoleName(role), err)
			}
			return nil
		}
		return fmt.Errorf("sealed assets %s: blob read: %w", sealedAssetRoleName(role), err)
	}
	mck, err := DeriveMCKFromLicenseFile(
		config.LicensePathFromEnv(),
		nil,
		HostFingerprint(),
	)
	if err != nil {
		return fmt.Errorf("sealed assets %s: mck: %w", sealedAssetRoleName(role), err)
	}
	plain, err := OpenAsset(label, sealed, mck)
	if err != nil {
		return fmt.Errorf("sealed assets %s: open: %w", sealedAssetRoleName(role), err)
	}
	if len(plain) == 0 {
		return fmt.Errorf("sealed assets %s: %w", sealedAssetRoleName(role), ErrSealFormat)
	}
	return nil
}

func readSealedBlobFile(path string) ([]byte, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	if len(data) == 0 {
		return nil, os.ErrNotExist
	}
	return data, nil
}

func sealedUnifiedFilterBlobPath() string {
	if v := os.Getenv("AD_EVENT_PROCESSOR_UNIFIED_FILTER_SEALED_BLOB"); v != "" {
		return v
	}
	return filepath.Join("internal", "ingestion", "unified_filter_sealed.bin")
}

func sealedProcessorCHIngestBlobPath() string {
	if v := os.Getenv("AD_EVENT_PROCESSOR_PROCESSOR_CH_INGEST_SEALED_BLOB"); v != "" {
		return v
	}
	return filepath.Join("internal", "ingestion", "processor_ch_ingest_sealed.bin")
}

func sealedControlRuntimeBlobPath() string {
	if v := os.Getenv("AD_EVENT_PROCESSOR_CONTROL_RUNTIME_SEALED_BLOB"); v != "" {
		return v
	}
	return filepath.Join("internal", "control", "control_runtime_sealed.bin")
}

func sealedEdgeBlobPath() string {
	if v := os.Getenv("AD_EVENT_PROCESSOR_EDGE_SEALED_BLOB"); v != "" {
		return v
	}
	return filepath.Join("internal", "edge", "edge_sealed.bin")
}
