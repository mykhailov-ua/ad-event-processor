package domains

import (
	"context"
	"fmt"
	"testing"
	"time"

	"ad-event-processor/internal/config"
	"ad-event-processor/pkg/domainhealth"
	"ad-event-processor/pkg/platformconfig"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/stretchr/testify/require"
)

func TestNormalizeBulkHostnames_csvAndDedup(t *testing.T) {
	t.Parallel()
	hosts, err := normalizeBulkHostnames(
		[]string{"a.example", "A.example"},
		"hostname\nb.example\nc.example,b.example",
	)
	require.NoError(t, err)
	require.Equal(t, []string{"a.example", "b.example", "c.example"}, hosts)
}

func TestNormalizeBulkHostnames_maxExceeded(t *testing.T) {
	t.Parallel()
	raw := make([]string, domainBulkMaxHostnames+1)
	for i := range raw {
		raw[i] = fmt.Sprintf("host-%d.example", i)
	}
	_, err := normalizeBulkHostnames(raw, "")
	require.Error(t, err)
}

func TestNormalizeBulkHostnames_empty(t *testing.T) {
	t.Parallel()
	_, err := normalizeBulkHostnames(nil, "  \n")
	require.Error(t, err)
}

type bulkEnqueueHost struct{}

func (bulkEnqueueHost) Pool() *pgxpool.Pool    { return nil }
func (bulkEnqueueHost) Config() *config.Config { return nil }
func (bulkEnqueueHost) GetPlatformConfig(context.Context) (platformconfig.Config, bool, error) {
	return platformconfig.Config{}, false, nil
}
func (bulkEnqueueHost) ReputationChecker() *domainhealth.ReputationChecker { return nil }
func (bulkEnqueueHost) CloudflareClient() DomainCloudflareClient           { return nil }
func (bulkEnqueueHost) StartBackgroundWorker(func())                       {}

func TestStartBulkParkProbe_100Hosts_returnsImmediately_holdout(t *testing.T) {
	hosts := make([]string, 100)
	for i := range hosts {
		hosts[i] = fmt.Sprintf("bulk-%03d.test", i)
	}
	dh := NewDomainHealth(bulkEnqueueHost{})
	start := time.Now()
	status, err := dh.StartBulkParkProbe(context.Background(), DomainBulkRequest{
		Hostnames:        hosts,
		CloudflareZoneID: "zone-bulk-100",
	})
	elapsed := time.Since(start)
	require.NoError(t, err)
	require.Less(t, elapsed, 2*time.Second, "bulk enqueue must not block on per-host work")
	require.Equal(t, 100, status.Total)
	require.Equal(t, "pending", status.Status)

	polled, err := dh.GetBulkJob(context.Background(), status.JobID)
	require.NoError(t, err)
	require.Equal(t, 100, polled.Total)
}
