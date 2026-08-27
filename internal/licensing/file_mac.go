package licensing

import (
	"crypto/ed25519"
	"crypto/hmac"
	"crypto/sha256"
	"crypto/subtle"
	"errors"
	"os"
	"strings"
)

var ErrLicenseMACMismatch = errors.New("license file mac mismatch")

const licenseMACSize = sha256.Size

func LicenseMACPath(licensePath string) string {
	return licensePath + ".mac"
}

func ComputeLicenseMAC(mckWork [32]byte, fileBytes []byte) [licenseMACSize]byte {
	mac := hmac.New(sha256.New, mckWork[:])
	_, _ = mac.Write(fileBytes)
	var out [licenseMACSize]byte
	copy(out[:], mac.Sum(nil))
	return out
}

func VerifyLicenseMAC(mckWork [32]byte, fileBytes, mac []byte) bool {
	want := ComputeLicenseMAC(mckWork, fileBytes)
	if len(mac) != licenseMACSize {
		return false
	}
	return subtle.ConstantTimeCompare(mac, want[:]) == 1
}

func WriteLicenseMAC(licensePath string, mckWork [32]byte, fileBytes []byte) error {
	mac := ComputeLicenseMAC(mckWork, fileBytes)
	return WriteFileAtomic(LicenseMACPath(licensePath), mac[:], 0o600)
}

func WriteLicenseMACForToken(licensePath, token string, hostFingerprint string) error {
	token = strings.TrimSpace(token)
	if token == "" {
		return ErrInvalidTokenFormat
	}
	mckWork, err := DeriveMCKWorkForRecheck(token, hostFingerprint)
	if err != nil {
		return err
	}
	return WriteLicenseMAC(licensePath, mckWork, []byte(token))
}

func verifyOrBootstrapLicenseMAC(licensePath string, pubKey ed25519.PublicKey, hostFingerprint string) error {
	fileBytes, err := readFileTrim(licensePath)
	if err != nil {
		return err
	}
	mckWork, err := DeriveMCKWorkForRecheckFromLicenseFile(licensePath, pubKey, hostFingerprint)
	if err != nil {
		return err
	}
	macPath := LicenseMACPath(licensePath)
	stored, readErr := os.ReadFile(macPath)
	if readErr != nil {
		if os.IsNotExist(readErr) {
			return WriteLicenseMAC(licensePath, mckWork, fileBytes)
		}
		return readErr
	}
	if !VerifyLicenseMAC(mckWork, fileBytes, stored) {
		return ErrLicenseMACMismatch
	}
	return nil
}
