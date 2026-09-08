package domains

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/require"
)

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
