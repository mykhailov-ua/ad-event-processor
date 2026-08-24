package moderatorintel_test

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"testing"
	"time"

	"ad-event-processor/pkg/moderatorintel"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseFeedV1_holdout(t *testing.T) {
	t.Parallel()
	now := time.Date(2026, 8, 24, 12, 0, 0, 0, time.UTC)
	body := []byte(`{
  "format": "moderator_intel_v1",
  "source": "vendor.example/pack-1",
  "expires_at": "2026-08-25T00:00:00Z",
  "entries": [
    {"prefix": "1.2.3.4/32", "network": "meta"},
    {"prefix": "2001:db8::/32", "network": "google"}
  ]
}`)
	feed, err := moderatorintel.ParseFeedV1(body, now)
	require.NoError(t, err)
	require.Len(t, feed.Entries, 2)
	assert.Equal(t, "vendor.example/pack-1", feed.Source)
	assert.Equal(t, uint8(1), feed.Entries[0].Network)
}

func TestVerifySignature_roundTrip(t *testing.T) {
	t.Parallel()
	secret := []byte("test-secret")
	body := []byte(`{"format":"moderator_intel_v1"}`)
	mac := hmac.New(sha256.New, secret)
	_, _ = mac.Write(body)
	sig := base64.RawURLEncoding.EncodeToString(mac.Sum(nil))
	assert.True(t, moderatorintel.VerifySignature(secret, body, sig))
	assert.False(t, moderatorintel.VerifySignature(secret, body, "bad"))
}

func TestParseFeedV1_rejectsExpired(t *testing.T) {
	t.Parallel()
	body := []byte(`{
  "format": "moderator_intel_v1",
  "source": "vendor",
  "expires_at": "2020-01-01T00:00:00Z",
  "entries": [{"prefix": "1.1.1.1/32", "network": "meta"}]
}`)
	_, err := moderatorintel.ParseFeedV1(body, time.Now().UTC())
	assert.Error(t, err)
}
