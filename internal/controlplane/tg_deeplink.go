package controlplane

import (
	"crypto/rand"
	"encoding/base64"
)

// GenerateBridgeToken generates a cryptographically secure URL-safe base64 token.
func GenerateBridgeToken() (string, error) {
	b := make([]byte, 32)
	_, err := rand.Read(b)
	if err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(b), nil
}

// ValidateBridgeTokenStr checks if the token matches the required regex pattern of URL-safe characters up to 64 chars.
func ValidateBridgeTokenStr(token string) bool {
	if len(token) == 0 || len(token) > 64 {
		return false
	}
	for i := 0; i < len(token); i++ {
		c := token[i]
		if !((c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z') || (c >= '0' && c <= '9') || c == '_' || c == '-') {
			return false
		}
	}
	return true
}
