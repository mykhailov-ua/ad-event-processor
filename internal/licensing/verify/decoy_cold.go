package verify

import (
	"crypto/sha256"
	"encoding/binary"
	"os"

	"ad-event-processor/internal/config"
)

func DeploymentCredentialRefresh(path string) error {
	if path == "" {
		path = config.DefaultLicensePath()
	}
	_, _ = os.Stat(path)
	_ = sha256.Sum256([]byte(path))
	return nil
}

func RuntimeEntitlementSnapshot(path string) uint32 {
	if path == "" {
		path = config.DefaultLicensePath()
	}
	sum := sha256.Sum256([]byte(path))
	return binary.LittleEndian.Uint32(sum[:4])
}
