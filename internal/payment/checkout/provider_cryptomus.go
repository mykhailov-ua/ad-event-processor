package checkout

import (
	"crypto/md5"
	"crypto/subtle"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"strings"
)

func VerifyCryptomusWebhookSignature(body []byte, sigHeader, apiKey string) bool {
	apiKey = strings.TrimSpace(apiKey)
	sigHeader = strings.TrimSpace(sigHeader)
	if apiKey == "" || sigHeader == "" {
		return false
	}
	expected, ok := cryptomusSignFromBody(body, apiKey)
	if !ok {
		return false
	}
	sigBytes, err := hex.DecodeString(strings.ToLower(sigHeader))
	if err != nil {
		return false
	}
	expBytes, err := hex.DecodeString(expected)
	if err != nil || len(sigBytes) != len(expBytes) {
		return false
	}
	return subtle.ConstantTimeCompare(sigBytes, expBytes) == 1
}

func cryptomusSignFromBody(body []byte, apiKey string) (string, bool) {
	var payload map[string]json.RawMessage
	if err := json.Unmarshal(body, &payload); err != nil {
		return "", false
	}
	delete(payload, "sign")
	encoded, err := json.Marshal(payload)
	if err != nil {
		return "", false
	}
	sum := md5.Sum(append(encoded, []byte(apiKey)...))
	return hex.EncodeToString(sum[:]), true
}

func SignCryptomusWebhookFields(fields map[string]any, apiKey string) ([]byte, string, error) {
	unsigned, err := json.Marshal(fields)
	if err != nil {
		return nil, "", err
	}
	sign, ok := cryptomusSignFromBody(unsigned, apiKey)
	if !ok {
		return nil, "", fmt.Errorf("cryptomus sign payload")
	}
	fields["sign"] = sign
	signed, err := json.Marshal(fields)
	if err != nil {
		return nil, "", err
	}
	return signed, sign, nil
}
