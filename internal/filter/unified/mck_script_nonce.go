package unified

import (
	"encoding/hex"
	"fmt"

	"ad-event-processor/internal/config"
	"ad-event-processor/internal/licensing"
)

func applyRedisScriptNoncePrefix(src string) (string, error) {
	if !config.LicenseSeedCouplingEnabled() {
		return src, nil
	}
	mckWork, err := licensing.DeriveMCKWorkForRecheckFromLicenseFile(
		config.LicensePathFromEnv(),
		nil,
		licensing.HostFingerprint(),
	)
	if err != nil {
		return "", fmt.Errorf("redis script nonce mck: %w", err)
	}
	nonce := licensing.RedisScriptNonceFromMCKWork(mckWork)
	return formatLuaWithRedisScriptNonce(src, nonce), nil
}

func formatLuaWithRedisScriptNonce(src string, nonce [8]byte) string {
	return "-- rsnonce:" + hex.EncodeToString(nonce[:]) + "\n" + src
}
