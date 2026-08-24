package landerhost

import (
	"crypto/hmac"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/base64"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"
)

const previewTokenTTL = time.Hour

func MintPreviewToken(secret []byte, landerID uuid.UUID, version int, now time.Time) (string, error) {
	if len(secret) == 0 {
		return "", fmt.Errorf("preview secret is required")
	}
	if landerID == uuid.Nil || version <= 0 {
		return "", fmt.Errorf("invalid preview scope")
	}
	exp := now.UTC().Add(previewTokenTTL).Unix()
	payload := landerID.String() + "|" + strconv.Itoa(version) + "|" + strconv.FormatInt(exp, 10)
	sig := signPreviewPayload(secret, payload)
	return base64.RawURLEncoding.EncodeToString([]byte(payload)) + "." + base64.RawURLEncoding.EncodeToString(sig), nil
}

func VerifyPreviewToken(secret []byte, token string, landerID uuid.UUID, now time.Time) (version int, ok bool) {
	if len(secret) == 0 || token == "" || landerID == uuid.Nil {
		return 0, false
	}
	parts := strings.Split(token, ".")
	if len(parts) != 2 {
		return 0, false
	}
	payloadBytes, err := base64.RawURLEncoding.DecodeString(parts[0])
	if err != nil {
		return 0, false
	}
	sigBytes, err := base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil {
		return 0, false
	}
	payload := string(payloadBytes)
	want := signPreviewPayload(secret, payload)
	if len(sigBytes) != len(want) || subtle.ConstantTimeCompare(sigBytes, want) != 1 {
		return 0, false
	}
	seg := strings.Split(payload, "|")
	if len(seg) != 3 {
		return 0, false
	}
	if seg[0] != landerID.String() {
		return 0, false
	}
	ver, err := strconv.Atoi(seg[1])
	if err != nil || ver <= 0 {
		return 0, false
	}
	exp, err := strconv.ParseInt(seg[2], 10, 64)
	if err != nil || now.UTC().Unix() > exp {
		return 0, false
	}
	return ver, true
}

func PreviewURL(base string, landerID uuid.UUID, token string) string {
	base = strings.TrimRight(strings.TrimSpace(base), "/")
	if base == "" {
		return ""
	}
	return base + "/lp-preview/" + landerID.String() + "/?token=" + token
}

func signPreviewPayload(secret []byte, payload string) []byte {
	mac := hmac.New(sha256.New, secret)
	_, _ = mac.Write([]byte(payload))
	return mac.Sum(nil)
}
