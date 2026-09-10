package signing

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"net/url"
	"strings"
)

// SignGETURL returns HMAC-SHA256 hex for a rendered GET URL without sig query param.
func SignGETURL(secret []byte, renderedURL string) string {
	if len(secret) == 0 || strings.TrimSpace(renderedURL) == "" {
		return ""
	}
	canonical := canonicalURLWithoutSig(renderedURL)
	mac := hmac.New(sha256.New, secret)
	mac.Write([]byte(canonical))
	return hex.EncodeToString(mac.Sum(nil))
}

func canonicalURLWithoutSig(raw string) string {
	u, err := url.Parse(raw)
	if err != nil {
		return raw
	}
	q := u.Query()
	q.Del("sig")
	u.RawQuery = q.Encode()
	return u.String()
}
