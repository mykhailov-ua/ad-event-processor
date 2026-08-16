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
	if secret == "" || sigHeader == "" {
		return false
	}
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write(body)
	expected := hex.EncodeToString(mac.Sum(nil))
	return subtle.ConstantTimeCompare([]byte(strings.TrimSpace(sigHeader)), []byte(expected)) == 1
}
