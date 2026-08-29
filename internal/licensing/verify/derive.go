package verify

import (
	"crypto/ed25519"
	"crypto/sha256"
	"encoding/base64"
	"encoding/binary"
	"errors"
	"io"
	"strings"

	"golang.org/x/crypto/hkdf"
)

var ErrMCKDerivation = errors.New("mck derivation failed")

func DeriveMCK(token, hwid string) ([32]byte, error) {
	var zero [32]byte
	token = strings.TrimSpace(token)
	if token == "" {
		return zero, ErrMCKDerivation
	}
	parts := strings.Split(token, ".")
	if len(parts) != 3 {
		return zero, ErrMCKDerivation
	}
	kid, err := JWTKeyID(token)
	if err != nil || kid == "" {
		return zero, ErrMCKDerivation
	}
	sig, err := base64.RawURLEncoding.DecodeString(parts[2])
	if err != nil || len(sig) == 0 {
		return zero, ErrMCKDerivation
	}
	payload, err := base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil || len(payload) == 0 {
		return zero, ErrMCKDerivation
	}
	claims, err := DecodeUnverified(token)
	if err != nil || claims == nil {
		return zero, ErrMCKDerivation
	}
	deploymentID := strings.TrimSpace(claims.DeploymentID)
	if deploymentID == "" {
		return zero, ErrMCKDerivation
	}
	ikm := make([]byte, 0, len(sig)+len(payload)+len(hwid)+len(kid))
	ikm = append(ikm, sig...)
	ikm = append(ikm, payload...)
	ikm = append(ikm, []byte(hwid)...)
	ikm = append(ikm, []byte(kid)...)
	return deriveMCKBytes(ikm, deploymentID)
}

func DeriveMCKVerified(token string, pubKey ed25519.PublicKey, hostFingerprint string) ([32]byte, error) {
	var zero [32]byte
	claims, err := VerifyJWT(token, pubKey)
	if err != nil {
		return zero, err
	}
	if err := VerifyDeploymentBind(claims, hostFingerprint); err != nil {
		return zero, err
	}
	return DeriveMCK(token, HostHWID())
}

func DeriveMCKFromLicenseFile(path string, pubKey ed25519.PublicKey, hostFingerprint string) ([32]byte, error) {
	var zero [32]byte
	data, err := readFileTrim(path)
	if err != nil {
		return zero, err
	}
	token := string(data)
	if len(pubKey) > 0 {
		if _, err := VerifyJWT(token, pubKey); err != nil {
			return zero, err
		}
	} else if _, err := VerifyJWTResolved(token); err != nil {
		return zero, err
	}
	claims, err := DecodeUnverified(token)
	if err != nil || claims == nil {
		return zero, ErrMCKDerivation
	}
	if err := VerifyDeploymentBind(claims, hostFingerprint); err != nil {
		return zero, err
	}
	hwid := HostHWID()
	return DeriveMCK(token, hwid)
}

func deriveMCKBytes(ikm []byte, deploymentID string) ([32]byte, error) {
	var out [32]byte
	reader := hkdf.New(sha256.New, ikm, []byte(deploymentID), []byte(MCKInfoLabel()))
	if _, err := io.ReadFull(reader, out[:]); err != nil {
		return out, ErrMCKDerivation
	}
	return out, nil
}

func FeatureSeedFromMCK(mck [32]byte) uint32 {
	lo := binary.BigEndian.Uint32(mck[0:4])
	hi := binary.BigEndian.Uint32(mck[4:8])
	return lo ^ hi
}
