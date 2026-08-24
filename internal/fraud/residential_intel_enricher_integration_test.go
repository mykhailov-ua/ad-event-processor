package fraud

import (
	"context"
	"testing"
	"time"

	"ad-event-processor/internal/database"
	"ad-event-processor/internal/edge"
	"ad-event-processor/pkg/piihash"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestResidentialIntelEnricher_integration_clickhouseCache(t *testing.T) {
	if testing.Short() {
		t.Skip("integration: residential intel enricher writes CH cache + Redis TTL")
	}

	conn, cleanupCH := setupClickHouseTest(t)
	defer cleanupCH()

	redisClient, cleanupRedis := database.SetupTestRedis(t)
	defer cleanupRedis()

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	ip := "203.0.113.91"
	require.NoError(t, edge.Record(ctx, redisClient, edge.Entry{IP: ip, TCPHash: 0x123, SeenAt: time.Now().UTC()}))

	enricher := NewResidentialIntelEnricher(ResidentialIntelEnricherConfig{
		Provider: &StubResidentialIntelProvider{
			Results: map[string]ResidentialIntelResult{
				ip: {ResidentialProxy: true, VPN: true, Proxy: true},
			},
		},
		Cache:       NewResidentialIntelCache(redisClient, time.Hour),
		CHWrite:     conn,
		RedisClient: redisClient,
		FeedDir:     t.TempDir(),
		RecentLim:   16,
		BatchLim:    16,
		ProviderID:  "stub",
	})

	stats, err := enricher.Run(ctx)
	require.NoError(t, err)
	assert.Equal(t, 1, stats.LookedUp)
	assert.Equal(t, 1, stats.FeedAppended)

	hasher := piihash.TestHasher()
	var residential uint8
	err = conn.QueryRow(ctx,
		"SELECT residential_proxy FROM ad_event_processor.residential_intel_cache WHERE ip_hash = ? LIMIT 1",
		piihash.FixedString16(hasher.HashIP(ip)),
	).Scan(&residential)
	require.NoError(t, err)
	assert.Equal(t, uint8(1), residential)
}

func TestResidentialIntelEnricher_integration_redisCacheAndFeed(t *testing.T) {
	if testing.Short() {
		t.Skip("integration: residential intel enricher Redis cache + proxy/VPN feed append")
	}

	redisClient, cleanupRedis := database.SetupTestRedis(t)
	defer cleanupRedis()

	ctx := context.Background()
	ip := "203.0.113.88"
	require.NoError(t, edge.Record(ctx, redisClient, edge.Entry{IP: ip, TCPHash: 0xabc, SeenAt: time.Now().UTC()}))

	feedDir := t.TempDir()
	enricher := NewResidentialIntelEnricher(ResidentialIntelEnricherConfig{
		Provider: &StubResidentialIntelProvider{
			Results: map[string]ResidentialIntelResult{
				ip: {ResidentialProxy: true, VPN: true},
			},
		},
		Cache:       NewResidentialIntelCache(redisClient, time.Hour),
		RedisClient: redisClient,
		FeedDir:     feedDir,
		RecentLim:   8,
		BatchLim:    8,
	})

	stats, err := enricher.Run(ctx)
	require.NoError(t, err)
	assert.Equal(t, 1, stats.LookedUp)
	assert.Equal(t, 1, stats.FeedAppended)

	cached, hit, err := enricher.cache.Get(ctx, ip)
	require.NoError(t, err)
	require.True(t, hit)
	assert.True(t, cached.ResidentialProxy)

	stats2, err := enricher.Run(ctx)
	require.NoError(t, err)
	assert.Equal(t, 0, stats2.LookedUp)
	assert.Equal(t, 1, stats2.CacheHits)
	assert.Equal(t, 0, stats2.FeedAppended)
}
