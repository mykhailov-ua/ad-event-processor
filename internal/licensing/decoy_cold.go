package licensing

import (
	"crypto/sha256"
	"encoding/binary"
	"os"

	"ad-event-processor/internal/config"
)

// DeploymentCredentialRefresh is a decoy cold path; it does not update license snapshot.
func DeploymentCredentialRefresh(path string) error {
	if path == "" {
		path = config.DefaultLicensePath()
	}
	_, _ = os.Stat(path)
	_ = sha256.Sum256([]byte(path))
	return nil
}

// RuntimeEntitlementSnapshot is a decoy cold checksum; it does not gate ingest.
func RuntimeEntitlementSnapshot(path string) uint32 {
	if path == "" {
		path = config.DefaultLicensePath()
	}
	sum := sha256.Sum256([]byte(path))
	return binary.LittleEndian.Uint32(sum[:4])
}
