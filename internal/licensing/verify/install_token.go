package verify

import (
	"crypto/ed25519"
	"strings"
	"time"

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
	hostFP := HostFingerprint()
	if config.LicenseSeedCouplingEnabled() {
		if err := WriteLicenseMACForToken(path, token, hostFP); err != nil {
			return err
		}
	}
	if config.LicenseClockAnchorEnabled() {
		return UpdateClockAnchor(path, pubKey, hostFP, time.Now().UTC(), 0)
	}
	return nil
}
