package verify

import (
	"crypto/ed25519"
	"strings"

	"ad-event-processor/internal/config"
)

func InstallToken(path, token string, pubKey ed25519.PublicKey) error {
	token = strings.TrimSpace(token)
	if token == "" {
		return ErrInvalidTokenFormat
	}
	if len(pubKey) > 0 {
		if _, err := VerifyJWT(token, pubKey); err != nil {
			return err
		}
	} else if _, err := VerifyJWTResolved(token); err != nil {
		return err
	}
	if err := WriteFileAtomic(path, []byte(token), 0o600); err != nil {
		return err
	}
	if config.LicenseSeedCouplingEnabled() {
		return WriteLicenseMACForToken(path, token, HostFingerprint())
	}
	return nil
}
