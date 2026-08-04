package licensing

import (
	"crypto/ed25519"
	"strings"
)

// InstallToken verifies and atomically writes a license JWT to path (0600).
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
	return WriteFileAtomic(path, []byte(token), 0o600)
}
