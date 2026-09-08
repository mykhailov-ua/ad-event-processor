package domains

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"ad-event-processor/internal/config"
	"ad-event-processor/internal/testutil"
	"ad-event-processor/pkg/domainhealth"
	"ad-event-processor/pkg/platformconfig"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/stretchr/testify/require"
)

type mockWildcardIssuer struct {
	result wildcardIssueResult
	err    error
}

func (m *mockWildcardIssuer) Issue(ctx context.Context, cf DomainCloudflareClient, req wildcardIssueRequest) (wildcardIssueResult, error) {
	if m.err != nil {
		return wildcardIssueResult{}, m.err
	}
	return m.result, nil
}

type wildcardTestHost struct {
	pool *pgxpool.Pool
	cfg  *config.Config
	cf   CloudflareAPI
}

func (h wildcardTestHost) Pool() *pgxpool.Pool    { return h.pool }
func (h wildcardTestHost) Config() *config.Config { return h.cfg }
func (h wildcardTestHost) GetPlatformConfig(context.Context) (platformconfig.Config, bool, error) {
	return platformconfig.Config{}, false, nil
}
func (h wildcardTestHost) ReputationChecker() *domainhealth.ReputationChecker { return nil }
func (h wildcardTestHost) CloudflareClient() DomainCloudflareClient           { return h.cf }
func (h wildcardTestHost) StartBackgroundWorker(func())                       {}

func TestHostnameMatchesWildcardZone(t *testing.T) {
	t.Parallel()
	require.True(t, hostnameMatchesWildcardZone("a.trk.example.com", "trk.example.com", false))
	require.False(t, hostnameMatchesWildcardZone("trk.example.com", "trk.example.com", false))
	require.True(t, hostnameMatchesWildcardZone("trk.example.com", "trk.example.com", true))
	require.False(t, hostnameMatchesWildcardZone("evil.com", "trk.example.com", true))
}

func TestNormalizeZoneName(t *testing.T) {
	t.Parallel()
	require.Equal(t, "trk.example.com", normalizeZoneName("*.trk.example.com"))
}

func TestSetupWildcardSSL_mockIssuer(t *testing.T) {
	if testing.Short() {
		t.Skip("integration: postgres testcontainers required")
	}
	cfg := testutil.DefaultPostgresConfig()
	cfg.MigrationDirs = []string{testutil.AdsMigrationsDir(), testutil.BillingMigrationsDir()}
	pool, cleanup := testutil.SetupPostgres(t, cfg)
	defer cleanup()

	notAfter := time.Now().UTC().Add(90 * 24 * time.Hour)
	prevIssuer := defaultWildcardIssuer
	defaultWildcardIssuer = &mockWildcardIssuer{
		result: wildcardIssueResult{
			CertPEM:  []byte("-----BEGIN CERTIFICATE-----\ntest\n-----END CERTIFICATE-----"),
			KeyPEM:   []byte("-----BEGIN PRIVATE KEY-----\ntest\n-----END PRIVATE KEY-----"),
			NotAfter: notAfter,
		},
	}
	defer func() { defaultWildcardIssuer = prevIssuer }()

	appCfg := &config.Config{}
	appCfg.Management.DomainSSLAcmeEmail = "ops@example.com"
	appCfg.TokenSymmetricKey = "01234567890123456789012345678901"

	host := wildcardTestHost{
		pool: pool,
		cfg:  appCfg,
		cf:   NewCloudflareClient("token", ""),
	}
	dh := NewDomainHealth(host)

	resp, err := dh.SetupWildcardSSL(context.Background(), WildcardSSLRequest{
		CloudflareZoneID: "zone-abc",
		ZoneName:         "trk.example.com",
		IncludeApex:      true,
	})
	require.NoError(t, err)
	require.Equal(t, "valid", resp.AcmeState)
	require.Equal(t, "*.trk.example.com", resp.WildcardHostname)

	allowed, err := dh.IsTLSAllowed(context.Background(), "click.trk.example.com")
	require.NoError(t, err)
	require.True(t, allowed)
	allowed, err = dh.IsTLSAllowed(context.Background(), "trk.example.com")
	require.NoError(t, err)
	require.True(t, allowed)
}

func TestCloudflareClient_UpsertTXTRecord(t *testing.T) {
	t.Parallel()
	var created bool
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == http.MethodGet && r.URL.Path == "/zones/zone-abc/dns_records":
			_ = json.NewEncoder(w).Encode(map[string]any{"success": true, "result": []any{}})
		case r.Method == http.MethodPost && r.URL.Path == "/zones/zone-abc/dns_records":
			created = true
			_ = json.NewEncoder(w).Encode(map[string]any{"success": true, "result": map[string]string{"id": "txt-1"}})
		default:
			t.Fatalf("unexpected %s %s", r.Method, r.URL.Path)
		}
	}))
	defer srv.Close()

	client := NewCloudflareClient("token", srv.URL)
	id, err := client.UpsertTXTRecord(context.Background(), "zone-abc", "_acme-challenge.trk.example.com", "challenge-token")
	require.NoError(t, err)
	require.Equal(t, "txt-1", id)
	require.True(t, created)
}
