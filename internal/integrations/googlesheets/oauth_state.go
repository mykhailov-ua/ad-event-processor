package googlesheets

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
)

const oauthStateTTL = 15 * time.Minute

func issueOAuthState(secret []byte, userID uuid.UUID) (string, error) {
	if len(secret) < 16 {
		return "", fmt.Errorf("oauth state secret too short")
	}
	nonce, err := uuid.NewV7()
	if err != nil {
		return "", err
	}
	exp := time.Now().UTC().Add(oauthStateTTL).Unix()
	payload := fmt.Sprintf("%s|%s|%d", userID.String(), nonce.String(), exp)
	sig := signState(secret, payload)
	raw := payload + "|" + sig
	return base64.RawURLEncoding.EncodeToString([]byte(raw)), nil
}

func verifyOAuthStateReturnUser(secret []byte, state string) (uuid.UUID, error) {
	if len(secret) < 16 {
		return uuid.Nil, ErrInvalidState
	}
	decoded, err := base64.RawURLEncoding.DecodeString(strings.TrimSpace(state))
	if err != nil {
		return uuid.Nil, ErrInvalidState
	}
	parts := strings.Split(string(decoded), "|")
	if len(parts) != 4 {
		return uuid.Nil, ErrInvalidState
	}
	expUnix, err := parseInt64(parts[2])
	if err != nil {
		return uuid.Nil, ErrInvalidState
	}
	if time.Now().UTC().Unix() > expUnix {
		return uuid.Nil, ErrStateExpired
	}
	expected := signState(secret, strings.Join(parts[:3], "|"))
	if !hmac.Equal([]byte(expected), []byte(parts[3])) {
		return uuid.Nil, ErrInvalidState
	}
	userID, err := uuid.Parse(parts[0])
	if err != nil {
		return uuid.Nil, ErrInvalidState
	}
	return userID, nil
}

func verifyOAuthState(secret []byte, state string, userID uuid.UUID) error {
	got, err := verifyOAuthStateReturnUser(secret, state)
	if err != nil {
		return err
	}
	if got != userID {
		return ErrInvalidState
	}
	return nil
}

func signState(secret []byte, payload string) string {
	mac := hmac.New(sha256.New, secret)
	_, _ = mac.Write([]byte(payload))
	return base64.RawURLEncoding.EncodeToString(mac.Sum(nil))
}

func parseInt64(s string) (int64, error) {
	var out int64
	for i := range len(s) {
		c := s[i]
		if c < '0' || c > '9' {
			return 0, fmt.Errorf("invalid int")
		}
		out = out*10 + int64(c-'0')
	}
	return out, nil
}
