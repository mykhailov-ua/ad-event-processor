package licensing

import (
	"crypto/ed25519"
	"crypto/rand"
	"encoding/json"
	"testing"
	"time"
)

func FuzzVerifyJWT(f *testing.F) {
	pub, priv, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		f.Fatal(err)
	}
	claims := LicenseClaims{
		Issuer:       "ad-event-processor-license",
		Subject:      "fuzz-subject",
		KeyID:        DefaultLicenseKeyID,
		DeploymentID: "dep-fuzz",
		ValidFrom:    time.Now().Add(-time.Hour),
		ValidUntil:   time.Now().Add(24 * time.Hour),
	}
	token, err := SignJWT(claims, priv, DefaultLicenseKeyID)
	if err != nil {
		f.Fatal(err)
	}
	f.Add([]byte(token))

	f.Fuzz(func(t *testing.T, data []byte) {
		if len(data) > maxLicenseTokenBytes {
			return
		}
		_, _ = VerifyJWT(string(data), pub)
	})
}

func FuzzDecodeUnverified(f *testing.F) {
	f.Add([]byte("eyJhbGciOiJFZERTQSIsInR5cCI6IkpXVCJ9.eyJzdWIiOiJ4In0.sig"))
	f.Fuzz(func(t *testing.T, data []byte) {
		if len(data) > maxLicenseTokenBytes {
			return
		}
		_, _ = DecodeUnverified(string(data))
	})
}

func FuzzJSONClaims(f *testing.F) {
	claims := LicenseClaims{Issuer: "ad-event-processor-license", Subject: "x"}
	raw, err := json.Marshal(claims)
	if err != nil {
		f.Fatal(err)
	}
	f.Add(raw)
	f.Fuzz(func(t *testing.T, data []byte) {
		if len(data) > maxLicenseTokenBytes {
			return
		}
		var c LicenseClaims
		_ = json.Unmarshal(data, &c)
	})
}
