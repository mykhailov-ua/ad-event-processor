package licensing

import (
	"crypto/sha256"
	"encoding/binary"
	"strings"
	"time"
)

const licenseFileRecheckMaxJitter = 120 * time.Second

func LicenseFileRecheckIntervalJittered(base time.Duration, deploymentID string) time.Duration {
	if base <= 0 {
		base = fileLicenseRecheckInterval()
	}
	deploymentID = strings.TrimSpace(deploymentID)
	if deploymentID == "" {
		return base
	}
	sum := sha256.Sum256([]byte(deploymentID))
	jitterSec := binary.LittleEndian.Uint32(sum[:4]) % uint32(licenseFileRecheckMaxJitter/time.Second)
	return base + time.Duration(jitterSec)*time.Second
}

func DeploymentIDFromLicensePath(path string) string {
	data, err := readFileTrim(path)
	if err != nil {
		return ""
	}
	claims, err := DecodeUnverified(string(data))
	if err != nil || claims == nil {
		return ""
	}
	return strings.TrimSpace(claims.DeploymentID)
}
