package controlplane

import (
	"context"
	"testing"

	"github.com/bidshard/ad-event-processor/internal/config"
	"github.com/bidshard/ad-event-processor/internal/controlplane/adminapi"
	"github.com/bidshard/ad-event-processor/internal/testutil"
	"github.com/stretchr/testify/require"
)

type mockCloudflareAPI struct {
	recordID  string
	sslStatus string
}

func (m *mockCloudflareAPI) ListZones(ctx context.Context) ([]CloudflareZone, error) {
	return nil, nil
}

func (m *mockCloudflareAPI) CreateDNSRecord(ctx context.Context, zoneID, name, recordType, content string, proxied bool) (string, error) {
	return m.recordID, nil
}

func (m *mockCloudflareAPI) ZoneSSLStatus(ctx context.Context, zoneID string) (string, error) {
	if m.sslStatus == "" {
		return "pending", nil
	}
	return m.sslStatus, nil
}

func TestDomainPark_createsPoolDomainAndHealthRow(t *testing.T) {
	cfg := testutil.DefaultPostgresConfig()
	cfg.MigrationDirs = []string{testutil.AdsMigrationsDir(), testutil.BillingMigrationsDir()}
	pool, cleanup := testutil.SetupPostgres(t, cfg)
	defer cleanup()

	ctx := context.Background()
	appCfg := &config.Config{}
	appCfg.Management.CloudflareDNSTarget = "203.0.113.10"
	svc := &Service{
		pool: pool,
		cfg:  appCfg,
	}
	svc.SetCloudflareAPI(&mockCloudflareAPI{recordID: "rec-456", sslStatus: "full"})

	resp, err := svc.ParkDomain(ctx, adminapi.ParkDomainRequest{
		Domain:           "track.buyer.test",
		CloudflareZoneID: "zone-abc",
	})
	require.NoError(t, err)
	require.True(t, resp.Success)
	require.Equal(t, "rec-456", resp.DNSRecordID)
	require.Equal(t, "full", resp.SSLStatus)

	var status string
	err = pool.QueryRow(ctx, `SELECT status FROM domain_pool_domains WHERE hostname = $1`, "track.buyer.test").Scan(&status)
	require.NoError(t, err)
	require.Equal(t, "pending", status)

	allowed, err := svc.IsTLSAllowed(ctx, "track.buyer.test")
	require.NoError(t, err)
	require.True(t, allowed)
}
