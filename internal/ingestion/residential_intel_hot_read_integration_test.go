package ingestion

import (
	"context"
	"encoding/json"
	"net/netip"
	"os"
	"path/filepath"
	"testing"

	"ad-event-processor/internal/config"
	"ad-event-processor/internal/domain"
	"ad-event-processor/internal/metrics"

	"github.com/google/uuid"
	"github.com/prometheus/client_golang/prometheus/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestResidentialIntelHotRead_holdoutIntelMatchWithoutRing(t *testing.T) {
	before := testutil.ToFloat64(metrics.ResidentialIntelHotMatchTotal)
	table := NewResidentialIntelTable()
	table.publishPrefixes(mustParsePrefix(t, "198.51.100.55/32"), 1)

	f := NewResidentialProxyFilter(nil)
	f.SetIntelTable(table, true)

	evt := domain.EventPool.Get().(*domain.Event)
	defer domain.EventPool.Put(evt)
	evt.Reset()
	acc := attachFraudAccumulator(evt)
	defer releaseFraudAccumulator(evt, acc)
	evt.CampaignID = uuid.New()
	evt.Type = "click"
	evt.IP = "198.51.100.55"

	require.NoError(t, f.Check(context.Background(), evt))
	assert.True(t, acc.has(FraudReasonResidentialProxy))
	assert.Equal(t, before+1, testutil.ToFloat64(metrics.ResidentialIntelHotMatchTotal))
}

func TestResidentialIntelHotRead_holdoutUninitializedFailOpen(t *testing.T) {
	table := NewResidentialIntelTable()
	f := NewResidentialProxyFilter(nil)
	f.SetIntelTable(table, true)

	evt := domain.EventPool.Get().(*domain.Event)
	defer domain.EventPool.Put(evt)
	evt.Reset()
	acc := attachFraudAccumulator(evt)
	defer releaseFraudAccumulator(evt, acc)
	evt.IP = "198.51.100.55"

	require.NoError(t, f.Check(context.Background(), evt))
	assert.False(t, acc.has(FraudReasonResidentialProxy))
}

func TestResidentialIntelFeedLoader_integrationRedisSnapshot(t *testing.T) {
	if testing.Short() {
		t.Skip("integration: residential intel hot read redis snapshot (run make test-integration)")
	}

	redisClient, cleanup := setupTestRedis(t)
	defer cleanup()

	ctx := context.Background()
	ip := "203.0.113.44"
	payload, err := json.Marshal(residentialIntelRedisEntry{ResidentialProxy: true, VPN: true, Proxy: true})
	require.NoError(t, err)
	require.NoError(t, redisClient.Set(ctx, residentialIntelRedisPrefix+ip, payload, 0).Err())

	table := NewResidentialIntelTable()
	cfg := &config.Config{ResidentialIntelHotReadEnabled: true}
	loader := NewResidentialIntelFeedLoader(cfg, table, redisClient)
	require.NotNil(t, loader)
	loader.reloadOnce(ctx)

	require.True(t, table.Ready())
	require.True(t, table.MatchIP(ip))
	require.False(t, table.MatchIP("203.0.113.45"))
}

func TestResidentialIntelFeedLoader_integrationFeedFile(t *testing.T) {
	if testing.Short() {
		t.Skip("integration: residential intel hot read feed file (run make test-integration)")
	}

	dir := t.TempDir()
	path := filepath.Join(dir, residentialIntelFeedFileName)
	require.NoError(t, os.WriteFile(path, []byte("203.0.113.45/32 0 vpn\n"), 0o644))

	table := NewResidentialIntelTable()
	cfg := &config.Config{
		ResidentialIntelHotReadEnabled: true,
		ResidentialIntelFeedDir:        dir,
	}
	loader := NewResidentialIntelFeedLoader(cfg, table, nil)
	require.NotNil(t, loader)
	loader.reloadOnce(context.Background())

	require.True(t, table.MatchIP("203.0.113.45"))
}

func mustParsePrefix(t *testing.T, s string) []netip.Prefix {
	t.Helper()
	p, err := netip.ParsePrefix(s)
	require.NoError(t, err)
	return []netip.Prefix{p}
}
