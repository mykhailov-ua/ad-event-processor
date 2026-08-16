package payment

import (
	"crypto/md5"
	"encoding/hex"
	"encoding/json"
	"strings"
)

func VerifyCryptomusWebhookSignature(body []byte, sigHeader, apiKey string) bool {
	apiKey = strings.TrimSpace(apiKey)
	sigHeader = strings.TrimSpace(sigHeader)
	if apiKey == "" || sigHeader == "" {
		return false
	}
	var payload map[string]json.RawMessage
	if err := json.Unmarshal(body, &payload); err != nil {
		return false
	}
	delete(payload, "sign")
	encoded, err := json.Marshal(payload)
	if err != nil {
		return false
	}
	sum := md5.Sum(append(encoded, []byte(apiKey)...))
	expected := hex.EncodeToString(sum[:])
	return strings.EqualFold(expected, sigHeader)
}
