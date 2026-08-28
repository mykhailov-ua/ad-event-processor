package fraud

import (
	"context"
	"fmt"
	"testing"
	"time"

	"ad-event-processor/internal/database"
	"ad-event-processor/pkg/piihash"

	"github.com/ClickHouse/clickhouse-go/v2/lib/driver"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func seedClickWithRTTSplit(t *testing.T, conn driver.Conn, ip string, deltaMS uint16, copies int) {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	campaignID := uuid.New()
	now := time.Now().UTC()
	h := testPIIHasher()
	rttSyn := uint16(40)
	ttfb := rttSyn + deltaMS
	for i := 0; i < copies; i++ {
		clickID := fmt.Sprintf("rtt-%s-%d-%d", ip, deltaMS, i)
		require.NoError(t, conn.Exec(ctx, `
			INSERT INTO ad_event_processor.clicks
			(click_id, campaign_id, ip_hash, ua_hash, pii_salt_version, tls_hash, payload,
			 rtt_syn_ms, ttfb_app_ms, rtt_split_delta_ms, created_at)
			VALUES (?, ?, ?, ?, ?, '', '', ?, ?, ?, ?)`,
			clickID, campaignID, piihash.FixedString16(h.HashIP(ip)), piihash.FixedString16(h.HashUA("ua-rtt")), h.Version(),
			rttSyn, ttfb, deltaMS, now,
		))
	}
}

func TestRTTSplitTunnelRule_holdoutHighDeltaLowVariance(t *testing.T) {
	if testing.Short() {
		t.Skip("integration: run make test-integration (Docker testcontainers)")
	}

	conn, cleanupCH := setupClickHouseTest(t)
	defer cleanupCH()

	ctx := context.Background()
	ip := "203.0.113.77"
	seedClickWithRTTSplit(t, conn, ip, 400, 6)

	rule := &rttSplitTunnelRule{
		clickhouseQuery: database.NewClickHouseQuery(conn, database.ClickHouseQueryConfig{}),
		cfg: AnalyzerConfig{
			Window:                time.Hour,
			RTTSplitTunnelEnabled: true,
			RTTSplitMinDeltaMS:    150,
			RTTSplitMaxVariance:   2500,
			RTTSplitMinSamples:    5,
		},
	}

	candidates, err := rule.Find(ctx)
	require.NoError(t, err)
	require.Len(t, candidates, 1)
	assert.Equal(t, "ivt_rtt_split_tunnel", candidates[0].Reason)
}

func TestRTTSplitTunnelRule_holdoutDisabledFailOpen(t *testing.T) {
	rule := &rttSplitTunnelRule{
		cfg: AnalyzerConfig{RTTSplitTunnelEnabled: false},
	}
	out, err := rule.Find(context.Background())
	require.NoError(t, err)
	assert.Nil(t, out)
}
