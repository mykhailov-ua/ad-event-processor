package domains

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/require"
)

type sslScriptMissingService struct{}

func (sslScriptMissingService) ListDomainHealth(ctx context.Context) ([]DomainHealthDTO, error) {
	return nil, nil
}

func (sslScriptMissingService) AddCustomDomain(ctx context.Context, hostname string) (DomainHealthDTO, error) {
	return DomainHealthDTO{}, nil
}

func (sslScriptMissingService) DeleteCustomDomain(ctx context.Context, hostname string) error {
	return nil
}

func (sslScriptMissingService) ProbeDomainNow(ctx context.Context, hostname string) (DomainHealthDTO, error) {
	return DomainHealthDTO{}, nil
}

func (sslScriptMissingService) SetupDomainSSL(ctx context.Context, hostname string) (DomainSSLSetupResult, error) {
	return DomainSSLSetupResult{}, fmt.Errorf("ssl setup script not found: scripts/install/setup_domain_ssl.sh")
}

func (sslScriptMissingService) IsTLSAllowed(ctx context.Context, hostname string) (bool, error) {
	return false, nil
}

func (sslScriptMissingService) ParkDomain(ctx context.Context, req ParkDomainRequest) (ParkDomainResponse, error) {
	return ParkDomainResponse{}, nil
}

func (sslScriptMissingService) SetupWildcardSSL(ctx context.Context, req WildcardSSLRequest) (WildcardSSLResponse, error) {
	return WildcardSSLResponse{}, nil
}

func (sslScriptMissingService) ListCloudflareZones(ctx context.Context) ([]CloudflareZone, error) {
	return nil, nil
}

func (sslScriptMissingService) StartBulkParkProbe(ctx context.Context, req DomainBulkRequest) (DomainBulkJobStatus, error) {
	return DomainBulkJobStatus{}, nil
}

func (sslScriptMissingService) StartBulkSSL(ctx context.Context, req DomainBulkRequest) (DomainBulkJobStatus, error) {
	return DomainBulkJobStatus{}, nil
}

func (sslScriptMissingService) GetBulkJob(ctx context.Context, jobID string) (DomainBulkJobStatus, error) {
	return DomainBulkJobStatus{}, nil
}

func (sslScriptMissingService) BurnDomain(ctx context.Context, hostname string, req BurnDomainRequest) (BurnDomainResponse, error) {
	return BurnDomainResponse{}, nil
}

func TestDomainHealthSetupSSL_scriptMissing_holdout501(t *testing.T) {
	h := &DomainHealthHTTPHandlers{Service: sslScriptMissingService{}}
	req := httptest.NewRequest(http.MethodPost, "/api/v1/domains/track.example.com/ssl/setup", http.NoBody)
	req.SetPathValue("hostname", "track.example.com")
	w := httptest.NewRecorder()
	h.setupSSL(w, req)
	require.Equal(t, http.StatusNotImplemented, w.Code)
	require.Contains(t, w.Body.String(), "NOT_IMPLEMENTED")
	require.Contains(t, w.Body.String(), "ssl setup script not found")
}

type stubDomainHealthService struct {
	allowed map[string]bool
}

func (s *stubDomainHealthService) ListDomainHealth(ctx context.Context) ([]DomainHealthDTO, error) {
	return nil, nil
}

func (s *stubDomainHealthService) AddCustomDomain(ctx context.Context, hostname string) (DomainHealthDTO, error) {
	return DomainHealthDTO{}, nil
}

func (s *stubDomainHealthService) DeleteCustomDomain(ctx context.Context, hostname string) error {
	return nil
}

func (s *stubDomainHealthService) ProbeDomainNow(ctx context.Context, hostname string) (DomainHealthDTO, error) {
	return DomainHealthDTO{}, nil
}

func (s *stubDomainHealthService) SetupDomainSSL(ctx context.Context, hostname string) (DomainSSLSetupResult, error) {
	return DomainSSLSetupResult{}, nil
}

