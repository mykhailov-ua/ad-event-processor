package auth

import (
	"crypto/sha256"
	"encoding/hex"
)

func apiKeyLookup(rawKey string) string {
	sum := sha256.Sum256([]byte(rawKey))
	return hex.EncodeToString(sum[:])
}
