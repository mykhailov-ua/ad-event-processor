package ingestion

import (
	"net/http"
	"testing"

	"ad-event-processor/internal/config"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

func TestDomainPoolTable_FallbackHost(t *testing.T) {
	t.Parallel()
	table := NewDomainPoolTable()
	poolID := uuid.MustParse("11111111-1111-4111-8111-111111111111")
	table.Publish(buildDomainPoolSnapshotFromRows([]domainPoolSyncRow{
		{poolID: poolID, hostname: "banned.example", status: "banned"},
		{poolID: poolID, hostname: "active.example", status: "active"},
	}, 1))

	fallback, ok := table.fallbackHost([]byte("banned.example"))
	require.True(t, ok)
	require.Equal(t, "active.example", string(fallback))

	_, ok = table.fallbackHost([]byte("active.example"))
	require.False(t, ok)
}

func TestClickRedirect_DomainRotation(t *testing.T) {
	runDomainBanRotationClickTest(t)
}

func TestDomainHealth_BanTriggersRotation(t *testing.T) {
	runDomainBanRotationClickTest(t)
}

func runDomainBanRotationClickTest(t *testing.T) {
	table := NewDomainPoolTable()
	poolID := uuid.MustParse("22222222-2222-4222-8222-222222222222")
	table.Publish(buildDomainPoolSnapshotFromRows([]domainPoolSyncRow{
		{poolID: poolID, hostname: "banned-track.test", status: "banned"},
		{poolID: poolID, hostname: "active-track.test", status: "active"},
	}, 1))

	cfg := &config.Config{MaxRequestBodySize: 1 << 20}
	h := NewAdsPacketHandler(cfg, &mockRegistry{}, nil, nil, nil, NewJumpHashSharder(1), "fraud-stream", nil)
	h.ConfigureDomainPool(table)

	path := "/click?campaign_id=550e8400-e29b-41d4-a716-446655440000&type=click"
	_, conn := ServeGnetHarness(h, BuildGnetHTTP("GET", path, map[string]string{
		"Host":           "banned-track.test",
		"Connection":     "keep-alive",
		"Content-Length": "0",
	}, nil))
	require.Equal(t, http.StatusFound, ParseGnetHTTPStatus(conn.Written()))
	resp := string(conn.Written())
	require.Contains(t, resp, "Location: https://active-track.test/click?")
	require.Contains(t, resp, "campaign_id=550e8400-e29b-41d4-a716-446655440000")
	t.Logf("fault_proof fault=domain_health_ban_rotate harness=unit_rcu_snapshot drop_assertion=location_host_changed")
}

func TestClickRedirect_DomainRotation_DMR(t *testing.T) {
	table := NewDomainPoolTable()
	poolID := uuid.MustParse("22222222-2222-4222-8222-222222222222")
	table.Publish(buildDomainPoolSnapshotFromRows([]domainPoolSyncRow{
		{poolID: poolID, hostname: "banned-track.test", status: "banned"},
		{poolID: poolID, hostname: "active-track.test", status: "active"},
	}, 1))

	cfg := &config.Config{MaxRequestBodySize: 1 << 20}
	h := NewAdsPacketHandler(cfg, &mockRegistry{}, nil, nil, nil, NewJumpHashSharder(1), "fraud-stream", nil)
	h.ConfigureDomainPool(table)

	path := "/click?campaign_id=550e8400-e29b-41d4-a716-446655440000&type=click&dmr=1"
	_, conn := ServeGnetHarness(h, BuildGnetHTTP("GET", path, map[string]string{
		"Host":           "banned-track.test",
		"Connection":     "keep-alive",
		"Content-Length": "0",
	}, nil))
	require.Equal(t, http.StatusOK, ParseGnetHTTPStatus(conn.Written()))
	resp := string(conn.Written())
	require.Contains(t, resp, "active-track.test/click?")
	require.Contains(t, resp, "campaign_id=550e8400-e29b-41d4-a716-446655440000")
	require.NotContains(t, resp, "302 Found")
}

func TestBuildTrackingDomainRotation(t *testing.T) {
	t.Parallel()
	loc := buildTrackingDomainRotation(nil, []byte("https"), []byte("next.example"), []byte("/click?x=1"))
	require.Equal(t, "https://next.example/click?x=1", string(loc))
}
