package payment

import (
	"crypto/hmac"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/hex"
	"strings"
)

func VerifyBTCPayWebhookSignature(body []byte, sigHeader, secret string) bool {
	secret = strings.TrimSpace(secret)
	sigHeader = strings.TrimSpace(sigHeader)
	if secret == "" || sigHeader == "" {
		return false
	}
	sigHex := sigHeader
	if after, ok := strings.CutPrefix(strings.ToLower(sigHeader), "sha256="); ok {
		sigHex = after
	}
	sigHex = strings.TrimSpace(sigHex)
	if len(sigHex) != sha256.Size*2 {
		return false
	}
	sigBytes, err := hex.DecodeString(sigHex)
	if err != nil || len(sigBytes) != sha256.Size {
		return false
	}
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write(body)
	return subtle.ConstantTimeCompare(sigBytes, mac.Sum(nil)) == 1
}

func SignBTCPayWebhookBody(body []byte, secret string) string {
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write(body)
	return "sha256=" + hex.EncodeToString(mac.Sum(nil))
}
