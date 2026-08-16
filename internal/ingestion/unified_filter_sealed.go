package ingestion

import (
	"fmt"
	"os"
	"path/filepath"
	"sync"

	"github.com/bidshard/ad-event-processor/internal/config"
	"github.com/bidshard/ad-event-processor/internal/licensing"
	"github.com/bidshard/ad-event-processor/internal/metrics"
)

const sealedUnifiedFilterAssetLabel = licensing.AssetLabelUnifiedFilter

var (
	unifiedFilterLuaMu     sync.RWMutex
	unifiedFilterLuaActive string
)

// InitUnifiedFilterLua resolves embedded vs sealed unified-filter.lua (cold startup only).
func InitUnifiedFilterLua() error {
	unifiedFilterLuaMu.RLock()
	if unifiedFilterLuaActive != "" {
		unifiedFilterLuaMu.RUnlock()
		return nil
	}
	unifiedFilterLuaMu.RUnlock()

	src, err := resolveUnifiedFilterLuaSource()
	if err != nil {
		return err
	}
	activateUnifiedFilterLuaSource(src)
	return nil
}

func sealedUnifiedFilterBlobPath() string {
	if v := os.Getenv("AD_EVENT_PROCESSOR_UNIFIED_FILTER_SEALED_BLOB"); v != "" {
		return v
	}
	return filepath.Join("internal", "ingestion", "unified_filter_sealed.bin")
}

func sealedUnifiedFilterBlob() ([]byte, error) {
	path := sealedUnifiedFilterBlobPath()
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	if len(data) == 0 {
		return nil, os.ErrNotExist
	}
	return data, nil
}

func resolveUnifiedFilterLuaSource() (string, error) {
	if config.LicenseAssetsUnsealed() {
		return unifiedFilterLua, nil
	}
	sealed, err := sealedUnifiedFilterBlob()
	if err != nil {
		if os.IsNotExist(err) {
			return unifiedFilterLua, nil
		}
		return "", err
	}
	mck, err := licensing.DeriveMCKFromLicenseFile(
		config.LicensePathFromEnv(),
		nil,
		licensing.HostFingerprint(),
	)
	if err != nil {
		metrics.LicenseLuaSealFailTotal.Inc()
		return "", fmt.Errorf("sealed lua mck: %w", err)
	}
	plain, err := licensing.OpenAsset(sealedUnifiedFilterAssetLabel, sealed, mck)
	if err != nil {
		metrics.LicenseLuaSealFailTotal.Inc()
		return "", fmt.Errorf("sealed lua open: %w", err)
	}
	if len(plain) == 0 {
		metrics.LicenseLuaSealFailTotal.Inc()
		return "", licensing.ErrSealFormat
	}
	return string(plain), nil
}

func activateUnifiedFilterLuaSource(src string) {
	unifiedFilterLuaMu.Lock()
	unifiedFilterLuaActive = src
	unifiedFilterLuaAny = src
	unifiedFilterLuaMu.Unlock()
}

func unifiedFilterLuaForScript() string {
	unifiedFilterLuaMu.RLock()
	defer unifiedFilterLuaMu.RUnlock()
	if unifiedFilterLuaActive != "" {
		return unifiedFilterLuaActive
	}
	return unifiedFilterLua
}
