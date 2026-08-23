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
	require.NoError(t, os.WriteFile(blockPath, []byte("ja3:771,4865\n"), 0o644))

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

	require.NoError(t, os.Remove(blockPath))
	loader.refreshOnce()
	ja3After, _, genAfter, ok := table.SnapshotSize()
	require.True(t, ok)
	require.Equal(t, ja3Before, ja3After)
	require.Equal(t, genBefore, genAfter)
}
