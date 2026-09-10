package inbound

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"net"
	"net/http"
	"os"
	"strings"
)

var ErrAuthFailed = errors.New("POSTBACK_AUTH_FAILED")

type AuthConfig struct {
	IPAllowlist []string
	Secret      []byte
}

func AuthDisabled() bool {
	return strings.TrimSpace(os.Getenv("POSTBACK_INBOUND_AUTH_DISABLED")) == "1"
}

func Configured(cfg AuthConfig) bool {
	return len(cfg.IPAllowlist) > 0 || len(cfg.Secret) > 0
}

func VerifyHTTP(cfg AuthConfig, r *http.Request, body []byte) error {
	if AuthDisabled() || !Configured(cfg) {
		return nil
	}
	if len(cfg.IPAllowlist) > 0 {
		if err := verifyIP(cfg.IPAllowlist, clientIP(r)); err != nil {
			return err
		}
	}
	if len(cfg.Secret) > 0 {
		if err := verifySignature(cfg.Secret, r, body); err != nil {
			return err
		}
	}
	return nil
}

func clientIP(r *http.Request) string {
	if r == nil {
		return ""
	}
	if xri := strings.TrimSpace(r.Header.Get("X-Real-IP")); xri != "" {
		return xri
	}
	if xff := strings.TrimSpace(r.Header.Get("X-Forwarded-For")); xff != "" {
		parts := strings.Split(xff, ",")
		return strings.TrimSpace(parts[0])
	}
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		return strings.TrimSpace(r.RemoteAddr)
	}
	return host
}

func verifyIP(allowlist []string, ip string) error {
	parsed := net.ParseIP(strings.TrimSpace(ip))
	if parsed == nil {
		return ErrAuthFailed
	}
	for _, raw := range allowlist {
		raw = strings.TrimSpace(raw)
		if raw == "" {
			continue
		}
		if strings.Contains(raw, "/") {
			_, network, err := net.ParseCIDR(raw)
			if err == nil && network.Contains(parsed) {
				return nil
			}
			continue
		}
		if parsed.Equal(net.ParseIP(raw)) {
			return nil
		}
	}
	return ErrAuthFailed
}

func verifySignature(secret []byte, r *http.Request, body []byte) error {
	sigHeader := strings.TrimSpace(r.Header.Get("X-Postback-Signature"))
	if sigHeader == "" {
		sigHeader = strings.TrimSpace(r.URL.Query().Get("sig"))
	}
	if sigHeader == "" {
		return ErrAuthFailed
	}
	sigHeader = strings.TrimPrefix(strings.ToLower(sigHeader), "sha256=")
	bodyHash := sha256.Sum256(body)
	canonical := fmt.Sprintf("%s|%s|%s|%s",
		strings.ToUpper(r.Method),
		r.URL.Path,
		sortedQuery(r.URL.Query()),
		hex.EncodeToString(bodyHash[:]),
	)
	mac := hmac.New(sha256.New, secret)
	mac.Write([]byte(canonical))
	expected := hex.EncodeToString(mac.Sum(nil))
	if !hmac.Equal([]byte(strings.ToLower(sigHeader)), []byte(expected)) {
		return ErrAuthFailed
	}
	return nil
}

func sortedQuery(values map[string][]string) string {
	if len(values) == 0 {
		return ""
	}
	keys := make([]string, 0, len(values))
	for k := range values {
		if k == "sig" {
			continue
		}
		keys = append(keys, k)
	}
	sortStrings(keys)
	parts := make([]string, 0, len(keys))
	for _, k := range keys {
		for _, v := range values[k] {
			parts = append(parts, k+"="+v)
		}
	}
	sortStrings(parts)
	return strings.Join(parts, "&")
}

func sortStrings(items []string) {
	for i := 1; i < len(items); i++ {
		for j := i; j > 0 && items[j] < items[j-1]; j-- {
			items[j], items[j-1] = items[j-1], items[j]
		}
	}
}
