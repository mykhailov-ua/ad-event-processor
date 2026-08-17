package adminapi_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/bidshard/ad-event-processor/internal/controlplane/adminapi"
	"github.com/stretchr/testify/require"
)

type stubDomainHealthService struct {
	allowed map[string]bool
}

func (s *stubDomainHealthService) ListDomainHealth(ctx context.Context) ([]adminapi.DomainHealthDTO, error) {
	return nil, nil
}
func (s *stubDomainHealthService) AddCustomDomain(ctx context.Context, hostname string) (adminapi.DomainHealthDTO, error) {
	return adminapi.DomainHealthDTO{}, nil
}
func (s *stubDomainHealthService) DeleteCustomDomain(ctx context.Context, hostname string) error {
	return nil
}
func (s *stubDomainHealthService) ProbeDomainNow(ctx context.Context, hostname string) (adminapi.DomainHealthDTO, error) {
	return adminapi.DomainHealthDTO{}, nil
}
func (s *stubDomainHealthService) SetupDomainSSL(ctx context.Context, hostname string) (adminapi.DomainSSLSetupResult, error) {
	return adminapi.DomainSSLSetupResult{}, nil
}
func (s *stubDomainHealthService) IsTLSAllowed(ctx context.Context, hostname string) (bool, error) {
	if s.allowed == nil {
		return false, nil
	}
	return s.allowed[hostname], nil
}
func (s *stubDomainHealthService) ParkDomain(ctx context.Context, req adminapi.ParkDomainRequest) (adminapi.ParkDomainResponse, error) {
	return adminapi.ParkDomainResponse{}, nil
}

func TestDomainHealthTLSAllowed_loopbackAllowed(t *testing.T) {
	h := &adminapi.DomainHealthHTTPHandlers{
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
	h := &adminapi.DomainHealthHTTPHandlers{
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
	h := &adminapi.DomainHealthHTTPHandlers{
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
	h := &adminapi.DomainHealthHTTPHandlers{
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
