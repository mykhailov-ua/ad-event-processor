package controlplane

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"ad-event-processor/internal/config"
	"ad-event-processor/internal/platformadmin"
	"ad-event-processor/internal/testutil"

	"github.com/stretchr/testify/require"
)

type bulkFakeCloudflare struct {
	calls int
}

func (m *bulkFakeCloudflare) ListZones(ctx context.Context) ([]platformadmin.CloudflareZone, error) {
	return nil, nil
}

func (m *bulkFakeCloudflare) CreateDNSRecord(ctx context.Context, zoneID, name, recordType, content string, proxied bool) (string, error) {
	m.calls++
	return "rec-bulk-" + name, nil
}

func (m *bulkFakeCloudflare) ZoneSSLStatus(ctx context.Context, zoneID string) (string, error) {
	return "pending", nil
}

func (m *bulkFakeCloudflare) UpsertTXTRecord(ctx context.Context, zoneID, name, content string) (string, error) {
	return "txt", nil
}

func (m *bulkFakeCloudflare) DeleteDNSRecord(ctx context.Context, zoneID, recordID string) error {
	return nil
}

func TestDomainBulkPark_returnsJobWithoutBlocking_holdout(t *testing.T) {
	if testing.Short() {
		t.Skip("integration: postgres testcontainers required")
	}
	cfg := testutil.DefaultPostgresConfig()
	cfg.MigrationDirs = []string{testutil.AdsMigrationsDir(), testutil.BillingMigrationsDir()}
	pool, cleanup := testutil.SetupPostgres(t, cfg)
	defer cleanup()

	ctx := context.Background()
	appCfg := &config.Config{}
	appCfg.Management.CloudflareDNSTarget = "203.0.113.10"
	cf := &bulkFakeCloudflare{}
	svc := &Service{pool: pool, cfg: appCfg}
	svc.SetCloudflareAPI(cf)

	h := &platformadmin.DomainHealthHTTPHandlers{Service: svc}
	mux := http.NewServeMux()
	h.Register(mux)

	body := `{"hostnames":["bulk-a.test","bulk-b.test"],"cloudflare_zone_id":"zone-bulk"}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/ops/domains/bulk", []byte(body))
	rec := httptest.NewRecorder()
	start := time.Now()
	mux.ServeHTTP(rec, req)
	elapsed := time.Since(start)
	require.Less(t, elapsed, 2*time.Second)
	require.Equal(t, http.StatusAccepted, rec.Code)

	var job platformadmin.DomainBulkJobStatus
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &job))
	require.NotEmpty(t, job.JobID)
	require.Equal(t, 2, job.Total)
	require.Equal(t, "pending", job.Status)

	deadline := time.Now().Add(30 * time.Second)
	for time.Now().Before(deadline) {
		statusReq := httptest.NewRequest(http.MethodGet, "/api/v1/ops/domains/jobs/"+job.JobID, nil)
		statusRec := httptest.NewRecorder()
		mux.ServeHTTP(statusRec, statusReq)
		require.Equal(t, http.StatusOK, statusRec.Code)
		var polled platformadmin.DomainBulkJobStatus
		require.NoError(t, json.Unmarshal(statusRec.Body.Bytes(), &polled))
		if polled.Status == "completed" || polled.Status == "failed" {
			require.Equal(t, 2, polled.Completed)
			require.GreaterOrEqual(t, cf.calls, 2)
			return
		}
		time.Sleep(200 * time.Millisecond)
	}
	t.Fatal("bulk job did not complete in time")
}