func (s *stubDomainHealthService) IsTLSAllowed(ctx context.Context, hostname string) (bool, error) {
	return s.allowed[hostname], nil
}

func (s *stubDomainHealthService) ParkDomain(ctx context.Context, req ParkDomainRequest) (ParkDomainResponse, error) {
	return ParkDomainResponse{}, nil
}

func (s *stubDomainHealthService) SetupWildcardSSL(ctx context.Context, req WildcardSSLRequest) (WildcardSSLResponse, error) {
	return WildcardSSLResponse{}, nil
}

func (s *stubDomainHealthService) ListCloudflareZones(ctx context.Context) ([]CloudflareZone, error) {
	return nil, nil
}

func (s *stubDomainHealthService) StartBulkParkProbe(ctx context.Context, req DomainBulkRequest) (DomainBulkJobStatus, error) {
	return DomainBulkJobStatus{}, nil
}

func (s *stubDomainHealthService) StartBulkSSL(ctx context.Context, req DomainBulkRequest) (DomainBulkJobStatus, error) {
	return DomainBulkJobStatus{}, nil
}

func (s *stubDomainHealthService) GetBulkJob(ctx context.Context, jobID string) (DomainBulkJobStatus, error) {
	return DomainBulkJobStatus{}, nil
}

func (s *stubDomainHealthService) BurnDomain(ctx context.Context, hostname string, req BurnDomainRequest) (BurnDomainResponse, error) {
	return BurnDomainResponse{}, nil
}

func TestDomainHealthTLSAllowed_loopbackAllowed(t *testing.T) {
	h := &DomainHealthHTTPHandlers{
		Service: &stubDomainHealthService{allowed: map[string]bool{
			"127.0.0.1": true,
		}},
		TLSAskAllowLocal: true,
	}
	req := httptest.NewRequest(http.MethodGet, "/api/v1/ops/domains/tls-allowed?domain=127.0.0.1", http.NoBody)
	req.RemoteAddr = "127.0.0.1:12345"
	w := httptest.NewRecorder()
	h.tlsAllowed(w, req)
	require.Equal(t, http.StatusOK, w.Code)
}

func TestDomainHealthTLSAllowed_deniedUnknown(t *testing.T) {
	h := &DomainHealthHTTPHandlers{
		Service:          &stubDomainHealthService{allowed: map[string]bool{}},
		TLSAskAllowLocal: true,
	}
	req := httptest.NewRequest(http.MethodGet, "/api/v1/ops/domains/tls-allowed?domain=unknown.example", http.NoBody)
	req.RemoteAddr = "127.0.0.1:12345"
	w := httptest.NewRecorder()
	h.tlsAllowed(w, req)
	require.Equal(t, http.StatusForbidden, w.Code)
}

func TestDomainHealthTLSAllowed_caddyAskQuery(t *testing.T) {
	h := &DomainHealthHTTPHandlers{
		Service: &stubDomainHealthService{allowed: map[string]bool{
			"track.example.com": true,
		}},
		TLSAskAllowLocal: true,
	}
	req := httptest.NewRequest(http.MethodGet, "/api/v1/ops/domains/tls-allowed?domain=track.example.com", http.NoBody)
	req.RemoteAddr = "127.0.0.1:12345"
	w := httptest.NewRecorder()
	h.tlsAllowed(w, req)
	require.Equal(t, http.StatusOK, w.Code)
}

func TestDomainHealthTLSAllowed_tokenRequired(t *testing.T) {
	h := &DomainHealthHTTPHandlers{
		Service: &stubDomainHealthService{allowed: map[string]bool{
			"track.example.com": true,
		}},
		TLSAskToken: "secret",
	}
	req := httptest.NewRequest(http.MethodGet, "/api/v1/ops/domains/tls-allowed?domain=track.example.com", http.NoBody)
	w := httptest.NewRecorder()
	h.tlsAllowed(w, req)
	require.Equal(t, http.StatusUnauthorized, w.Code)
}
