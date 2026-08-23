package ingestion

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/bidshard/ad-event-processor/internal/config"
	"github.com/stretchr/testify/require"
)

func TestTLSFingerprint_FeedRefreshFailClosed_RetainsSnapshot(t *testing.T) {
	dir := t.TempDir()
	blockPath := filepath.Join(dir, "ja3_blocklist.txt")
	allowPath := filepath.Join(dir, "ja3_allowlist.txt")
	require.NoError(t, os.WriteFile(blockPath, []byte("ja3:771,4865\n"), 0o644))
	require.NoError(t, os.WriteFile(allowPath, []byte("ja3:771,4865-4866,0-23,29-23-24,0\n"), 0o644))

	table := NewTLSFingerprintTable()
	cfg := &config.Config{
		TLSFingerprintL1Enabled:   true,
		TLSFingerprintFeedDir:     dir,
		TLSFingerprintFeedRefresh: time.Hour,
	}
	loader := NewTLSFingerprintFeedLoader(cfg, table)
	require.NotNil(t, loader)

	loader.refreshOnce()
	require.True(t, table.Ready())
	ja3Before, _, genBefore, ok := table.SnapshotSize()
	require.True(t, ok)
	require.Equal(t, 1, ja3Before)
	ja3AllowBefore, _, ok := table.AllowlistSize()
	require.True(t, ok)
	require.Equal(t, 1, ja3AllowBefore)

	require.NoError(t, os.Remove(blockPath))
	loader.refreshOnce()
	ja3After, _, genAfter, ok := table.SnapshotSize()
	require.True(t, ok)
	require.Equal(t, ja3Before, ja3After)
	require.Equal(t, genBefore, genAfter)
	ja3AllowAfter, _, ok := table.AllowlistSize()
	require.True(t, ok)
	require.Equal(t, ja3AllowBefore, ja3AllowAfter)
}

func TestTLSFingerprint_FeedLoaderLoadsAllowlist(t *testing.T) {
	dir := t.TempDir()
	blockPath := filepath.Join(dir, "ja3_blocklist.txt")
	allowPath := filepath.Join(dir, "ja3_allowlist.txt")
	ja3 := "771,4865-4866,0-23,29-23-24,0"
	require.NoError(t, os.WriteFile(blockPath, []byte("ja3:"+ja3+"\n"), 0o644))
	require.NoError(t, os.WriteFile(allowPath, []byte("ja3:"+ja3+"\n"), 0o644))

	table := NewTLSFingerprintTable()
	cfg := &config.Config{
		TLSFingerprintL1Enabled:   true,
		TLSFingerprintFeedDir:     dir,
		TLSFingerprintFeedRefresh: time.Hour,
	}
	loader := NewTLSFingerprintFeedLoader(cfg, table)
	require.NotNil(t, loader)
	loader.refreshOnce()

	require.True(t, table.Ready())
	ja3Allow, _, ok := table.AllowlistSize()
	require.True(t, ok)
	require.Equal(t, 1, ja3Allow)
	require.True(t, table.MatchJA3Allowed([]byte(ja3)))
	require.False(t, table.shouldBlockJA3([]byte(ja3)))
}
