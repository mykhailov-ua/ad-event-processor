package netintel

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"os"
	"path/filepath"
	"testing"
	"time"

	"ad-event-processor/internal/config"

	"github.com/stretchr/testify/require"
)

func TestModeratorIntel_FeedRefreshFailClosed_RetainsSnapshot(t *testing.T) {
	dir := t.TempDir()
	secret := []byte("feed-secret")
	body := []byte(`{
  "format": "moderator_intel_v1",
  "source": "vendor/pack-1",
  "expires_at": "2099-01-01T00:00:00Z",
  "entries": [{"prefix": "203.0.113.0/24", "network": "meta"}]
}`)
	writeModeratorIntelFeed(t, dir, body, secret)

	cfg := &config.Config{
		ModeratorIntelEnabled:     true,
		ModeratorIntelFeedDir:     dir,
		ModeratorIntelFeedRefresh: time.Hour,
		ModeratorIntelFeedSecret:  string(secret),
	}
	table := NewModeratorIPTable()
	loader := NewModeratorIntelFeedLoader(cfg, table)
	require.NotNil(t, loader)
	loader.refreshOnce(context.Background())
	require.True(t, table.Ready())
	ok, netID := table.MatchIP("203.0.113.55")
	require.True(t, ok)
	require.Equal(t, uint8(1), netID)

	require.NoError(t, os.WriteFile(filepath.Join(dir, moderatorIntelFeedFile), []byte(`{bad json`), 0o644))
	loader.refreshOnce(context.Background())
	require.True(t, table.Ready())
	ok, _ = table.MatchIP("203.0.113.55")
	require.True(t, ok)
	t.Log("fault_proof fault=feed_corrupt_retain_snapshot harness=moderator_intel_lpm")
}

func writeModeratorIntelFeed(t *testing.T, dir string, body, secret []byte) {
	t.Helper()
	require.NoError(t, os.WriteFile(filepath.Join(dir, moderatorIntelFeedFile), body, 0o644))
	mac := hmac.New(sha256.New, secret)
	_, _ = mac.Write(body)
	sig := base64.RawURLEncoding.EncodeToString(mac.Sum(nil))
	require.NoError(t, os.WriteFile(filepath.Join(dir, moderatorIntelSigFile), []byte(sig), 0o644))
}
