package entitlements_test

import (
	"crypto/ed25519"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"strings"
	"testing"
	"time"

	"ad-event-processor/internal/licensing/entitlements"
	verify "ad-event-processor/internal/licensing/verify"
)

func BenchmarkEffective(b *testing.B) {
	dep := entitlements.Entitlements{
		Limits:   entitlements.Limits{MaxRPS: 50000, MaxRequestsPerDay: 10_000_000, MaxActiveCampaigns: 500},
		Features: entitlements.FeatureSet{RtbLive: true, MlFraudBoost: true},
	}
	cust := entitlements.Entitlements{
		Limits:   entitlements.Limits{MaxRPS: 10000, MaxRequestsPerDay: 500_000, MaxActiveCampaigns: 50},
		Features: entitlements.FeatureSet{RtbLive: false, MlFraudBoost: false},
	}
	b.ReportAllocs()
	for b.Loop() {
		_ = entitlements.Effective(dep, cust)
	}
}

func BenchmarkVerifyJWT(b *testing.B) {
	pub, priv, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		b.Fatal(err)
	}
	claims := entitlements.LicenseClaims{
		Issuer:       "ad-event-processor-license",
		Subject:      "lic-bench",
		KeyID:        "2026-01",
		DeploymentID: "dep-bench",
		Plan:         "growth",
		ValidFrom:    time.Now().Add(-time.Hour),
		ValidUntil:   time.Now().Add(24 * time.Hour),
		GraceDays:    7,
	}
	claims.Limits.MaxRPS = 1000
	token := benchJWT(b, priv, claims)

	b.ReportAllocs()
	for b.Loop() {
		if _, err := verify.VerifyJWT(token, pub); err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkLicenseSpoolAppend(b *testing.B) {
	dir := b.TempDir()
	cfg := entitlements.DefaultLicenseSpoolConfig()
	cfg.SegmentSizeBytes = 1024 * 1024
	spool, err := entitlements.OpenLicenseSpoolWithConfig(dir, cfg)
	if err != nil {
		b.Fatal(err)
	}
	defer func() { _ = spool.Close() }()

	token := strings.Repeat("x", 400) + ".payload.sig"
	b.ReportAllocs()
	for b.Loop() {
		if err := spool.AppendDurably(token); err != nil {
			b.Fatal(err)
		}
	}
}

func benchJWT(tb testing.TB, priv ed25519.PrivateKey, claims entitlements.LicenseClaims) string {
	tb.Helper()
	header := map[string]string{"alg": "EdDSA", "typ": "JWT", "kid": "2026-01"}
	headerBytes, err := json.Marshal(header)
	if err != nil {
		tb.Fatal(err)
	}
	claimsBytes, err := json.Marshal(claims)
	if err != nil {
		tb.Fatal(err)
	}
	headerB64 := base64.RawURLEncoding.EncodeToString(headerBytes)
	claimsB64 := base64.RawURLEncoding.EncodeToString(claimsBytes)
	signingInput := headerB64 + "." + claimsB64
	sig := ed25519.Sign(priv, []byte(signingInput))
	return signingInput + "." + base64.RawURLEncoding.EncodeToString(sig)
}
