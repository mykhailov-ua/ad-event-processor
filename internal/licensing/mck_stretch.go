package licensing

import (
	"crypto/ed25519"
	"strings"

	"golang.org/x/crypto/argon2"
)

func StretchMCKForRecheck(mck [32]byte, deploymentID string) ([32]byte, error) {
	var zero [32]byte
	deploymentID = strings.TrimSpace(deploymentID)
	if deploymentID == "" {
		return zero, ErrMCKDerivation
	}
	sum := argon2.IDKey(mck[:], []byte(deploymentID), hwidArgonTime, hwidArgonMemory, hwidArgonThreads, hwidArgonKeyLen)
	var out [32]byte
	copy(out[:], sum)
	return out, nil
}

func DeriveMCKWorkForRecheck(token, hwid string) ([32]byte, error) {
	mck, err := DeriveMCK(token, hwid)
	if err != nil {
		return mck, err
	}
	claims, err := DecodeUnverified(token)
	if err != nil || claims == nil {
		var zero [32]byte
		return zero, ErrMCKDerivation
	}
	return StretchMCKForRecheck(mck, claims.DeploymentID)
}

func DeriveMCKWorkForRecheckFromLicenseFile(path string, pubKey ed25519.PublicKey, hostFingerprint string) ([32]byte, error) {
	var zero [32]byte
	mck, err := DeriveMCKFromLicenseFile(path, pubKey, hostFingerprint)
	if err != nil {
		return zero, err
	}
	data, err := readFileTrim(path)
	if err != nil {
		return zero, err
	}
	claims, err := DecodeUnverified(string(data))
	if err != nil || claims == nil {
		return zero, ErrMCKDerivation
	}
	return StretchMCKForRecheck(mck, claims.DeploymentID)
}

func FeatureSeedFromLicenseFileRecheck(path string, pubKey ed25519.PublicKey, hostFingerprint string) (uint32, error) {
	mckWork, err := DeriveMCKWorkForRecheckFromLicenseFile(path, pubKey, hostFingerprint)
	if err != nil {
		return 0, err
	}
	return FeatureSeedFromMCK(mckWork), nil
}
