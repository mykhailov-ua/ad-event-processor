package controlplane_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"ad-event-processor/internal/controlplane"
	"github.com/stretchr/testify/require"
)

type stubDomainHealthService struct {
	allowed map[string]bool
}

func (s *stubDomainHealthService) ListDomainHealth(ctx context.Context) ([]controlplane.DomainHealthDTO, error) {
	return nil, nil
}

func (s *stubDomainHealthService) AddCustomDomain(ctx context.Context, hostname string) (controlplane.DomainHealthDTO, error) {
	return controlplane.DomainHealthDTO{}, nil
}

func (s *stubDomainHealthService) DeleteCustomDomain(ctx context.Context, hostname string) error {
	return nil
}

func (s *stubDomainHealthService) ProbeDomainNow(ctx context.Context, hostname string) (controlplane.DomainHealthDTO, error) {
	return controlplane.DomainHealthDTO{}, nil
}

func (s *stubDomainHealthService) SetupDomainSSL(ctx context.Context, hostname string) (controlplane.DomainSSLSetupResult, error) {
	return controlplane.DomainSSLSetupResult{}, nil
}

func (s *stubDomainHealthService) IsTLSAllowed(ctx context.Context, hostname string) (bool, error) {
	if s.allowed == nil {
		return false, nil
	}
	return s.allowed[hostname], nil
}

func (s *stubDomainHealthService) ParkDomain(ctx context.Context, req controlplane.ParkDomainRequest) (controlplane.ParkDomainResponse, error) {
	return controlplane.ParkDomainResponse{}, nil
}

func TestDomainHealthTLSAllowed_loopbackAllowed(t *testing.T) {
	h := &controlplane.DomainHealthHTTPHandlers{
		Service: &stubDomainHealthService{allowed: map[string]bool{
			"buyer.example.com": true,
		}},
		TLSAskAllowLocal: true,
	}
	mux := http.NewServeMux()
	h.Register(mux)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/ops/domains/buyer.example.com/tls-allowed", http.NoBody)
	req.RemoteAddr = "127.0.0.1:12345"
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)
	require.Equal(t, http.StatusOK, rec.Code)
	require.Contains(t, rec.Body.String(), `"allowed":true`)
}

func TestDomainHealthTLSAllowed_deniedUnknown(t *testing.T) {
	h := &controlplane.DomainHealthHTTPHandlers{
		Service:          &stubDomainHealthService{allowed: map[string]bool{}},
		TLSAskAllowLocal: true,
	}
	mux := http.NewServeMux()
	h.Register(mux)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/ops/domains/evil.example.com/tls-allowed", http.NoBody)
	req.RemoteAddr = "127.0.0.1:12345"
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)
	require.Equal(t, http.StatusForbidden, rec.Code)
}

func TestDomainHealthTLSAllowed_caddyAskQuery(t *testing.T) {
	h := &controlplane.DomainHealthHTTPHandlers{
		Service: &stubDomainHealthService{allowed: map[string]bool{
			"buyer.example.com": true,
		}},
		TLSAskToken:      "secret-ask",
		TLSAskAllowLocal: false,
	}
	mux := http.NewServeMux()
	h.Register(mux)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/ops/domains/tls-allowed?domain=buyer.example.com&token=secret-ask", http.NoBody)
	req.RemoteAddr = "10.0.0.5:12345"
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)
	require.Equal(t, http.StatusOK, rec.Code)
}

func TestDomainHealthTLSAllowed_tokenRequired(t *testing.T) {
	h := &controlplane.DomainHealthHTTPHandlers{
		Service: &stubDomainHealthService{allowed: map[string]bool{
			"buyer.example.com": true,
		}},
		TLSAskToken:      "secret-ask",
		TLSAskAllowLocal: false,
	}
	mux := http.NewServeMux()
	h.Register(mux)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/ops/domains/buyer.example.com/tls-allowed", http.NoBody)
	req.RemoteAddr = "10.0.0.5:12345"
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)
	require.Equal(t, http.StatusUnauthorized, rec.Code)

	req = httptest.NewRequest(http.MethodGet, "/api/v1/ops/domains/buyer.example.com/tls-allowed?token=secret-ask", http.NoBody)
	req.RemoteAddr = "10.0.0.5:12345"
	rec = httptest.NewRecorder()
	mux.ServeHTTP(rec, req)
	require.Equal(t, http.StatusOK, rec.Code)
}
